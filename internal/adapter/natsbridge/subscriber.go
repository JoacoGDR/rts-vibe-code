package natsbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ErrNotHosted signals that the consumer's runner does not host the match
// referenced by a payload. Callbacks return this to ask the bridge to NAK
// the message so another engine can try.
var ErrNotHosted = errors.New("match not hosted on this engine")

// CommandHandler is invoked once per delivered command message after the
// payload has been decoded.
type CommandHandler func(payload CommandPayload) error

// StartHandler is invoked once per delivered start message.
type StartHandler func(payload StartPayload) error

// ResyncHandler is invoked once per resync request.
type ResyncHandler func(payload ResyncPayload)

// SubscribeCommands attaches a durable JetStream consumer that delivers
// player commands to the handler. Returning ErrNotHosted causes the
// message to be NAK'd with a 1s delay; any other error NAKs with 2s.
func SubscribeCommands(ctx context.Context, logger *slog.Logger, js jetstream.JetStream, handler CommandHandler) error {
	cons, err := js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:       "engine-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: SubjCmdAll,
		MaxAckPending: 256,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}
	_, err = cons.Consume(func(msg jetstream.Msg) {
		var p CommandPayload
		if err := json.Unmarshal(msg.Data(), &p); err != nil {
			logger.Warn("bad cmd payload", "err", err)
			_ = msg.Term()
			return
		}
		if err := handler(p); err != nil {
			if errors.Is(err, ErrNotHosted) {
				_ = msg.NakWithDelay(time.Second)
				return
			}
			logger.Warn("submit failed", "err", err, "match", p.MatchID)
			_ = msg.NakWithDelay(2 * time.Second)
			return
		}
		_ = msg.Ack()
	})
	return err
}

// SubscribeStarts listens for new-match notifications. Core-api publishes
// one of these whenever a match transitions to active.
func SubscribeStarts(ctx context.Context, logger *slog.Logger, js jetstream.JetStream, handler StartHandler) error {
	cons, err := js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:       "engine-starter",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: SubjStartAll,
	})
	if err != nil {
		return fmt.Errorf("create starter consumer: %w", err)
	}
	_, err = cons.Consume(func(msg jetstream.Msg) {
		var p StartPayload
		if err := json.Unmarshal(msg.Data(), &p); err != nil {
			logger.Warn("bad start payload", "err", err)
			_ = msg.Term()
			return
		}
		if err := handler(p); err != nil {
			logger.Error("start handler failed", "err", err, "match", p.MatchID)
			_ = msg.Term()
			return
		}
		_ = msg.Ack()
	})
	return err
}

// SubscribeResyncs listens for client resync requests.
func SubscribeResyncs(_ context.Context, logger *slog.Logger, nc *nats.Conn, handler ResyncHandler) error {
	_, err := nc.Subscribe(SubjResyncAll, func(m *nats.Msg) {
		var p ResyncPayload
		if err := json.Unmarshal(m.Data, &p); err != nil {
			logger.Warn("bad resync payload", "err", err)
			return
		}
		handler(p)
	})
	return err
}
