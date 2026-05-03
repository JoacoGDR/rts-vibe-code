package natsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/nats-io/nats.go"
)

// Chat subject helpers and payloads. The chat traffic uses core NATS
// (no JetStream) — durability is handled by the Postgres write the
// chatsvc performs before publishing.
const (
	SubjChatIngressAll = "chat.*.ingress"
)

// ChatIngressSubject is the subject the gateway publishes user-authored
// messages to. Core-api subscribes here, validates, persists and then
// fans out to the per-scope subjects.
func ChatIngressSubject(matchID string) string {
	return fmt.Sprintf("chat.%s.ingress", matchID)
}

// ChatScopeSubject converts a [chatdom] scope key into the NATS subject
// the gateway subscribes on. Tokens are dot-separated to match NATS
// conventions; the scope key itself uses colons because it doubles as a
// Postgres column.
func ChatScopeSubject(scope string) string {
	return "chat.scope." + strings.ReplaceAll(scope, ":", ".")
}

// ChatScopeWildcardForUser returns the subject pattern a gateway uses to
// receive every chat message addressed to a single user. The DM publisher
// emits one message per recipient under chat.scope.dm.<matchID>.inbox.<userID>
// so each user only needs one DM subscription regardless of how many
// open conversations they have.
func ChatScopeWildcardForUser(userID string) string {
	return "chat.scope.dm.*.inbox." + userID
}

// ChatIngressPayload is the on-wire shape the gateway publishes when a
// user authors a message. Core-api translates this into a SendInput
// before calling chatsvc.
type ChatIngressPayload struct {
	MatchID    string `json:"match_id"`
	AuthorID   string `json:"author_user_id"`
	AuthorSlot string `json:"author_slot"`
	Scope      string `json:"scope"`
	Body       string `json:"body"`
}

// PublishChatIngress is the gateway's entry point for forwarding an
// authored message to core-api.
func (p *Publisher) PublishChatIngress(_ context.Context, payload ChatIngressPayload) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.NC.Publish(ChatIngressSubject(payload.MatchID), raw)
}

// PublishChatMessage fans out an already-persisted message to every
// subscriber on the scope's NATS subject. Implements
// [chatsvc.Publisher].
func (p *Publisher) PublishChatMessage(_ context.Context, scope string, payload []byte) error {
	return p.NC.Publish(ChatScopeSubject(scope), payload)
}

// ChatIngressHandler is invoked once per ingress message after the
// payload has been decoded.
type ChatIngressHandler func(ctx context.Context, payload ChatIngressPayload) error

// SubscribeChatIngress wires a core NATS subscription for chat ingress.
// We deliberately use core NATS (not JetStream) for chat: the message
// becomes durable when chatsvc writes it to Postgres, and a missed
// in-flight message is fine — the History endpoint backfills clients.
func SubscribeChatIngress(ctx context.Context, logger *slog.Logger, nc *nats.Conn, handler ChatIngressHandler) error {
	_, err := nc.Subscribe(SubjChatIngressAll, func(m *nats.Msg) {
		var p ChatIngressPayload
		if err := json.Unmarshal(m.Data, &p); err != nil {
			logger.Warn("bad chat ingress payload", "err", err)
			return
		}
		if err := handler(ctx, p); err != nil {
			logger.Warn("chat ingress rejected", "err", err, "match", p.MatchID, "author", p.AuthorID)
		}
	})
	return err
}
