package lobbysvc_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/natsbridge"
	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
	"github.com/joaquing/clone-supremacy/internal/domain/userdom"
	"github.com/joaquing/clone-supremacy/internal/service/lobbysvc"
)

type fakeRepo struct {
	match   lobbydom.Match
	players []lobbydom.Player
	started bool
	active  bool
}

func (f *fakeRepo) Create(_ context.Context, name, mapID string, createdBy uuid.UUID, speed float64) (lobbydom.Match, error) {
	f.match = lobbydom.Match{
		ID: uuid.New(), Name: name, MapID: mapID, Status: lobbydom.StatusWaiting,
		CreatedBy: createdBy, SpeedFactor: speed,
	}
	return f.match, nil
}
func (f *fakeRepo) Get(_ context.Context, id uuid.UUID) (lobbydom.Match, error) {
	if id != f.match.ID {
		return lobbydom.Match{}, lobbydom.ErrNotFound
	}
	if f.active {
		f.match.Status = lobbydom.StatusActive
	} else if f.started {
		f.match.Status = lobbydom.StatusStarting
	}
	return f.match, nil
}
func (f *fakeRepo) ListLobby(context.Context, uuid.UUID) ([]lobbydom.Match, error) {
	return nil, nil
}
func (f *fakeRepo) AddPlayer(_ context.Context, p lobbydom.Player) error {
	f.players = append(f.players, p)
	return nil
}
func (f *fakeRepo) ListPlayers(context.Context, uuid.UUID) ([]lobbydom.Player, error) {
	return f.players, nil
}
func (f *fakeRepo) MarkStarting(context.Context, uuid.UUID) error {
	f.started = true
	return nil
}
func (f *fakeRepo) MarkActive(context.Context, uuid.UUID) error {
	f.active = true
	return nil
}
func (f *fakeRepo) MarkControlledByAI(context.Context, uuid.UUID, string, bool) (bool, error) {
	return true, nil
}
func (f *fakeRepo) RemovePlayer(_ context.Context, _, uid uuid.UUID) error {
	out := f.players[:0]
	for _, p := range f.players {
		if p.UserID != uid {
			out = append(out, p)
		}
	}
	f.players = out
	return nil
}

type fakeBots struct{ ids []uuid.UUID }

func (b *fakeBots) ListBots(_ context.Context, limit int) ([]userdom.User, error) {
	out := make([]userdom.User, 0, limit)
	for i := 0; i < limit && i < len(b.ids); i++ {
		out = append(out, userdom.User{ID: b.ids[i], Email: "bot@test"})
	}
	return out, nil
}

type fakePub struct {
	last natsbridge.StartPayload
}

func (p *fakePub) PublishStart(_ context.Context, payload natsbridge.StartPayload) error {
	p.last = payload
	return nil
}

func TestStartFillsEmptySlotWithBot(t *testing.T) {
	human := uuid.New()
	botID := uuid.New()
	matchID := uuid.New()
	repo := &fakeRepo{
		match: lobbydom.Match{
			ID: matchID, MapID: "tiny-2p", Status: lobbydom.StatusWaiting,
			CreatedBy: human, SpeedFactor: 60,
		},
		players: []lobbydom.Player{
			{MatchID: matchID, UserID: human, Slot: "red", Alive: true},
		},
	}
	pub := &fakePub{}
	svc := lobbysvc.NewWithBots(repo, &fakeBots{ids: []uuid.UUID{botID}}, pub, 60)

	_, err := svc.Start(context.Background(), repo.match.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pub.last.SlotAssignments) != 2 {
		t.Fatalf("assignments: %+v", pub.last.SlotAssignments)
	}
	if pub.last.SlotAssignments["blue"] != botID.String() {
		t.Fatalf("expected bot in blue, got %+v", pub.last.SlotAssignments)
	}
}

func TestCreateAutoStartFillsFourPlayerClassic4p(t *testing.T) {
	human := uuid.New()
	bots := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	pub := &fakePub{}
	svc := lobbysvc.NewWithBots(&fakeRepo{}, &fakeBots{ids: bots}, pub, 60)

	view, err := svc.Create(context.Background(), lobbysvc.CreateInput{
		Name: "Ops", MapID: "classic-4p", Slot: "red", UserID: human,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Match.Status != lobbydom.StatusActive {
		t.Fatalf("status: %s", view.Match.Status)
	}
	if len(view.Players) != 4 {
		t.Fatalf("players: %d", len(view.Players))
	}
	ai := 0
	for _, p := range view.Players {
		if p.ControlledByAI {
			ai++
		}
	}
	if ai != 3 {
		t.Fatalf("expected 3 AI players, got %d", ai)
	}
	if len(pub.last.SlotAssignments) != 4 {
		t.Fatalf("assignments: %+v", pub.last.SlotAssignments)
	}
}

func TestCreateOpenLobbyStaysWaiting(t *testing.T) {
	human := uuid.New()
	falseVal := false
	svc := lobbysvc.NewWithBots(&fakeRepo{}, &fakeBots{ids: []uuid.UUID{uuid.New()}}, &fakePub{}, 60)

	view, err := svc.Create(context.Background(), lobbysvc.CreateInput{
		Name: "Lobby", MapID: "tiny-2p", Slot: "red", UserID: human, AutoStart: &falseVal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Match.Status != lobbydom.StatusWaiting {
		t.Fatalf("status: %s", view.Match.Status)
	}
	if len(view.Players) != 1 {
		t.Fatalf("players: %d", len(view.Players))
	}
}
