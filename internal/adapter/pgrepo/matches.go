package pgrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquing/clone-supremacy/internal/domain/lobbydom"
)

// Matches is the Postgres-backed implementation of the
// [lobbysvc.MatchesRepository] interface.
type Matches struct {
	pool *pgxpool.Pool
}

func NewMatches(pool *pgxpool.Pool) *Matches { return &Matches{pool: pool} }

func (r *Matches) Create(ctx context.Context, name, mapID string, createdBy uuid.UUID, speed float64) (lobbydom.Match, error) {
	id := uuid.New()
	row := r.pool.QueryRow(ctx, `
        INSERT INTO matches (id, name, map_id, status, created_by, speed_factor)
        VALUES ($1, $2, $3, 'waiting', $4, $5)
        RETURNING id, name, map_id, status, created_by, created_at, started_at, ended_at, winner_user_id, speed_factor`,
		id, name, mapID, createdBy, speed)
	return scanMatch(row)
}

func (r *Matches) Get(ctx context.Context, id uuid.UUID) (lobbydom.Match, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, name, map_id, status, created_by, created_at, started_at, ended_at, winner_user_id, speed_factor
        FROM matches WHERE id = $1`, id)
	m, err := scanMatch(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return lobbydom.Match{}, lobbydom.ErrNotFound
	}
	return m, err
}

func (r *Matches) ListForUser(ctx context.Context, userID uuid.UUID) ([]lobbydom.Match, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT m.id, m.name, m.map_id, m.status, m.created_by, m.created_at,
               m.started_at, m.ended_at, m.winner_user_id, m.speed_factor
        FROM matches m
        JOIN match_players mp ON mp.match_id = m.id
        WHERE mp.user_id = $1
        ORDER BY m.created_at DESC`, userID)
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

