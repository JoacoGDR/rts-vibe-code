package lobbysvc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
	"github.com/joaquing/clone-supremacy/internal/domain/userdom"
	"github.com/joaquing/clone-supremacy/pkg/errs"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

// BotUsersRepository lists pre-seeded bot accounts for AI fill.
type BotUsersRepository interface {
	ListBots(ctx context.Context, limit int) ([]userdom.User, error)
}

// Leave removes the caller from a waiting match. In an active match it
// hands the slot to AI instead of removing the player row.
func (s *Service) Leave(ctx context.Context, matchID, userID uuid.UUID) (MatchView, error) {
	m, err := s.repo.Get(ctx, matchID)
	if err != nil {
		return MatchView{}, mapNotFound(err)
	}
	if m.Status == lobbydom.StatusActive || m.Status == lobbydom.StatusStarting {
		return s.Handoff(ctx, matchID, userID)
	}
	if m.Status != lobbydom.StatusWaiting {
		return MatchView{}, errs.New(errs.Conflict, lobbydom.ErrCannotLeaveActive.Error())
	}
	if err := s.repo.RemovePlayer(ctx, matchID, userID); err != nil {
		if errors.Is(err, lobbydom.ErrPlayerNotInMatch) {
			return MatchView{}, errs.New(errs.NotFound, "not in match")
		}
		return MatchView{}, errs.Wrap(err, errs.Internal, "leave failed")
	}
	return s.View(ctx, matchID)
}

// KickInput removes another player from a waiting lobby.
type KickInput struct {
	MatchID   uuid.UUID
	ActorID   uuid.UUID
	TargetUID uuid.UUID
}

func (s *Service) Kick(ctx context.Context, in KickInput) (MatchView, error) {
	m, err := s.repo.Get(ctx, in.MatchID)
	if err != nil {
		return MatchView{}, mapNotFound(err)
	}
	if m.CreatedBy != in.ActorID {
		return MatchView{}, errs.New(errs.Forbidden, lobbydom.ErrNotCreator.Error())
	}
	if m.Status != lobbydom.StatusWaiting {
		return MatchView{}, errs.New(errs.Conflict, "match already started")
	}
	if err := s.repo.RemovePlayer(ctx, in.MatchID, in.TargetUID); err != nil {
		if errors.Is(err, lobbydom.ErrPlayerNotInMatch) {
			return MatchView{}, errs.New(errs.NotFound, "target not in match")
		}
		return MatchView{}, errs.Wrap(err, errs.Internal, "kick failed")
	}
	return s.View(ctx, in.MatchID)
}

// Handoff flips the caller's slot to AI control in an active match.
func (s *Service) Handoff(ctx context.Context, matchID, userID uuid.UUID) (MatchView, error) {
	m, err := s.repo.Get(ctx, matchID)
	if err != nil {
		return MatchView{}, mapNotFound(err)
	}
	if m.Status != lobbydom.StatusActive && m.Status != lobbydom.StatusStarting {
		return MatchView{}, errs.New(errs.Conflict, "match is not running")
	}
	players, err := s.repo.ListPlayers(ctx, matchID)
	if err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "listing players")
	}
	var slot string
	for _, p := range players {
		if p.UserID == userID {
			slot = p.Slot
			break
		}
	}
	if slot == "" {
		return MatchView{}, errs.New(errs.NotFound, "not in match")
	}
	if _, err := s.repo.MarkControlledByAI(ctx, matchID, slot, true); err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "handoff failed")
	}
	return s.View(ctx, matchID)
}

// PatchSlotAIInput toggles AI control on an empty waiting slot (creator only).
type PatchSlotAIInput struct {
	MatchID   uuid.UUID
	ActorID   uuid.UUID
	Slot      string
	EnableAI  bool
}

