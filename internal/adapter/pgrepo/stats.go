package pgrepo

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MatchStatsRow is the persisted post-game summary.
type MatchStatsRow struct {
	MatchID        uuid.UUID
	DurationSec    int
	TotalUnits     int
	TotalCombats   int
	CapitalsTaken  int
	PerSlot        map[string]any
}

// StatsRepo persists match_stats rows.
type StatsRepo struct {
	pool *pgxpool.Pool
}

func NewStatsRepo(pool *pgxpool.Pool) *StatsRepo { return &StatsRepo{pool: pool} }

func (r *StatsRepo) Upsert(ctx context.Context, row MatchStatsRow) error {
	raw, err := json.Marshal(row.PerSlot)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
        INSERT INTO match_stats (match_id, duration_sec, total_units, total_combats, capitals_taken, per_slot)
        VALUES ($1, $2, $3, $4, $5, $6::jsonb)
        ON CONFLICT (match_id) DO UPDATE SET
            duration_sec = EXCLUDED.duration_sec,
            total_units = EXCLUDED.total_units,
            total_combats = EXCLUDED.total_combats,
            capitals_taken = EXCLUDED.capitals_taken,
            per_slot = EXCLUDED.per_slot,
            recorded_at = now()`,
		row.MatchID, row.DurationSec, row.TotalUnits, row.TotalCombats, row.CapitalsTaken, string(raw))
	return err
}

func (r *StatsRepo) Get(ctx context.Context, matchID uuid.UUID) (MatchStatsRow, error) {
	var row MatchStatsRow
	var raw []byte
	err := r.pool.QueryRow(ctx, `
        SELECT match_id, duration_sec, total_units, total_combats, capitals_taken, per_slot
        FROM match_stats WHERE match_id = $1`, matchID).Scan(
		&row.MatchID, &row.DurationSec, &row.TotalUnits, &row.TotalCombats, &row.CapitalsTaken, &raw)
	if err != nil {
		return MatchStatsRow{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &row.PerSlot)
	}
	if row.PerSlot == nil {
		row.PerSlot = map[string]any{}
	}
	return row, nil
}
