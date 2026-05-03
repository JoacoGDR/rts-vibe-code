package chatsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/chatdom"
	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
	"github.com/joaquing/clone-supremacy/pkg/errs"
	"github.com/joaquing/clone-supremacy/pkg/shared/wire"
)

// MessagesPerSecond is the per-user soft rate limit. The service drops
// authoring attempts above this with [errs.RateLimited].
const MessagesPerSecond = 5

// Service is the chat use-case orchestrator. Construct it once per
// process and reuse — the in-memory rate limiter is keyed by user.
type Service struct {
	repo      MessagesRepository
	players   PlayersRepository
	publisher Publisher

	mu       sync.Mutex
	throttle map[uuid.UUID]*tokenBucket
}

// New constructs a Service with the given dependencies.
func New(repo MessagesRepository, players PlayersRepository, publisher Publisher) *Service {
	return &Service{
		repo:      repo,
		players:   players,
		publisher: publisher,
		throttle:  map[uuid.UUID]*tokenBucket{},
	}
}

// SendInput is the use-case argument for posting a chat message. The
// gateway resolves the issuer's slot from Redis before forwarding, so the
// service does not need to repeat that lookup.
//
// Scope is a fully-qualified scope key ("world:<matchID>", etc.). For
// DMs the gateway constructs the scope using the canonical
// [chatdom.DMScope] helper so a/b and b/a resolve to the same record.
type SendInput struct {
	MatchID    uuid.UUID
	AuthorID   uuid.UUID
	AuthorSlot string
	Scope      string
	Body       string
}

// Send validates, persists, and publishes a single chat message. Returns
// the freshly-stamped [chatdom.Message] so the caller can echo it on the
// HTTP response when posting via REST.
func (s *Service) Send(ctx context.Context, in SendInput) (chatdom.Message, error) {
	body, err := chatdom.ValidateBody(in.Body)
	if err != nil {
		return chatdom.Message{}, errs.Wrap(err, errs.BadRequest, "invalid body")
	}
	kind, parts, err := chatdom.ParseScope(in.Scope)
	if err != nil {
		return chatdom.Message{}, errs.Wrap(err, errs.BadRequest, "invalid scope")
	}
	if err := s.checkScope(ctx, in, kind, parts); err != nil {
		return chatdom.Message{}, err
	}
	if !s.allow(in.AuthorID) {
		return chatdom.Message{}, errs.New(errs.RateLimited, "chat rate exceeded")
	}

	msg := chatdom.Message{
		ID:         uuid.New(),
		MatchID:    in.MatchID,
		Scope:      in.Scope,
		AuthorID:   in.AuthorID,
		AuthorSlot: in.AuthorSlot,
		Body:       body,
		SentAt:     time.Now(),
	}
	if err := s.repo.Insert(ctx, msg); err != nil {
		return chatdom.Message{}, errs.Wrap(err, errs.Internal, "persisting chat message")
	}
	if err := s.fanout(ctx, msg, kind, parts); err != nil {
		// Persisted successfully; live fanout is best-effort.
		return msg, nil //nolint:nilerr
	}
	return msg, nil
}

// History reads recent messages for the given scope. Authorisation is
// the caller's job — the gateway / HTTP controller must check the
// requesting user belongs to the scope before invoking this.
func (s *Service) History(ctx context.Context, q chatdom.HistoryQuery) ([]chatdom.Message, error) {
	out, err := s.repo.History(ctx, q)
	if err != nil {
		return nil, errs.Wrap(err, errs.Internal, "loading chat history")
	}
	return out, nil
}

// checkScope enforces the membership rules the service can verify
// without engine context (match exists, user is a player, DM author is a
// participant). Coalition membership is trusted from the gateway because
// it requires live diplomacy state.
func (s *Service) checkScope(ctx context.Context, in SendInput, kind chatdom.ChannelKind, parts []string) error {
	players, err := s.players.ListPlayers(ctx, in.MatchID)
	if err != nil {
		return errs.Wrap(err, errs.Internal, "loading players")
	}
	if !isPlayer(players, in.AuthorID, in.AuthorSlot) {
		return errs.New(errs.Forbidden, "not a member of this match")
	}
	switch kind {
	case chatdom.ChannelWorld:
		if parts[0] != in.MatchID.String() {
			return errs.New(errs.BadRequest, "scope match id mismatch")
		}
	case chatdom.ChannelCoalition:
		if parts[0] != in.MatchID.String() {
			return errs.New(errs.BadRequest, "scope match id mismatch")
		}
	case chatdom.ChannelDM:
		if parts[0] != in.MatchID.String() {
			return errs.New(errs.BadRequest, "scope match id mismatch")
		}
		if !strings.Contains(in.Scope, in.AuthorID.String()) {
			return errs.New(errs.Forbidden, "author not a DM participant")
		}
		if err := dmPartnerIsPlayer(in, parts, players); err != nil {
			return err
		}
	}
	return nil
}

// fanout publishes the message on the right NATS subject(s) for the
// scope kind. DMs are emitted twice (once per participant) so each
// recipient can subscribe to a single per-user dm subject.
func (s *Service) fanout(ctx context.Context, msg chatdom.Message, kind chatdom.ChannelKind, parts []string) error {
	wireMsg := wire.ChatInbound{
		ID:         msg.ID.String(),
		MatchID:    msg.MatchID.String(),
		Scope:      msg.Scope,
		AuthorID:   msg.AuthorID.String(),
		AuthorSlot: msg.AuthorSlot,
		Body:       msg.Body,
		SentAt:     msg.SentAt,
	}
	payload, err := json.Marshal(wireMsg)
	if err != nil {
		return err
	}
	switch kind {
	case chatdom.ChannelWorld, chatdom.ChannelCoalition:
		return s.publisher.PublishChatMessage(ctx, msg.Scope, payload)
	case chatdom.ChannelDM:
		// parts: [matchID, userA, userB] (sorted). Each participant's
		// gateway subscribes to their personal inbox subject; we publish
		// once per recipient so the gateway only needs one
		// subscription per user instead of a wildcard.
		dmA := dmPerUserScope(msg.MatchID.String(), parts[1])
		dmB := dmPerUserScope(msg.MatchID.String(), parts[2])
		if err := s.publisher.PublishChatMessage(ctx, dmA, payload); err != nil {
			return err
		}
		if dmA != dmB {
			return s.publisher.PublishChatMessage(ctx, dmB, payload)
		}
	}
	return nil
}

// dmPerUserScope is the canonical key the gateway subscribes to for a
// per-user DM inbox. Kept as a package-private helper because only the
// fanout cares about it.
func dmPerUserScope(matchID, userID string) string {
	return "dm:" + matchID + ":inbox:" + userID
}

func dmPartnerIsPlayer(in SendInput, parts []string, players []lobbydom.Player) error {
	// parts: [matchID, userA, userB]
	a, errA := uuid.Parse(parts[1])
	b, errB := uuid.Parse(parts[2])
	if errors.Join(errA, errB) != nil {
		return errs.New(errs.BadRequest, "invalid dm participants")
	}
	other := a
	if a == in.AuthorID {
		other = b
	}
	for _, p := range players {
		if p.UserID == other {
			return nil
		}
	}
	return errs.New(errs.NotFound, "dm partner is not in this match")
}

func isPlayer(players []lobbydom.Player, userID uuid.UUID, slot string) bool {
	for _, p := range players {
		if p.UserID == userID && (slot == "" || p.Slot == slot) {
			return true
		}
	}
	return false
}
