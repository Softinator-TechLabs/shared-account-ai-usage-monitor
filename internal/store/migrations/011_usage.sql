CREATE TABLE IF NOT EXISTS tm_usage (
 workspace text NOT NULL, person text NOT NULL, device text NOT NULL, source_ref text NOT NULL,
 observed_at timestamptz NOT NULL, received_at timestamptz NOT NULL DEFAULT now(), body jsonb NOT NULL,
 PRIMARY KEY(workspace,person,device,source_ref)
);
CREATE INDEX IF NOT EXISTS tm_usage_sources ON tm_usage(workspace,source_ref);
CREATE TABLE IF NOT EXISTS tm_device_viewers (
 device_digest text PRIMARY KEY REFERENCES tm_tokens(digest) ON DELETE CASCADE,
 url text NOT NULL, encrypted_key bytea
);
