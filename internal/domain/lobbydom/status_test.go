package lobbydom_test

import (
	"testing"

	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
)

func TestStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to lobbydom.Status
		ok       bool
	}{
		{lobbydom.StatusWaiting, lobbydom.StatusStarting, true},
		{lobbydom.StatusWaiting, lobbydom.StatusActive, false},
		{lobbydom.StatusStarting, lobbydom.StatusActive, true},
		{lobbydom.StatusActive, lobbydom.StatusEnded, true},
		{lobbydom.StatusActive, lobbydom.StatusAbandoned, true},
		{lobbydom.StatusEnded, lobbydom.StatusAbandoned, false},
	}
	for _, tc := range cases {
		if got := tc.from.CanTransitionTo(tc.to); got != tc.ok {
			t.Fatalf("%s -> %s: want %v got %v", tc.from, tc.to, tc.ok, got)
		}
	}
}
