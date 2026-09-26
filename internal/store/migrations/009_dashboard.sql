CREATE TABLE IF NOT EXISTS tm_snapshot_metrics (
 snapshot_id text PRIMARY KEY REFERENCES tm_snapshots(id) ON DELETE CASCADE,
 body jsonb NOT NULL
);
