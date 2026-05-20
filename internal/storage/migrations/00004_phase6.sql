-- +goose Up
-- +goose StatementBegin
-- Phase 6: post-game stats and lifecycle hardening. Match status
-- values `starting` and `abandoned` are plain TEXT — no enum change.

CREATE TABLE IF NOT EXISTS match_stats (
    match_id        UUID PRIMARY KEY REFERENCES matches(id) ON DELETE CASCADE,
    duration_sec    INTEGER NOT NULL DEFAULT 0,
    total_units     INTEGER NOT NULL DEFAULT 0,
    total_combats   INTEGER NOT NULL DEFAULT 0,
    capitals_taken  INTEGER NOT NULL DEFAULT 0,
    per_slot        JSONB NOT NULL DEFAULT '{}'::jsonb,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS match_stats;
-- +goose StatementEnd
