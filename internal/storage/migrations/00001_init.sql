-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    color           TEXT NOT NULL DEFAULT '#5fa8ff',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at   TIMESTAMPTZ
);

CREATE TABLE matches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    map_id          TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'waiting', -- waiting | active | ended
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    winner_user_id  UUID REFERENCES users(id),
    speed_factor    DOUBLE PRECISION NOT NULL DEFAULT 60.0
);

CREATE INDEX matches_status_idx ON matches(status);

CREATE TABLE match_players (
    match_id    UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id),
    slot        TEXT NOT NULL,            -- e.g. "red" / "blue"
    color       TEXT NOT NULL,
    alive       BOOLEAN NOT NULL DEFAULT TRUE,
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, user_id)
);

CREATE INDEX match_players_user_idx ON match_players(user_id);

CREATE TABLE snapshots (
    id          BIGSERIAL PRIMARY KEY,
    match_id    UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    seq         BIGINT NOT NULL,           -- last applied JetStream cmd seq
    taken_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload     JSONB NOT NULL,
    UNIQUE (match_id, seq)
);

CREATE INDEX snapshots_match_taken_idx ON snapshots(match_id, taken_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS match_players;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
