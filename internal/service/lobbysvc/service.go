package lobbysvc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
	"github.com/joaquing/clone-supremacy/pkg/errs"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

// MatchesRepository is the dependency the lobby service needs. Postgres
// implementation lives in [pgrepo.Matches].
type MatchesRepository interface {
	Create(ctx context.Context, name, mapID string, createdBy uuid.UUID, speed float64) (lobbydom.Match, error)
	Get(ctx context.Context, id uuid.UUID) (lobbydom.Match, error)
	ListLobby(ctx context.Context, userID uuid.UUID) ([]lobbydom.Match, error)
	AddPlayer(ctx context.Context, p lobbydom.Player) error
	ListPlayers(ctx context.Context, matchID uuid.UUID) ([]lobbydom.Player, error)
	MarkActive(ctx context.Context, id uuid.UUID) error
}

// MatchStartPublisher is the engine notification dependency.
type MatchStartPublisher interface {
	PublishStart(ctx context.Context, p natsbridge.StartPayload) error
}

// Service is the lobby use-case orchestrator.
type Service struct {
	repo      MatchesRepository
	publisher MatchStartPublisher
	speed     float64
}

func New(repo MatchesRepository, publisher MatchStartPublisher, speed float64) *Service {
	return &Service{repo: repo, publisher: publisher, speed: speed}
}

// MatchView bundles a match with its player roster — convenient for the
// controller to map onto api.MatchView in one go.
type MatchView struct {
	Match   lobbydom.Match
	Players []lobbydom.Player
}

// CreateInput is the use-case argument for creating a match.
type CreateInput struct {
	Name   string
	MapID  string
	Slot   string
	UserID uuid.UUID
}

func (s *Service) Create(ctx context.Context, in CreateInput) (MatchView, error) {
	mapID := in.MapID
	if mapID == "" {
		mapID = "tiny-2p"
	}
	mp, err := maps.Load(mapID)
	if err != nil {
		return MatchView{}, errs.Wrap(err, errs.BadRequest, "unknown map")
	}
	slot := in.Slot
	if slot == "" {
		slot = mp.Slots[0].ID
	}
	m, err := s.repo.Create(ctx, in.Name, mapID, in.UserID, s.speed)
	if err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "creating match")
	}
	color := slotColor(mp, slot)
	if err := s.repo.AddPlayer(ctx, lobbydom.Player{
		MatchID: m.ID, UserID: in.UserID, Slot: slot, Color: color, Alive: true,
	}); err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "adding player")
	}
	return s.View(ctx, m.ID)
}

// JoinInput is the use-case argument for joining a match.
type JoinInput struct {
	MatchID uuid.UUID
	UserID  uuid.UUID
	Slot    string
}

// JoinResult tells the controller what happened: the resulting view, plus
// a flag that's true when the user was already in the match (so a 200
// "already_joined" response can be rendered).
type JoinResult struct {
	View          MatchView
	AlreadyJoined bool
	JoinedSlot    string
}

func (s *Service) Join(ctx context.Context, in JoinInput) (JoinResult, error) {
	m, err := s.repo.Get(ctx, in.MatchID)
	if err != nil {
		if errors.Is(err, lobbydom.ErrNotFound) {
			return JoinResult{}, errs.New(errs.NotFound, "match not found")
		}
		return JoinResult{}, errs.Wrap(err, errs.Internal, "fetching match")
	}
	if m.Status != lobbydom.StatusWaiting {
		return JoinResult{}, errs.New(errs.Conflict, "match already started")
	}
	mp, err := maps.Load(m.MapID)
	if err != nil {
		return JoinResult{}, errs.Wrap(err, errs.Internal, "loading map")
	}
	existing, _ := s.repo.ListPlayers(ctx, m.ID)
	taken := map[string]bool{}
	for _, p := range existing {
		taken[p.Slot] = true
		if p.UserID == in.UserID {
			view, err := s.View(ctx, m.ID)
			if err != nil {
				return JoinResult{}, err
			}
			return JoinResult{View: view, AlreadyJoined: true, JoinedSlot: p.Slot}, nil
		}
	}
	chosen := in.Slot
	if chosen == "" {
		for _, slot := range mp.Slots {
			if !taken[slot.ID] {
				chosen = slot.ID
				break
			}
		}
	}
	if chosen == "" || taken[chosen] {
		return JoinResult{}, errs.New(errs.Conflict, "no free slot")
	}
	color := slotColor(mp, chosen)
	if err := s.repo.AddPlayer(ctx, lobbydom.Player{
		MatchID: m.ID, UserID: in.UserID, Slot: chosen, Color: color, Alive: true,
	}); err != nil {
		return JoinResult{}, errs.Wrap(err, errs.Internal, "adding player")
	}
	view, err := s.View(ctx, m.ID)
	if err != nil {
		return JoinResult{}, err
	}
	return JoinResult{View: view, JoinedSlot: chosen}, nil
}

// Start transitions a match to active and notifies the engine to host
// it.
func (s *Service) Start(ctx context.Context, matchID uuid.UUID) (MatchView, error) {
	m, err := s.repo.Get(ctx, matchID)
	if err != nil {
		if errors.Is(err, lobbydom.ErrNotFound) {
			return MatchView{}, errs.New(errs.NotFound, "match not found")
		}
		return MatchView{}, errs.Wrap(err, errs.Internal, "fetching match")
	}
	if m.Status != lobbydom.StatusWaiting {
		return MatchView{}, errs.New(errs.Conflict, "already started")
	}
	players, err := s.repo.ListPlayers(ctx, m.ID)
	if err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "listing players")
	}
	if len(players) < 2 {
		return MatchView{}, errs.New(errs.Conflict, "need at least 2 players")
	}
	if err := s.repo.MarkActive(ctx, m.ID); err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "marking active")
	}
	slotAssign := map[string]string{}
	for _, p := range players {
		slotAssign[p.Slot] = p.UserID.String()
	}
	if err := s.publisher.PublishStart(ctx, natsbridge.StartPayload{
		MatchID:         m.ID.String(),
		MapID:           m.MapID,
		Speed:           m.SpeedFactor,
		StartedAt:       time.Now(),
		SlotAssignments: slotAssign,
	}); err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "publishing start")
	}
	return s.View(ctx, m.ID)
}

// Get returns a single match's view by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (MatchView, error) {
	view, err := s.View(ctx, id)
	if err != nil {
		if errors.Is(err, lobbydom.ErrNotFound) {
			return MatchView{}, errs.New(errs.NotFound, "match not found")
		}
		return MatchView{}, err
	}
	return view, nil
}

// ListLobby returns lobby-visible matches: all waiting games (joinable by
// anyone) plus any match the user is already a player in.
func (s *Service) ListLobby(ctx context.Context, userID uuid.UUID) ([]MatchView, error) {
	ms, err := s.repo.ListLobby(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(err, errs.Internal, "listing matches")
	}
	out := make([]MatchView, 0, len(ms))
	for _, m := range ms {
		v, err := s.View(ctx, m.ID)
		if err == nil {
			out = append(out, v)
		}
	}
	return out, nil
}

// View loads a match record + its player roster.
func (s *Service) View(ctx context.Context, id uuid.UUID) (MatchView, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return MatchView{}, err
	}
	pls, err := s.repo.ListPlayers(ctx, id)
	if err != nil {
		return MatchView{}, err
	}
	return MatchView{Match: m, Players: pls}, nil
}

func slotColor(mp *maps.Map, slotID string) string {
	for _, s := range mp.Slots {
		if s.ID == slotID {
			return s.Color
		}
	}
	return "#888888"
}
