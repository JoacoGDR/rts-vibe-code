package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/adapter/redisrepo"
	"github.com/joaquing/clone-supremacy/internal/auth"
	"github.com/joaquing/clone-supremacy/internal/platform/metrics"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

const (
	writeTimeout      = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = 30 * time.Second
	maxMessageSize    = 1 << 14 // 16 KiB
	clientSendBacklog = 64
)

type hub struct {
	logger      *slog.Logger
	metrics     *metrics.Registry
	nc          *nats.Conn
	js          jetstream.JetStream
	rdb         *redis.Client
	tickets     *auth.TicketBroker
	upgrader    websocket.Upgrader
	rateLimit   int
	mu          sync.Mutex
	connections map[string]*conn
}

// hubConfig groups the dependencies a hub needs at construction so the
// signature stays under the line-length budget.
type hubConfig struct {
	Logger    *slog.Logger
	Metrics   *metrics.Registry
	NC        *nats.Conn
	JS        jetstream.JetStream
	RDB       *redis.Client
	Tickets   *auth.TicketBroker
	RateLimit int
}

func newHub(c hubConfig) *hub {
	return &hub{
		logger:    c.Logger,
		metrics:   c.Metrics,
		nc:        c.NC,
		js:        c.JS,
		rdb:       c.RDB,
		tickets:   c.Tickets,
		rateLimit: c.RateLimit,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(*http.Request) bool { return true }, // dev: tighten in Phase 7
		},
		connections: map[string]*conn{},
	}
}

type conn struct {
	id        string
	hub       *hub
	ws        *websocket.Conn
	user      auth.Ticket
	send      chan []byte
	matchID   string
	slot      string
	subs      []*nats.Subscription
	logger    *slog.Logger
	rate      *tokenBucket
	closed    chan struct{}
	closeOnce sync.Once
	chat      *chatState
}

func (h *hub) handleWS(w http.ResponseWriter, r *http.Request) {
	ticketID := r.URL.Query().Get("ticket")
	if ticketID == "" {
		http.Error(w, "ticket required", http.StatusUnauthorized)
		return
	}
	tk, err := h.tickets.Redeem(r.Context(), ticketID)
	if err != nil {
		http.Error(w, "invalid ticket", http.StatusUnauthorized)
		return
	}
	matchID := r.URL.Query().Get("match_id")

	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("ws upgrade failed", "err", err)
		return
	}
	ws.SetReadLimit(maxMessageSize)
	_ = ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	connID := uuid.NewString()
	c := &conn{
		id:      connID,
		hub:     h,
		ws:      ws,
		user:    tk,
		send:    make(chan []byte, clientSendBacklog),
		matchID: matchID,
		logger:  h.logger.With("conn_id", connID, "user", tk.UserID.String(), "match", matchID),
		rate:    newTokenBucket(h.rateLimit, time.Second),
		closed:  make(chan struct{}),
	}

	h.register(c)
	h.metrics.WSConnections.Inc()
	c.logger.Info("ws connected")

	if matchID != "" {
		if err := c.subscribeMatch(matchID); err != nil {
			c.logger.Warn("subscribe failed", "err", err)
			c.close()
			return
		}
	}

	go c.writeLoop()
	go c.readLoop() //nolint:gosec // the goroutine intentionally outlives the request; lifetime is bounded by c.closed.
}

// publishPresence fires a single "user opened the match" ping. The
// worker subscribes and stamps match_players.last_seen_at; absence of
// any ping for the configured threshold (default 72h) is what flips
// the slot to AI control.
//
// Skips when the connection has not been associated with a match/slot
// yet — the caller is expected to invoke this after subscribeMatch.
func (c *conn) publishPresence() {
	if c.matchID == "" || c.slot == "" {
		return
	}
	_ = c.hub.publisher().PublishPresence(context.Background(), natsbridge.PresencePayload{
		MatchID: c.matchID,
		UserID:  c.user.UserID.String(),
		Slot:    c.slot,
		At:      time.Now(),
	})
}

func (h *hub) register(c *conn) {
	h.mu.Lock()
	h.connections[c.id] = c
	h.mu.Unlock()
}

func (h *hub) unregister(c *conn) {
	h.mu.Lock()
	delete(h.connections, c.id)
	h.mu.Unlock()
	h.metrics.WSConnections.Dec()
}

func (c *conn) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		for _, s := range c.subs {
			_ = s.Unsubscribe()
		}
		_ = c.ws.Close()
		c.hub.unregister(c)
	})
}

