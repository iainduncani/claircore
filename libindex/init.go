package libindex

import (
	"context"
	"fmt"

	"github.com/quay/claircore/internal/indexer"
	"github.com/quay/claircore/internal/indexer/postgres"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jmoiron/sqlx"
)

// initialize a indexer.Store given libindex.Opts
func initStore(ctx context.Context, opts *Opts) (*sqlx.DB, indexer.Store, error) {
	// we are going to use pgx for more control over connection pool and
	// and a cleaner api around bulk inserts
	cfg, err := pgxpool.ParseConfig(opts.ConnString)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse ConnString: %v", err)
	}
	cfg.MaxConns = 30
	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ConnPool: %v", err)
	}

	// setup sqlx
	db, err := sqlx.Open("pgx", opts.ConnString)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open db: %v", err)
	}

	store := postgres.NewStore(db, pool)
	return db, store, nil
}