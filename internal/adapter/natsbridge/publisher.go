package natsbridge

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes commands, start notifications and (per-slot or
// public) state/event envelopes. Both the gateway (commands) and the
// engine (state/events/start) use this type.
type Publisher struct {
	JS jetstream.JetStream
	NC *nats.Conn
}

// PublishCommand asks the engine to run a player command.
func (p *Publisher) PublishCommand(ctx context.Context, payload CommandPayload) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := p.JS.Publish(ctx, CmdSubject(payload.MatchID), raw); err != nil {
		return err
	}
	return nil
}

// PublishStart asks an engine to host a new match.
func (p *Publisher) PublishStart(ctx context.Context, payload StartPayload) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := p.JS.Publish(ctx, StartSubject(payload.MatchID), raw); err != nil {
		return err
	}
	return nil
}

// PublishResync forwards a client resync request to the engine.
func (p *Publisher) PublishResync(_ context.Context, matchID string) error {
	body, err := json.Marshal(ResyncPayload{MatchID: matchID})
	if err != nil {
		return err
	}
	return p.NC.Publish(ResyncSubject(matchID), body)
}

// PublishPublicState fan-outs the un-filtered state to lobby observers.
func (p *Publisher) PublishPublicState(_ context.Context, matchID string, payload []byte) error {
	return p.NC.Publish(PublicStateSubject(matchID), payload)
}

// PublishSlotState fan-outs a per-slot filtered state.
func (p *Publisher) PublishSlotState(_ context.Context, matchID, slot string, payload []byte) error {
	return p.NC.Publish(SlotStateSubject(matchID, slot), payload)
}

// PublishPublicEvent fan-outs an un-filtered event.
func (p *Publisher) PublishPublicEvent(_ context.Context, matchID string, payload []byte) error {
	return p.NC.Publish(PublicEventSubject(matchID), payload)
}

// PublishSlotEvent fan-outs a per-slot filtered event.
func (p *Publisher) PublishSlotEvent(_ context.Context, matchID, slot string, payload []byte) error {
	return p.NC.Publish(SlotEventSubject(matchID, slot), payload)
}

// LookupUserSlot is exported for callers (gateway) that need to resolve a
// user's slot before subscribing to per-slot subjects. The lookup itself
// lives in the redisrepo adapter; this is a re-export trampoline so the
// gateway only needs to import natsbridge for everything message-bus.
//
// We keep a separate redisrepo.LookupUserSlot for tests that don't need a
// publisher.
