package storage

import (
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// openDBFromPool returns a *sql.DB sharing the pgxpool connection config.
// goose only needs a *sql.DB to drive migrations.
func openDBFromPool(pool *pgxpool.Pool) (*sql.DB, error) {
	cfg := pool.Config().ConnConfig
	connStr := stdlib.RegisterConnConfig(cfg)
	return sql.Open("pgx", connStr)
}
