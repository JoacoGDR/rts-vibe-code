package aibot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/joaquing/clone-supremacy/internal/aibot/heuristic"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// sessionThrottle is the minimum gap between two heuristic decisions for
// the same session. Stops the bot from firing every time a state ticks.
const sessionThrottle = 750 * time.Millisecond

// session drives one bot's WebSocket lifecycle for one match. It is
// owned by the runner and torn down via cancel().
type session struct {
	logger   *slog.Logger
	core     *CoreClient
	gateway  string
	matchID  string
	slot     string
	humanUID uuid.UUID
	botUID   uuid.UUID

	mu     sync.Mutex
	state  *wire.MatchState
	prng   *rand.Rand
	lastAt time.Time
	ws     *websocket.Conn
}

func newSession(logger *slog.Logger, core *CoreClient, gateway, matchID, slot string, humanUID, botUID uuid.UUID) *session {
	seed1 := uint64(time.Now().UnixNano())
	seed2 := uint64(humanUID.ID()) ^ uint64(botUID.ID())
	return &session{
		logger:   logger.With("match", matchID, "slot", slot, "bot", botUID.String()),
		core:     core,
		gateway:  gateway,
		matchID:  matchID,
		slot:     slot,
		humanUID: humanUID,
		botUID:   botUID,
		prng:     rand.New(rand.NewPCG(seed1, seed2)),
	}
}

// run is the per-session main loop. It logs in as a bot, dials the
// gateway, and pumps state envelopes through the heuristic until ctx is
// cancelled or the WebSocket closes.
func (s *session) run(ctx context.Context) error {
	if _, err := s.core.LoginAsBot(ctx, s.botUID); err != nil {
		return fmt.Errorf("login-as-bot: %w", err)
	}
	ticket, err := s.core.MintWSTicket(ctx)
	if err != nil {
		return fmt.Errorf("ws-ticket: %w", err)
	}
	dialURL, err := buildWSURL(s.gateway, ticket, s.matchID)
	if err != nil {
		return err
	}
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 5 * time.Second
	conn, _, err := dialer.DialContext(ctx, dialURL, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	s.mu.Lock()
	s.ws = conn
	s.mu.Unlock()
	defer func() { _ = conn.Close() }()

	hello := wire.ClientEnvelope{Type: wire.ClientHello, MatchID: s.matchID}
	if err := conn.WriteJSON(hello); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	stop := make(chan struct{})
	defer close(stop)
	go s.pingLoop(stop)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if !errIsClose(err) {
				s.logger.Warn("bot ws read failed", "err", err)
			}
			return err
		}
		s.handleEnvelope(raw)
	}
}

// pingLoop keeps the connection healthy independently of the read loop.
// A 20s cadence matches what the React client does so the gateway
// doesn't see bots as anomalously chatty.
func (s *session) pingLoop(stop chan struct{}) {
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			s.mu.Lock()
			ws := s.ws
			s.mu.Unlock()
			if ws == nil {
				return
			}
			_ = ws.WriteJSON(wire.ClientEnvelope{Type: wire.ClientPing})
		}
	}
}

// handleEnvelope dispatches one incoming server envelope. Only state
// triggers a heuristic decision today; events and chat are observed but
// not acted on (the heuristic re-derives intent purely from state).
func (s *session) handleEnvelope(raw []byte) {
	var env wire.ServerEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return
	}
	switch env.Type {
	case wire.ServerState:
		if env.State != nil {
			s.onState(env.State)
		}
	case wire.ServerError:
		s.logger.Debug("bot ws server error", "err", env.Error)
	}
}

// onState caches the latest state and runs the throttled decision pass.
func (s *session) onState(state *wire.MatchState) {
	s.mu.Lock()
	s.state = state
	if time.Since(s.lastAt) < sessionThrottle {
		s.mu.Unlock()
		return
	}
	s.lastAt = time.Now()
	prng := s.prng
	ws := s.ws
	s.mu.Unlock()
	if ws == nil {
		return
	}
	cmds := heuristic.Decide(state, s.slot, prng)
	for _, c := range cmds {
		env := wire.ClientEnvelope{
			Type:    wire.ClientCommand,
			MatchID: s.matchID,
			Command: &c,
		}
		if err := ws.WriteJSON(env); err != nil {
			s.logger.Warn("bot ws write failed", "err", err)
			return
		}
	}
}

// buildWSURL turns a gateway base URL (which may be ws://, wss://,
// http:// or https://) plus a ticket and match id into the full upgrade
// URL the bot session dials.
func buildWSURL(base, ticket, matchID string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "ws":
		u.Scheme = "ws"
	case "https", "wss":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/ws"
	}
	q := u.Query()
	q.Set("ticket", ticket)
	q.Set("match_id", matchID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// errIsClose reports whether the error is a "the other side closed the
// connection" type, which is normal at shutdown and should not be
// surfaced as a warning.
func errIsClose(err error) bool {
	if err == nil {
		return false
	}
	if websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure,
	) {
		return true
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}
