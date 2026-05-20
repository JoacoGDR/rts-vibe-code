package visibility

import (
	"fmt"
	"reflect"
	"testing"
)

func TestForMatchesLegacyBruteForce(t *testing.T) {
	cases := []struct {
		provinces, slots, unitsPerSlot int
	}{
		{9, 2, 2},
		{100, 10, 5},
		{400, 50, 10},
	}
	for _, tc := range cases {
		name := fmt.Sprintf("p%d_s%d_u%d", tc.provinces, tc.slots, tc.unitsPerSlot)
		t.Run(name, func(t *testing.T) {
			m := BuildSyntheticMatch(tc.provinces, tc.slots, tc.unitsPerSlot)
			at := m.GameNow
			for slot := range m.Players {
				legacyP, legacyU := forLegacy(m, slot, at)
				optP, optU := For(m, slot, at)
				if !reflect.DeepEqual(legacyP, optP) {
					t.Fatalf("slot %s provinces differ", slot)
				}
				if !reflect.DeepEqual(legacyU, optU) {
					t.Fatalf("slot %s units differ", slot)
				}
			}
		})
	}
}
