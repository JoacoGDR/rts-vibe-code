package pgrepo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
)

// UpdateStatus moves a match from `from` to `to` in one statement so
// concurrent transitions cannot both succeed.
func (r *Matches) UpdateStatus(ctx context.Context, id uuid.UUID, from, to lobbydom.Status) error {
	tag, err := r.pool.Exec(ctx, `
        UPDATE matches SET status = $3
        WHERE id = $1 AND status = $2`, id, from, to)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return lobbydom.ErrInvalidTransition
	}
	return nil
}

// MarkStarting transitions waiting → starting and stamps started_at.
func (r *Matches) MarkStarting(ctx context.Context, id uuid.UUID) error {
	return r.UpdateStatus(ctx, id, lobbydom.StatusWaiting, lobbydom.StatusStarting)
}

// MarkActive transitions starting → active.
func (r *Matches) MarkActive(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE matches SET status = 'active', started_at = COALESCE(started_at, now())
        WHERE id = $1 AND status = 'starting'`, id)
	if err != nil {
		return err
	}
	return nil
}

// MarkAbandoned transitions active → abandoned.
func (r *Matches) MarkAbandoned(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE matches SET status = 'abandoned', ended_at = now()
        WHERE id = $1 AND status = 'active'`, id)
	return err
}

// RemovePlayer deletes a player row (waiting lobby leave / kick).
func (r *Matches) RemovePlayer(ctx context.Context, matchID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
        DELETE FROM match_players WHERE match_id = $1 AND user_id = $2`,
		matchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return lobbydom.ErrPlayerNotInMatch
	}
	return nil
}

// SetSlotAIControl updates controlled_by_ai for one slot.
func (r *Matches) SetSlotAIControl(ctx context.Context, matchID uuid.UUID, slot string, on bool) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE match_players SET controlled_by_ai = $3
        WHERE match_id = $1 AND slot = $2`, matchID, slot, on)
	return err
}

// ActiveMatches returns every match currently in active status.
func (r *Matches) ActiveMatches(ctx context.Context) ([]lobbydom.Match, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, name, map_id, status, created_by, created_at, started_at, ended_at, winner_user_id, speed_factor
        FROM matches WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []lobbydom.Match{}
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MatchAbandonedCandidate reports whether every alive human-controlled
// slot in the match has been silent longer than threshold. Returns the
// match id when true.
func (r *Matches) MatchAbandonedCandidate(ctx context.Context, matchID uuid.UUID, thresholdSeconds int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
        SELECT NOT EXISTS (
            SELECT 1 FROM match_players mp
            WHERE mp.match_id = $1
              AND mp.alive = TRUE
              AND mp.controlled_by_ai = FALSE
              AND (mp.last_seen_at IS NULL OR mp.last_seen_at >= now() - ($2 || ' seconds')::interval)
        )
        AND EXISTS (
            SELECT 1 FROM match_players mp2
            WHERE mp2.match_id = $1 AND mp2.alive = TRUE
        )`, matchID, thresholdSeconds).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return ok, err
}
