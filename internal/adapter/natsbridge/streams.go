package natsbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// EnsureStreams creates the durable command stream that holds both
// player commands and start announcements. Idempotent.
func EnsureStreams(ctx context.Context, js jetstream.JetStream) error {
	cfg := jetstream.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{SubjCmdAll, SubjStartAll},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		Storage:   jetstream.FileStorage,
	}
	if _, err := js.CreateOrUpdateStream(ctx, cfg); err != nil {
		return fmt.Errorf("create cmd stream: %w", err)
	}
	return nil
}
