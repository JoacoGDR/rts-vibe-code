package chatsvc

import (
	"context"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/chatdom"
	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
)

// MessagesRepository is the persistence dependency. The Postgres
// implementation lives in [pgrepo.Chat]. The query type lives in
// [chatdom] so the contract belongs to the domain layer.
type MessagesRepository interface {
	Insert(ctx context.Context, m chatdom.Message) error
	History(ctx context.Context, q chatdom.HistoryQuery) ([]chatdom.Message, error)
}

// PlayersRepository is the slice of [lobbysvc.MatchesRepository] that
// chatsvc actually needs. Declared here as a narrow interface so the
// service does not pull in the entire lobby contract.
type PlayersRepository interface {
	Get(ctx context.Context, id uuid.UUID) (lobbydom.Match, error)
	ListPlayers(ctx context.Context, matchID uuid.UUID) ([]lobbydom.Player, error)
}

// Publisher is the live-fanout dependency. The NATS implementation lives
// in [natsbridge].
type Publisher interface {
	PublishChatMessage(ctx context.Context, scope string, payload []byte) error
}