func (r *Matches) AddPlayer(ctx context.Context, p lobbydom.Player) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO match_players (match_id, user_id, slot, color, alive, controlled_by_ai)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (match_id, user_id) DO UPDATE SET
            slot = EXCLUDED.slot,
            color = EXCLUDED.color,
            controlled_by_ai = EXCLUDED.controlled_by_ai`,
		p.MatchID, p.UserID, p.Slot, p.Color, p.Alive, p.ControlledByAI)
	return err
}

func (r *Matches) ListPlayers(ctx context.Context, matchID uuid.UUID) ([]lobbydom.Player, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT match_id, user_id, slot, color, alive, controlled_by_ai, last_seen_at, joined_at
        FROM match_players WHERE match_id = $1
        ORDER BY slot`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []lobbydom.Player{}
	for rows.Next() {
		var p lobbydom.Player
		if err := rows.Scan(
			&p.MatchID, &p.UserID, &p.Slot, &p.Color, &p.Alive,
			&p.ControlledByAI, &p.LastSeenAt, &p.JoinedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// TouchPresence records that the worker observed a presence ping for
// the given match/slot. Best-effort: a failure here only delays AI
// takeover, never loses correctness.
func (r *Matches) TouchPresence(ctx context.Context, matchID uuid.UUID, slot string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE match_players SET last_seen_at = $3
        WHERE match_id = $1 AND slot = $2`, matchID, slot, at)
	return err
}

// MarkControlledByAI flips the ai-control flag for one slot. The bool is
// returned so the caller can detect the transition (true = it changed,
// false = it was already in the requested state) without an extra read.
func (r *Matches) MarkControlledByAI(ctx context.Context, matchID uuid.UUID, slot string, on bool) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
        UPDATE match_players SET controlled_by_ai = $3
        WHERE match_id = $1 AND slot = $2 AND controlled_by_ai <> $3`,
		matchID, slot, on)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SlotsNeedingTakeover returns every (matchID, slot) tuple whose
// last_seen_at is older than threshold and has not yet been flipped to
// AI control. Only active matches are considered.
func (r *Matches) SlotsNeedingTakeover(ctx context.Context, threshold time.Duration) ([]InactiveSlot, error) {
	if threshold <= 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
        SELECT mp.match_id, mp.user_id, mp.slot, mp.last_seen_at
        FROM match_players mp
        JOIN matches m ON m.id = mp.match_id
        WHERE m.status = 'active'
          AND mp.alive = TRUE
          AND mp.controlled_by_ai = FALSE
          AND mp.last_seen_at IS NOT NULL
          AND mp.last_seen_at < now() - ($1 || ' seconds')::interval
        ORDER BY mp.match_id, mp.slot`, int64(threshold.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InactiveSlot{}
	for rows.Next() {
		var s InactiveSlot
		if err := rows.Scan(&s.MatchID, &s.UserID, &s.Slot, &s.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// AISlots returns every slot currently flagged as AI-controlled. The
// ai-bot mode polls this to discover work.
func (r *Matches) AISlots(ctx context.Context) ([]InactiveSlot, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT mp.match_id, mp.user_id, mp.slot, mp.last_seen_at
        FROM match_players mp
        JOIN matches m ON m.id = mp.match_id
        WHERE m.status = 'active'
          AND mp.alive = TRUE
          AND mp.controlled_by_ai = TRUE
        ORDER BY mp.match_id, mp.slot`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InactiveSlot{}
	for rows.Next() {
		var s InactiveSlot
		if err := rows.Scan(&s.MatchID, &s.UserID, &s.Slot, &s.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// InactiveSlot is the read-model returned by [Matches.SlotsNeedingTakeover]
// and [Matches.AISlots]. UserID is the human player whose seat the bot
// will sit in (the bot logs in as a separate service account, but it
// preserves UserID so heartbeats and chat history still attribute to the
// original player).
type InactiveSlot struct {
	MatchID    uuid.UUID
	UserID     uuid.UUID
	Slot       string
	LastSeenAt *time.Time
}

func (r *Matches) MarkActive(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE matches SET status = 'active', started_at = now()
        WHERE id = $1 AND status = 'waiting'`, id)
	return err
}

func (r *Matches) MarkEnded(ctx context.Context, id uuid.UUID, winner *uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE matches SET status = 'ended', ended_at = now(), winner_user_id = $2
        WHERE id = $1`, id, winner)
	return err
}

// maxSnapshotSeq guards the uint64 -> int64 conversion below; Postgres'
// BIGINT column is signed, so we reject seqs that would wrap. In practice
// Phase 1's seq is a tick counter and will never come close.
const maxSnapshotSeq = uint64(1) << 62

func (r *Matches) SaveSnapshot(ctx context.Context, matchID uuid.UUID, seq uint64, payload any) error {
	if seq > maxSnapshotSeq {
		return fmt.Errorf("snapshot seq %d exceeds storage limit", seq)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
        INSERT INTO snapshots (match_id, seq, payload)
        VALUES ($1, $2, $3::jsonb)
        ON CONFLICT (match_id, seq) DO NOTHING`,
		matchID, int64(seq), string(raw))
	return err
}

func (r *Matches) LatestSnapshot(ctx context.Context, matchID uuid.UUID) ([]byte, uint64, error) {
	var raw []byte
	var seq int64
	err := r.pool.QueryRow(ctx, `
        SELECT payload, seq FROM snapshots
        WHERE match_id = $1
        ORDER BY taken_at DESC LIMIT 1`, matchID).Scan(&raw, &seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	if seq < 0 {
		return nil, 0, fmt.Errorf("invalid snapshot seq %d", seq)
	}
	return raw, uint64(seq), nil
}

func scanMatch(row rowScanner) (lobbydom.Match, error) {
	var m lobbydom.Match
	if err := row.Scan(&m.ID, &m.Name, &m.MapID, &m.Status, &m.CreatedBy, &m.CreatedAt,
		&m.StartedAt, &m.EndedAt, &m.WinnerUserID, &m.SpeedFactor); err != nil {
		return lobbydom.Match{}, err
	}
	return m, nil
}
