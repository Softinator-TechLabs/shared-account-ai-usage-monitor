CREATE TABLE IF NOT EXISTS tm_native_quotas (
 workspace text NOT NULL,
 device text NOT NULL,
 event_id text NOT NULL,
 person text NOT NULL,
 profile text NOT NULL,
 observed_at timestamptz NOT NULL,
 received_at timestamptz NOT NULL DEFAULT now(),
 digest text NOT NULL,
 body jsonb NOT NULL,
 PRIMARY KEY(workspace,device,event_id)
);
CREATE INDEX IF NOT EXISTS tm_native_quota_recent ON tm_native_quotas(workspace,observed_at DESC);
