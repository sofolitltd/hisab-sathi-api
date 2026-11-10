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

-- ক্রয়-বিক্রয়: প্রোডাক্ট (পণ্য বা মোবাইল ব্যাংকিং/কার্ড কমিশনের ফিক্সড-দাম আইটেম)।
CREATE TABLE IF NOT EXISTS products (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uid text NOT NULL,
  name text NOT NULL,
  buy_price double precision NOT NULL DEFAULT 0,
  sell_price double precision NOT NULL DEFAULT 0,
  updated_at bigint NOT NULL,
  deleted boolean NOT NULL DEFAULT false,
  seq bigint NOT NULL DEFAULT nextval('sync_seq'),
  PRIMARY KEY (user_id, uid)
);
ALTER TABLE products ADD COLUMN IF NOT EXISTS unit text NOT NULL DEFAULT 'piece';
CREATE INDEX IF NOT EXISTS idx_products_seq ON products(user_id, seq);

-- বিক্রয়: দামের স্ন্যাপশট রাখা হয় যাতে পরে প্রোডাক্টের দাম বদলালেও পুরনো লাভ হিসাব ঠিক থাকে।
-- customer_uid ফাঁকা থাকতে পারে (কোনো কাস্টমারের সাথে link না করলে)।
CREATE TABLE IF NOT EXISTS sales (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uid text NOT NULL,
  product_uid text NOT NULL,
  product_name text NOT NULL DEFAULT '',
  unit_buy_price double precision NOT NULL DEFAULT 0,
  unit_sell_price double precision NOT NULL DEFAULT 0,
  quantity double precision NOT NULL DEFAULT 1,
  customer_uid text NOT NULL DEFAULT '',
  received_amount double precision NOT NULL DEFAULT 0,
  profit double precision,
  date bigint NOT NULL,
  note text NOT NULL DEFAULT '',
  updated_at bigint NOT NULL,
  deleted boolean NOT NULL DEFAULT false,
  seq bigint NOT NULL DEFAULT nextval('sync_seq'),
  PRIMARY KEY (user_id, uid)
);
ALTER TABLE sales ADD COLUMN IF NOT EXISTS profit double precision;
ALTER TABLE sales ADD COLUMN IF NOT EXISTS discount double precision NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_sales_seq ON sales(user_id, seq);

-- ক্রয় (রিস্টক): কস্ট রেকর্ড করে, অ্যাপে এটাই পরের বিক্রয়ের buy_price ঠিক করে।
CREATE TABLE IF NOT EXISTS purchases (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uid text NOT NULL,
  product_uid text NOT NULL,
  quantity double precision NOT NULL DEFAULT 1,
  unit_cost double precision NOT NULL DEFAULT 0,
  date bigint NOT NULL,
  note text NOT NULL DEFAULT '',
  updated_at bigint NOT NULL,
  deleted boolean NOT NULL DEFAULT false,
  seq bigint NOT NULL DEFAULT nextval('sync_seq'),
  PRIMARY KEY (user_id, uid)
);
CREATE INDEX IF NOT EXISTS idx_purchases_seq ON purchases(user_id, seq);

-- ব্যয়: ক্যাটেগরি-ভিত্তিক সাধারণ ব্যয় (দোকান ভাড়া, খাবার, বাজার ইত্যাদি) — প্রোডাক্ট ক্রয়ের
-- (purchases) সাথে সম্পর্কহীন, স্বাধীন টেবিল।
CREATE TABLE IF NOT EXISTS expenses (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  uid text NOT NULL,
  category text NOT NULL,
  amount double precision NOT NULL DEFAULT 0,
  date bigint NOT NULL,
  note text NOT NULL DEFAULT '',
  updated_at bigint NOT NULL,
  deleted boolean NOT NULL DEFAULT false,
  seq bigint NOT NULL DEFAULT nextval('sync_seq'),
  PRIMARY KEY (user_id, uid)
);
CREATE INDEX IF NOT EXISTS idx_expenses_seq ON expenses(user_id, seq);