// subscribeMatch attaches the connection to its slot's filtered state/event
// subjects. We always look up the slot fresh (in case the player joined
// after first connecting). If the slot cannot be resolved (engine has not
// started the match yet) we fall back to the public subjects so the lobby
// still gets unfiltered "match started" notifications.
func (c *conn) subscribeMatch(matchID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	slot, _ := redisrepo.LookupUserSlot(ctx, c.hub.rdb, matchID, c.user.UserID.String())
	c.slot = slot

	stateSubject := natsbridge.PublicStateSubject(matchID)
	eventSubject := natsbridge.PublicEventSubject(matchID)
	if slot != "" {
		stateSubject = natsbridge.SlotStateSubject(matchID, slot)
		eventSubject = natsbridge.SlotEventSubject(matchID, slot)
	}

	stateSub, err := c.hub.nc.Subscribe(stateSubject, c.stateFanOut)
	if err != nil {
		return err
	}
	eventSub, err := c.hub.nc.Subscribe(eventSubject, c.fanOut)
	if err != nil {
		_ = stateSub.Unsubscribe()
		return err
	}
	c.subs = append(c.subs, stateSub, eventSub)
	if err := c.subscribeChat(matchID); err != nil {
		return err
	}
	return nil
}

// stateFanOut forwards a state envelope to the WS while also peeking at
// it so the chat layer can refresh its coalition cache.
func (c *conn) stateFanOut(msg *nats.Msg) {
	c.maybeUpdateCoalitionFromState(msg.Data)
	c.fanOut(msg)
}

func (c *conn) fanOut(msg *nats.Msg) {
	select {
	case c.send <- msg.Data:
	default:
		c.logger.Warn("dropping message, client too slow")
	}
}

func (c *conn) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-c.closed:
			return
		case payload, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteMessage(websocket.TextMessage, payload); err != nil {
				c.logger.Debug("write fail", "err", err)
				c.close()
				return
			}
			c.hub.metrics.WSMessagesOut.Inc()
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		}
	}
}

func (c *conn) readLoop() {
	defer c.close()
	for {
		_, raw, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		c.hub.metrics.WSMessagesIn.Inc()
		var env wire.ClientEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			c.sendError("bad_json", err.Error())
			continue
		}
		switch env.Type {
		case wire.ClientHello:
			c.handleHello(env)
		case wire.ClientPing:
			c.send <- mustJSON(wire.ServerEnvelope{Type: wire.ServerPong, SentAt: time.Now()})
		case wire.ClientCommand:
			c.handleCommand(env)
		case wire.ClientChat:
			c.handleClientChat(env)
		case wire.ClientResync:
			// Forward to the engine via NATS so the runner re-publishes the
			// latest filtered state on the per-slot subject we are
			// subscribed to. The engine answers asynchronously.
			if c.matchID != "" {
				_ = c.hub.publisher().PublishResync(context.Background(), c.matchID)
			}
			c.send <- mustJSON(wire.ServerEnvelope{Type: wire.ServerHelloAck, SentAt: time.Now()})
		case wire.ClientGoodbye:
			return
		default:
			c.sendError("unknown_type", string(env.Type))
		}
	}
}

func (c *conn) handleHello(env wire.ClientEnvelope) {
	if env.WireVersion != 0 && env.WireVersion != wire.WireVersion {
		c.sendError("wire_version", "client wire version is incompatible with server")
		return
	}
	matchID := env.MatchID
	if matchID == "" {
		c.sendError("missing_match", "hello must include match_id")
		return
	}
	if c.matchID == "" {
		if err := c.subscribeMatch(matchID); err != nil {
			c.sendError("subscribe_failed", err.Error())
			return
		}
		c.matchID = matchID
	}
	c.publishPresence()
	c.send <- mustJSON(wire.ServerEnvelope{
		Type:    wire.ServerHelloAck,
		MatchID: matchID,
		SentAt:  time.Now(),
	})
}

func (c *conn) handleCommand(env wire.ClientEnvelope) {
	if env.Command == nil {
		c.sendError("missing_command", "command body required")
		return
	}
	if c.matchID == "" || env.MatchID != c.matchID {
		c.sendError("wrong_match", "command must target the connected match")
		return
	}
	if !c.rate.Allow() {
		c.sendError("rate_limited", "")
		return
	}

	payload := natsbridge.CommandPayload{
		MatchID:        c.matchID,
		UserID:         c.user.UserID.String(),
		Slot:           c.slot,
		IdempotencyKey: env.Command.Idempotency,
		Kind:           env.Command.Kind,
		UnitID:         env.Command.UnitID,
		From:           env.Command.From,
		To:             env.Command.To,
		IssuedAt:       env.Command.IssuedAt,
		Args:           env.Command.Args,
	}
	if err := c.hub.publisher().PublishCommand(context.Background(), payload); err != nil {
		c.sendError("publish_failed", err.Error())
		return
	}
}

func (c *conn) sendError(code, msg string) {
	c.send <- mustJSON(wire.ServerEnvelope{
		Type:   wire.ServerError,
		SentAt: time.Now(),
		Error:  &wire.ErrorPayload{Code: code, Message: msg},
	})
}

// publisher returns a fresh natsbridge.Publisher tied to the hub's NATS
// connections. Cheap to construct so we don't bother caching.
func (h *hub) publisher() *natsbridge.Publisher {
	return &natsbridge.Publisher{JS: h.js, NC: h.nc}
}

func (h *hub) shutdown() {
	h.mu.Lock()
	conns := make([]*conn, 0, len(h.connections))
	for _, c := range h.connections {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		c.close()
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error","error":{"code":"marshal_fail","message":"` + err.Error() + `"}}`)
	}
	return b
}
