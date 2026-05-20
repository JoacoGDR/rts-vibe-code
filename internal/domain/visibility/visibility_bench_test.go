package visibility

import (
	"testing"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
)

func BenchmarkFor_medium(b *testing.B) {
	m := BuildSyntheticMatch(400, 50, 10)
	slot := "s000"
	at := m.GameNow
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		For(m, slot, at)
	}
}

func BenchmarkFor_large(b *testing.B) {
	m := BuildSyntheticMatch(1000, 100, 20)
	slot := "s000"
	at := m.GameNow
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		For(m, slot, at)
	}
}

func BenchmarkBatchFiltered_allSlots(b *testing.B) {
	m := BuildSyntheticMatch(400, 50, 10)
	batch := NewBatch(m)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for slot := range m.Players {
			_ = batch.Filtered(slot)
		}
	}
}

func BenchmarkBroadcastPattern_legacy(b *testing.B) {
	m := BuildSyntheticMatch(400, 50, 10)
	at := m.GameNow
	events := 3
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for e := 0; e < events; e++ {
			for slot := range m.Players {
				For(m, slot, at)
			}
		}
		for slot := range m.Players {
			Filtered(m, slot)
		}
	}
}

func BenchmarkBroadcastPattern_batch(b *testing.B) {
	m := BuildSyntheticMatch(400, 50, 10)
	events := 3
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := NewBatch(m)
		for e := 0; e < events; e++ {
			ev := matchdom.AppliedEvent{Kind: "macro_pulse", Province: "p000"}
			for slot := range m.Players {
				_ = batch.SlotObservesEvent(slot, ev)
			}
		}
		for slot := range m.Players {
			_ = batch.Filtered(slot)
		}
	}
}
