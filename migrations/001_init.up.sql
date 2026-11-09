CREATE TABLE IF NOT EXISTS links (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  code TEXT NOT NULL UNIQUE CHECK (length(code) BETWEEN 4 AND 32),
  destination TEXT NOT NULL CHECK (length(destination) BETWEEN 1 AND 2048),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- List query: WHERE (created_at,id) < cursor ORDER BY created_at DESC,id DESC.
CREATE INDEX IF NOT EXISTS links_created_cursor_idx ON links (created_at DESC, id DESC);
CREATE TABLE IF NOT EXISTS idempotency_keys (
  key TEXT PRIMARY KEY,
  response JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idempotency_expiry_idx ON idempotency_keys (expires_at);
CREATE TABLE IF NOT EXISTS outbox (
  event_id TEXT PRIMARY KEY,
  topic TEXT NOT NULL,
  message_key TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ,
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0)
);
CREATE INDEX IF NOT EXISTS outbox_pending_idx ON outbox (created_at) WHERE published_at IS NULL;
