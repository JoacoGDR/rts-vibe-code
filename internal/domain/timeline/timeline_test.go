package timeline

import (
	"testing"
	"time"
)

func TestTimelineOrdersByTime(t *testing.T) {
	tl := New()
	now := time.Now()
	tl.Push(&Event{At: now.Add(3 * time.Second), Kind: Arrival})
	tl.Push(&Event{At: now.Add(1 * time.Second), Kind: Arrival})
	tl.Push(&Event{At: now.Add(2 * time.Second), Kind: Arrival})

	want := []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second}
	for i, w := range want {
		ev := tl.Pop()
		if ev == nil {
			t.Fatalf("pop %d: expected event", i)
		}
		got := ev.At.Sub(now)
		if got != w {
			t.Fatalf("pop %d: want %v got %v", i, w, got)
		}
	}
	if tl.Pop() != nil {
		t.Fatal("expected empty timeline")
	}
}