func (s *Service) PatchSlotAI(ctx context.Context, in PatchSlotAIInput) (MatchView, error) {
	m, err := s.repo.Get(ctx, in.MatchID)
	if err != nil {
		return MatchView{}, mapNotFound(err)
	}
	if m.CreatedBy != in.ActorID {
		return MatchView{}, errs.New(errs.Forbidden, lobbydom.ErrNotCreator.Error())
	}
	if m.Status != lobbydom.StatusWaiting {
		return MatchView{}, errs.New(errs.Conflict, "match already started")
	}
	players, err := s.repo.ListPlayers(ctx, in.MatchID)
	if err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "listing players")
	}
	for _, p := range players {
		if p.Slot == in.Slot {
			return MatchView{}, errs.New(errs.Conflict, "slot is occupied")
		}
	}
	if in.EnableAI {
		if err := s.assignBotToSlot(ctx, in.MatchID, in.Slot, m.MapID); err != nil {
			return MatchView{}, err
		}
	} else {
		// Remove bot occupying this slot if any.
		for _, p := range players {
			if p.Slot == in.Slot {
				_ = s.repo.RemovePlayer(ctx, in.MatchID, p.UserID)
			}
		}
	}
	return s.View(ctx, in.MatchID)
}

func mapNotFound(err error) error {
	if errors.Is(err, lobbydom.ErrNotFound) {
		return errs.New(errs.NotFound, "match not found")
	}
	return errs.Wrap(err, errs.Internal, "fetching match")
}

// fillBots assigns bot users to every empty slot on the map before start.
func (s *Service) fillBots(ctx context.Context, matchID uuid.UUID, mapID string) error {
	if s.bots == nil {
		return nil
	}
	mp, err := maps.Load(mapID)
	if err != nil {
		return errs.Wrap(err, errs.Internal, "loading map")
	}
	players, err := s.repo.ListPlayers(ctx, matchID)
	if err != nil {
		return err
	}
	occupied := map[string]bool{}
	for _, p := range players {
		occupied[p.Slot] = true
	}
	need := 0
	for _, sl := range mp.Slots {
		if !occupied[sl.ID] {
			need++
		}
	}
	if need == 0 {
		return nil
	}
	bots, err := s.bots.ListBots(ctx, need)
	if err != nil {
		return errs.Wrap(err, errs.Internal, "listing bots")
	}
	if len(bots) < need {
		return errs.New(errs.Conflict, "not enough bot accounts configured")
	}
	bi := 0
	for _, sl := range mp.Slots {
		if occupied[sl.ID] {
			continue
		}
		bot := bots[bi]
		bi++
		if err := s.repo.AddPlayer(ctx, lobbydom.Player{
			MatchID: matchID, UserID: bot.ID, Slot: sl.ID,
			Color: sl.Color, Alive: true, ControlledByAI: true,
		}); err != nil {
			return errs.Wrap(err, errs.Internal, "assigning bot")
		}
	}
	return nil
}

func (s *Service) assignBotToSlot(ctx context.Context, matchID uuid.UUID, slot, mapID string) error {
	if s.bots == nil {
		return errs.New(errs.Internal, "bot pool not configured")
	}
	mp, err := maps.Load(mapID)
	if err != nil {
		return errs.Wrap(err, errs.BadRequest, "unknown map")
	}
	color := slotColor(mp, slot)
	bots, err := s.bots.ListBots(ctx, 1)
	if err != nil || len(bots) == 0 {
		return errs.New(errs.Conflict, "no bot accounts available")
	}
	return s.repo.AddPlayer(ctx, lobbydom.Player{
		MatchID: matchID, UserID: bots[0].ID, Slot: slot,
		Color: color, Alive: true, ControlledByAI: true,
	})
}

// publishStartWithFill marks starting, fills bots, marks active, publishes.
func (s *Service) publishStartWithFill(ctx context.Context, m lobbydom.Match) (MatchView, error) {
	if err := s.repo.MarkStarting(ctx, m.ID); err != nil {
		if errors.Is(err, lobbydom.ErrInvalidTransition) {
			return MatchView{}, errs.New(errs.Conflict, "already started")
		}
		return MatchView{}, errs.Wrap(err, errs.Internal, "marking starting")
	}
	if err := s.fillBots(ctx, m.ID, m.MapID); err != nil {
		return MatchView{}, err
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
	players, err = s.repo.ListPlayers(ctx, m.ID)
	if err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "listing players after fill")
	}
	slotAssign := map[string]string{}
	for _, p := range players {
		slotAssign[p.Slot] = p.UserID.String()
	}
	if err := s.publisher.PublishStart(ctx, natsbridge.StartPayload{
		MatchID: m.ID.String(), MapID: m.MapID, Speed: m.SpeedFactor,
		StartedAt: time.Now(), SlotAssignments: slotAssign,
	}); err != nil {
		return MatchView{}, errs.Wrap(err, errs.Internal, "publishing start")
	}
	return s.View(ctx, m.ID)
}
