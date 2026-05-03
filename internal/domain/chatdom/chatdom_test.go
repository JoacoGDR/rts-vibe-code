package chatdom_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/chatdom"
)

func TestValidateBodyTrims(t *testing.T) {
	body, err := chatdom.ValidateBody("  hello  ")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if body != "hello" {
		t.Fatalf("expected trimmed body, got %q", body)
	}
}

func TestValidateBodyRejectsEmpty(t *testing.T) {
	if _, err := chatdom.ValidateBody("   "); !errors.Is(err, chatdom.ErrEmptyBody) {
		t.Fatalf("expected ErrEmptyBody, got %v", err)
	}
}

func TestValidateBodyRejectsTooLong(t *testing.T) {
	body := strings.Repeat("x", chatdom.MaxBodyBytes+1)
	if _, err := chatdom.ValidateBody(body); !errors.Is(err, chatdom.ErrBodyTooLong) {
		t.Fatalf("expected ErrBodyTooLong, got %v", err)
	}
}

func TestValidateBodyRejectsControlChars(t *testing.T) {
	if _, err := chatdom.ValidateBody("hi\x00there"); !errors.Is(err, chatdom.ErrControlChars) {
		t.Fatalf("expected ErrControlChars, got %v", err)
	}
}

func TestDMScopeIsOrderIndependent(t *testing.T) {
	matchID := uuid.New()
	a, b := uuid.New(), uuid.New()
	if chatdom.DMScope(matchID, a, b) != chatdom.DMScope(matchID, b, a) {
		t.Fatal("DM scope must be order-independent")
	}
}

func TestParseScope(t *testing.T) {
	matchID := uuid.New()
	kind, _, err := chatdom.ParseScope(chatdom.WorldScope(matchID))
	if err != nil || kind != chatdom.ChannelWorld {
		t.Fatalf("expected world, got %s err=%v", kind, err)
	}
	kind, _, err = chatdom.ParseScope(chatdom.CoalitionScope(matchID, "blue"))
	if err != nil || kind != chatdom.ChannelCoalition {
		t.Fatalf("expected coalition, got %s err=%v", kind, err)
	}
	kind, _, err = chatdom.ParseScope(chatdom.DMScope(matchID, uuid.New(), uuid.New()))
	if err != nil || kind != chatdom.ChannelDM {
		t.Fatalf("expected dm, got %s err=%v", kind, err)
	}
	if _, _, err := chatdom.ParseScope("nope"); !errors.Is(err, chatdom.ErrUnknownScope) {
		t.Fatalf("expected ErrUnknownScope, got %v", err)
	}
}
