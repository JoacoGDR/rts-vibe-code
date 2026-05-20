package enginesvc

import (
	"context"
	"testing"
	"time"

	"github.com/joaquing/clone-supremacy/internal/domain/matchdom"
	"github.com/joaquing/clone-supremacy/internal/domain/visibility"
)

type noopBroadcaster struct{}

func (noopBroadcaster) PublishPublicState(context.Context, string, []byte) error { return nil }
func (noopBroadcaster) PublishSlotState(context.Context, string, string, []byte) error {
	return nil
}
func (noopBroadcaster) PublishPublicEvent(context.Context, string, []byte) error { return nil }
func (noopBroadcaster) PublishSlotEvent(context.Context, string, string, []byte) error {
	return nil
}
func (noopBroadcaster) PublishFinalState(context.Context, string, []byte) error { return nil }

func BenchmarkRunnerBroadcastEvents(b *testing.B) {
	m := visibility.BuildSyntheticMatch(400, 50, 10)
	r := NewRunner(nil, nil, noopBroadcaster{}, nil)
	events := []matchdom.AppliedEvent{
		{Kind: "macro_pulse", Province: "p000", Seq: 1, OccurAt: time.Now()},
		{Kind: "macro_pulse", Province: "p001", Seq: 2, OccurAt: time.Now()},
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.broadcastEvents(ctx, m, events)
	}
}
