package diplomacydom_test

import (
	"errors"
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/diplomacydom"
)

func TestDefaultStanceIsWar(t *testing.T) {
	r := diplomacydom.New()
	if r.Stance("red", "blue") != diplomacydom.War {
		t.Fatalf("expected default stance war, got %s", r.Stance("red", "blue"))
	}
}

func TestSelfStanceIsPeace(t *testing.T) {
	r := diplomacydom.New()
	if r.Stance("red", "red") != diplomacydom.Peace {
		t.Fatalf("self stance should be peace")
	}
}

func TestPeaceProposeAccept(t *testing.T) {
	r := diplomacydom.New()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	if err := r.ProposePeace("red", "blue", now); err != nil {
		t.Fatalf("propose peace: %v", err)
	}
	if r.Stance("red", "blue") != diplomacydom.War {
		t.Fatal("stance must remain war until accepted")
	}
	if err := r.AcceptPeace("blue", "red", now); err != nil {
		t.Fatalf("accept peace: %v", err)
	}
	if r.Stance("red", "blue") != diplomacydom.Peace {
		t.Fatalf("expected peace, got %s", r.Stance("red", "blue"))
	}
}

func TestProposerCannotAcceptOwnOffer(t *testing.T) {
	r := diplomacydom.New()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	_ = r.ProposePeace("red", "blue", now)
	if err := r.AcceptPeace("red", "blue", now); !errors.Is(err, diplomacydom.ErrNoPendingOffer) {
		t.Fatalf("expected ErrNoPendingOffer, got %v", err)
	}
}

func TestDeclareWarRevokesPacts(t *testing.T) {
	r := diplomacydom.New()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	_ = r.ProposePeace("red", "blue", now)
	_ = r.AcceptPeace("blue", "red", now)
	if err := r.GrantPact("red", "blue", diplomacydom.ShareMap, now); err != nil {
		t.Fatalf("grant: %v", err)
	}
	later := now.Add(2 * time.Hour)
	if err := r.DeclareWar("red", "blue", later); err != nil {
		t.Fatalf("declare war: %v", err)
	}
	if r.HasPact("red", "blue", diplomacydom.ShareMap) {
		t.Fatal("share-map pact should be revoked when war is declared")
	}
}

func TestCooldownBlocksRapidChanges(t *testing.T) {
	r := diplomacydom.New()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	_ = r.ProposePeace("red", "blue", now)
	_ = r.AcceptPeace("blue", "red", now)
	if err := r.DeclareWar("red", "blue", now.Add(time.Minute)); !errors.Is(err, diplomacydom.ErrCooldown) {
		t.Fatalf("expected cooldown, got %v", err)
	}
}

func TestAllianceTransitiveCoalition(t *testing.T) {
	r := diplomacydom.New()
	now := time.Now()
	_ = r.ProposeAlliance("red", "blue", now)
	_ = r.AcceptAlliance("blue", "red", now)
	_ = r.ProposeAlliance("blue", "green", now)
	_ = r.AcceptAlliance("green", "blue", now)

	coal := r.Coalition("red", []string{"red", "blue", "green", "yellow"})
	if !coal["red"] || !coal["blue"] || !coal["green"] {
		t.Fatalf("expected red/blue/green in coalition, got %v", coal)
	}
	if coal["yellow"] {
		t.Fatal("yellow should not be in coalition")
	}
	if got := r.CoalitionID("green", []string{"red", "blue", "green", "yellow"}); got != "blue" {
		// "blue" sorts before "green" and "red" alphabetically
		t.Fatalf("expected coalition leader 'blue', got %q", got)
	}
}

func TestRightOfWayGatesMovement(t *testing.T) {
	r := diplomacydom.New()
	now := time.Now()
	_ = r.ProposePeace("red", "blue", now)
	_ = r.AcceptPeace("blue", "red", now)

	if r.MayMoveThrough("red", "blue") {
		t.Fatal("plain peace must not allow movement onto enemy provinces")
	}
	_ = r.GrantPact("blue", "red", diplomacydom.RightOfWay, now)
	if !r.MayMoveThrough("red", "blue") {
		t.Fatal("right-of-way grant should permit movement")
	}
}

func TestShareMapVisibility(t *testing.T) {
	r := diplomacydom.New()
	now := time.Now()
	if r.SeesThrough("red", "blue") {
		t.Fatal("default war: red must not see through blue")
	}
	_ = r.GrantPact("blue", "red", diplomacydom.ShareMap, now)
	if !r.SeesThrough("red", "blue") {
		t.Fatal("share-map grant should give red sight through blue")
	}
}
