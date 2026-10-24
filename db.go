package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE SEQUENCE IF NOT EXISTS sync_seq;

-- লগইন: মোবাইল নম্বর + ৫ ডিজিটের PIN (শুধু PIN-এর bcrypt হ্যাশ থাকে)।
CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  phone text,
  pin_hash text,
  failed_attempts int NOT NULL DEFAULT 0,
  locked_until timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- আগের ইমেইল/পাসওয়ার্ড সংস্করণ থেকে সরানো (শুধু পরীক্ষার ইউজার ছিল)।
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name = 'users' AND column_name = 'email') THEN
    DELETE FROM users WHERE phone IS NULL;
    ALTER TABLE users DROP COLUMN email;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_name = 'users' AND column_name = 'password_hash') THEN
    ALTER TABLE users RENAME COLUMN password_hash TO pin_hash;
  END IF;
END $$;
ALTER TABLE users ALTER COLUMN phone SET NOT NULL;
ALTER TABLE users ALTER COLUMN pin_hash SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

CREATE TABLE IF NOT EXISTS customers (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uid text NOT NULL,
  ledger text NOT NULL DEFAULT 'pabo',
  name text NOT NULL,
  phone text NOT NULL DEFAULT '',
  address text NOT NULL DEFAULT '',
  due_date bigint,
  reminder_at bigint,
  created_at bigint NOT NULL,
  updated_at bigint NOT NULL,
  deleted boolean NOT NULL DEFAULT false,
  seq bigint NOT NULL DEFAULT nextval('sync_seq'),
  PRIMARY KEY (user_id, uid)
);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS photo text NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS extra_phones text NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS plan text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_customers_seq ON customers(user_id, seq);

CREATE TABLE IF NOT EXISTS entries (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uid text NOT NULL,
  customer_uid text NOT NULL,
  type text NOT NULL,
  amount double precision NOT NULL,
  date bigint NOT NULL,
  note text NOT NULL DEFAULT '',
  updated_at bigint NOT NULL,
  deleted boolean NOT NULL DEFAULT false,
  seq bigint NOT NULL DEFAULT nextval('sync_seq'),
  PRIMARY KEY (user_id, uid)
);
CREATE INDEX IF NOT EXISTS idx_entries_seq ON entries(user_id, seq);

CREATE TABLE IF NOT EXISTS contact_logs (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uid text NOT NULL,
  customer_uid text NOT NULL,
  kind text NOT NULL,
  note text NOT NULL DEFAULT '',
  at bigint NOT NULL,
  updated_at bigint NOT NULL,
  deleted boolean NOT NULL DEFAULT false,
  seq bigint NOT NULL DEFAULT nextval('sync_seq'),
  PRIMARY KEY (user_id, uid)
);
CREATE INDEX IF NOT EXISTS idx_contact_logs_seq ON contact_logs(user_id, seq);
`

// openDB Neon-এর পুল করা কানেকশনে (pgbouncer) চলার জন্য simple protocol ব্যবহার করে।
func openDB(ctx context.Context) (*pgxpool.Pool, error) {
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
