package store

import (
         "context"

         "github.com/jackc/pgx/v5/pgxpool"
     )


type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, connString string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &DB{Pool: pool}, nil
}
