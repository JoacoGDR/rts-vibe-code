-- +goose Up
-- +goose StatementBegin
-- Phase 5: AI takeover, inactivity tracking, notifications.
--   * users.is_bot               — distinguishes pre-seeded bot accounts
--                                  from real players, lets ai-bot session
--                                  pool hand out service identities.
--   * match_players.controlled_by_ai — flipped by the worker once a slot
--                                  has missed enough heartbeats.
--   * match_players.last_seen_at — refreshed by the worker as it consumes
--                                  match.<id>.heartbeat.<slot>.
--   * notifications              — persistent queue for the bell + SMTP
--                                  worker. created_at is the natural
--                                  paging axis; sent_at = NULL means the
--                                  email has not been delivered yet.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_bot BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE match_players
    ADD COLUMN IF NOT EXISTS controlled_by_ai BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS last_seen_at     TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS users_is_bot_idx
    ON users(is_bot) WHERE is_bot = TRUE;

-- Seed pool of bot identities. Passwords are intentionally empty —
-- bots only ever authenticate via POST /auth/login-as-bot, which
-- bypasses the bcrypt check after verifying is_bot=true. The display
-- names mirror what `internal/aibot/seed.go::defaultBotDisplay` would
-- pick, so devs see consistent labels everywhere.
INSERT INTO users (email, password_hash, display_name, color, is_bot) VALUES
    ('bot-01@bots.supremacy.local', '', 'bot-01', '#7d8da1', TRUE),
    ('bot-02@bots.supremacy.local', '', 'bot-02', '#a78bfa', TRUE),
    ('bot-03@bots.supremacy.local', '', 'bot-03', '#fbbf24', TRUE),
    ('bot-04@bots.supremacy.local', '', 'bot-04', '#34d399', TRUE),
    ('bot-05@bots.supremacy.local', '', 'bot-05', '#f472b6', TRUE),
    ('bot-06@bots.supremacy.local', '', 'bot-06', '#60a5fa', TRUE)
ON CONFLICT (email) DO NOTHING;

CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_id    UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at     TIMESTAMPTZ,
    read_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS notifications_user_created_idx
    ON notifications(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS notifications_unsent_idx
    ON notifications(created_at)
    WHERE sent_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS notifications_unsent_idx;
DROP INDEX IF EXISTS notifications_user_created_idx;
DROP TABLE IF EXISTS notifications;
DROP INDEX IF EXISTS users_is_bot_idx;
ALTER TABLE match_players
    DROP COLUMN IF EXISTS last_seen_at,
    DROP COLUMN IF EXISTS controlled_by_ai;
ALTER TABLE users
    DROP COLUMN IF EXISTS is_bot;
-- +goose StatementEnd
