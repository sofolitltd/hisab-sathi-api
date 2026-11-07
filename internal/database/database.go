// Package database Neon Postgres-এ কানেকশন খোলে আর স্কিমা বসায়।
package database

import (
	"context"
	_ "embed"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// schema প্রতিবার চালু হওয়ার সময় চলে — তাই সব কমান্ড IF NOT EXISTS / idempotent।
//
//go:embed schema.sql
var schema string

// Open Neon-এর পুল করা কানেকশনে (pgbouncer) চলার জন্য simple protocol ব্যবহার করে।
func Open(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
