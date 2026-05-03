-- +goose Up
-- +goose StatementBegin
-- Phase 4: chat persistence. Diplomacy state lives in-memory on the
-- engine (it is recoverable from snapshots + the JetStream command log)
-- so it intentionally has no Postgres footprint here. The AI takeover
-- and notification tables described in the plan land in their own
-- migration in Phase 5.

CREATE TABLE chat_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id        UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    scope           TEXT NOT NULL,
    author_user_id  UUID NOT NULL REFERENCES users(id),
    author_slot     TEXT NOT NULL,
    body            TEXT NOT NULL,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX chat_messages_scope_sent_idx
    ON chat_messages(scope, sent_at DESC);

CREATE INDEX chat_messages_match_idx
    ON chat_messages(match_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS chat_messages_match_idx;
DROP INDEX IF EXISTS chat_messages_scope_sent_idx;
DROP TABLE IF EXISTS chat_messages;
-- +goose StatementEnd
