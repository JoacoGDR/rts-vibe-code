package gateway

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// chatState carries the per-connection chat bookkeeping. Bundling it in
// its own struct keeps `conn` from growing further every release.
type chatState struct {
	mu            sync.Mutex
	coalitionLead string // empty until the first state envelope arrives
	worldSub      *nats.Subscription
	dmSub         *nats.Subscription
	coalSub       *nats.Subscription
	matchID       string
	userID        string
}

// subscribeChat wires the world and DM inbox subscriptions immediately,
// and registers the wildcard coalition subscription with a filter that
// uses the latest cached coalition leader. The leader is refreshed by
// [conn.maybeUpdateCoalitionFromState] every time a state envelope
// arrives.
func (c *conn) subscribeChat(matchID string) error {
	if c.chat == nil {
		c.chat = &chatState{matchID: matchID, userID: c.user.UserID.String()}
	}
	worldSub, err := c.hub.nc.Subscribe(natsbridge.ChatScopeSubject(worldScope(matchID)), c.fanOut)
	if err != nil {
		return err
	}
	dmSub, err := c.hub.nc.Subscribe(natsbridge.ChatScopeWildcardForUser(c.chat.userID), c.fanOut)
	if err != nil {
		_ = worldSub.Unsubscribe()
		return err
	}
	coalSub, err := c.hub.nc.Subscribe(coalitionWildcard(matchID), c.chatCoalitionFanOut)
	if err != nil {
		_ = worldSub.Unsubscribe()
		_ = dmSub.Unsubscribe()
		return err
	}
	c.chat.worldSub, c.chat.dmSub, c.chat.coalSub = worldSub, dmSub, coalSub
	c.subs = append(c.subs, worldSub, dmSub, coalSub)
	return nil
}

// chatCoalitionFanOut filters incoming coalition messages so only the
// ones addressed to this user's current coalition reach the WS.
func (c *conn) chatCoalitionFanOut(msg *nats.Msg) {
	c.chat.mu.Lock()
	leader := c.chat.coalitionLead
	c.chat.mu.Unlock()
	if leader == "" {
		return
	}
	expected := natsbridge.ChatScopeSubject(coalitionScope(c.chat.matchID, leader))
	if msg.Subject != expected {
		return
	}
	c.fanOut(msg)
}

// handleClientChat consumes a `chat` envelope from the WS, validates the
// scope locally and forwards the payload to core-api via the chat
// ingress NATS subject.
func (c *conn) handleClientChat(env wire.ClientEnvelope) {
	if c.matchID == "" || (env.MatchID != "" && env.MatchID != c.matchID) {
		c.sendError("wrong_match", "chat must target the connected match")
		return
	}
	if env.Chat == nil || env.Chat.Body == "" {
		c.sendError("missing_chat", "chat body required")
		return
	}
	if !c.rate.Allow() {
		c.sendError("rate_limited", "")
		return
	}
	scope, err := c.scopeFromOutbound(env.Chat)
	if err != nil {
		c.sendError("bad_scope", err.Error())
		return
	}
	payload := natsbridge.ChatIngressPayload{
		MatchID:    c.matchID,
		AuthorID:   c.user.UserID.String(),
		AuthorSlot: c.slot,
		Scope:      scope,
		Body:       env.Chat.Body,
	}
	if err := c.hub.publisher().PublishChatIngress(context.Background(), payload); err != nil {
		c.sendError("publish_failed", err.Error())
	}
}

// scopeFromOutbound translates the user-facing scope ("world",
// "coalition", "dm") into the canonical chatdom scope key. The
// transformation depends on the current coalition leader (for coalition)
// and on the addressed user (for DMs).
func (c *conn) scopeFromOutbound(out *wire.ChatOutbound) (string, error) {
	switch out.Scope {
	case "world":
		return worldScope(c.matchID), nil
	case "coalition":
		c.chat.mu.Lock()
		lead := c.chat.coalitionLead
		c.chat.mu.Unlock()
		if lead == "" {
			lead = c.slot
		}
		return coalitionScope(c.matchID, lead), nil
	case "dm":
		if out.TargetUserID == "" {
			return "", errInvalidScope("dm requires target_user_id")
		}
		return dmScopeForPair(c.matchID, c.user.UserID.String(), out.TargetUserID), nil
	default:
		return "", errInvalidScope("unknown scope")
	}
}

// errInvalidScope is a tiny ad-hoc error type so the WS controller can
// surface a meaningful "code" without dragging the whole errs package
// into wire validation.
type errInvalidScope string

func (e errInvalidScope) Error() string { return string(e) }

// maybeUpdateCoalitionFromState peeks at every outbound state envelope
// to learn the user's current coalition leader. Only re-computes when
// the diplomacy section changes shape; the lookup is O(slots^2) which
// is fine for ≤ 8 slots.
func (c *conn) maybeUpdateCoalitionFromState(payload []byte) {
	if c.chat == nil || c.slot == "" {
		return
	}
	var env wire.ServerEnvelope
	if err := json.Unmarshal(payload, &env); err != nil || env.State == nil {
		return
	}
	leader := coalitionLeaderFor(c.slot, env.State.Players, env.State.Diplomacy)
	c.chat.mu.Lock()
	c.chat.coalitionLead = leader
	c.chat.mu.Unlock()
}

// coalitionLeaderFor walks the wire-level treaty list and returns the
// alphabetically-smallest slot in the user's alliance graph (the
// coalition leader). When the user has no alliance the leader is the
// slot itself.
func coalitionLeaderFor(slot string, players []wire.PlayerState, treaties []wire.TreatyState) string {
	allies := map[string]bool{slot: true}
	stack := []string{slot}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, t := range treaties {
			if t.Stance != "alliance" {
				continue
			}
			other := ""
			switch cur {
			case t.SlotA:
				other = t.SlotB
			case t.SlotB:
				other = t.SlotA
			}
			if other == "" || allies[other] {
				continue
			}
			allies[other] = true
			stack = append(stack, other)
		}
	}
	leader := slot
	for _, p := range players {
		if allies[p.ID] && p.ID < leader {
			leader = p.ID
		}
	}
	return leader
}

func worldScope(matchID string) string { return "world:" + matchID }
func coalitionScope(matchID, lead string) string {
	return "coal:" + matchID + ":" + lead
}
func coalitionWildcard(matchID string) string {
	return "chat.scope.coal." + matchID + ".*"
}

// dmScopeForPair sorts the two user IDs so a/b and b/a map to the same
// scope key — must agree with chatdom.DMScope.
func dmScopeForPair(matchID, a, b string) string {
	pair := []string{a, b}
	if pair[0] > pair[1] {
		pair[0], pair[1] = pair[1], pair[0]
	}
	return "dm:" + matchID + ":" + pair[0] + ":" + pair[1]
}
