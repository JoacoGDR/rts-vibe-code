package chatsvc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/chatdom"
	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
	"github.com/joaquing/clone-supremacy/internal/service/chatsvc"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

type fakeRepo struct {
	inserted []chatdom.Message
}

func (r *fakeRepo) Insert(_ context.Context, m chatdom.Message) error {
	r.inserted = append(r.inserted, m)
	return nil
}

func (r *fakeRepo) History(_ context.Context, _ chatdom.HistoryQuery) ([]chatdom.Message, error) {
	return r.inserted, nil
}

type fakePlayers struct {
	players []lobbydom.Player
}

func (f *fakePlayers) Get(_ context.Context, _ uuid.UUID) (lobbydom.Match, error) {
	return lobbydom.Match{}, nil
}

func (f *fakePlayers) ListPlayers(_ context.Context, _ uuid.UUID) ([]lobbydom.Player, error) {
	return f.players, nil
}

type fakePub struct {
	calls []string
}

func (p *fakePub) PublishChatMessage(_ context.Context, scope string, _ []byte) error {
	p.calls = append(p.calls, scope)
	return nil
}

func TestSendValidatesAndPersistsWorld(t *testing.T) {
	matchID := uuid.New()
	authorID := uuid.New()
	repo := &fakeRepo{}
	players := &fakePlayers{players: []lobbydom.Player{{
		MatchID: matchID, UserID: authorID, Slot: "red",
	}}}
	pub := &fakePub{}
	svc := chatsvc.New(repo, players, pub)

	msg, err := svc.Send(context.Background(), chatsvc.SendInput{
		MatchID: matchID, AuthorID: authorID, AuthorSlot: "red",
		Scope: chatdom.WorldScope(matchID), Body: "hello there",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("expected one message persisted, got %d", len(repo.inserted))
	}
	if msg.Body != "hello there" {
		t.Fatalf("unexpected body %q", msg.Body)
	}
	if len(pub.calls) != 1 {
		t.Fatalf("expected one publish, got %d", len(pub.calls))
	}
}

func TestSendRejectsNonMember(t *testing.T) {
	matchID := uuid.New()
	authorID := uuid.New()
	repo := &fakeRepo{}
	players := &fakePlayers{} // empty roster
	svc := chatsvc.New(repo, players, &fakePub{})

	_, err := svc.Send(context.Background(), chatsvc.SendInput{
		MatchID: matchID, AuthorID: authorID, AuthorSlot: "red",
		Scope: chatdom.WorldScope(matchID), Body: "hi",
	})
	e := errs.As(err)
	if e == nil || e.Code != errs.Forbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestSendRateLimitsAfterBurst(t *testing.T) {
	matchID := uuid.New()
	authorID := uuid.New()
	repo := &fakeRepo{}
	players := &fakePlayers{players: []lobbydom.Player{{
		MatchID: matchID, UserID: authorID, Slot: "red",
	}}}
	svc := chatsvc.New(repo, players, &fakePub{})

	scope := chatdom.WorldScope(matchID)
	in := chatsvc.SendInput{
		MatchID: matchID, AuthorID: authorID, AuthorSlot: "red", Scope: scope,
	}
	for i := 0; i < chatsvc.MessagesPerSecond; i++ {
		in.Body = "ok"
		if _, err := svc.Send(context.Background(), in); err != nil {
			t.Fatalf("burst %d: %v", i, err)
		}
	}
	in.Body = "overflow"
	_, err := svc.Send(context.Background(), in)
	e := errs.As(err)
	if e == nil || e.Code != errs.RateLimited {
		t.Fatalf("expected rate-limit, got %v", err)
	}
}

func TestSendRejectsDMAsNonParticipant(t *testing.T) {
	matchID := uuid.New()
	authorID := uuid.New()
	other := uuid.New()
	stranger := uuid.New()
	repo := &fakeRepo{}
	players := &fakePlayers{players: []lobbydom.Player{
		{MatchID: matchID, UserID: authorID, Slot: "red"},
		{MatchID: matchID, UserID: other, Slot: "blue"},
	}}
	svc := chatsvc.New(repo, players, &fakePub{})
	scope := chatdom.DMScope(matchID, other, stranger)
	_, err := svc.Send(context.Background(), chatsvc.SendInput{
		MatchID: matchID, AuthorID: authorID, AuthorSlot: "red",
		Scope: scope, Body: "secret",
	})
	if err == nil {
		t.Fatal("expected DM rejection when author is not a participant")
	}
	e := errs.As(err)
	if e == nil || (e.Code != errs.Forbidden && e.Code != errs.NotFound) {
		t.Fatalf("expected forbidden/not_found, got %v", err)
	}
}

// TestSendDMFanoutsToBothParticipants ensures the publisher is invoked
// once per participant on a DM send.
func TestSendDMFanoutsToBothParticipants(t *testing.T) {
	matchID := uuid.New()
	authorID := uuid.New()
	other := uuid.New()
	repo := &fakeRepo{}
	players := &fakePlayers{players: []lobbydom.Player{
		{MatchID: matchID, UserID: authorID, Slot: "red"},
		{MatchID: matchID, UserID: other, Slot: "blue"},
	}}
	pub := &fakePub{}
	svc := chatsvc.New(repo, players, pub)
	scope := chatdom.DMScope(matchID, authorID, other)
	if _, err := svc.Send(context.Background(), chatsvc.SendInput{
		MatchID: matchID, AuthorID: authorID, AuthorSlot: "red",
		Scope: scope, Body: "hi",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(pub.calls) != 2 {
		t.Fatalf("expected two DM publishes, got %d", len(pub.calls))
	}
}

// silence unused import noise in case we trim time later.
var _ = time.Second

// silence unused linter error import.
var _ = errors.New
