CREATE TABLE IF NOT EXISTS tm_policies (
 workspace text PRIMARY KEY, version integer NOT NULL CHECK(version > 0), body jsonb NOT NULL
);
CREATE TABLE IF NOT EXISTS tm_snapshots (
 id text PRIMARY KEY, workspace text NOT NULL, source_ref text NOT NULL, revision text NOT NULL,
 owner_id text NOT NULL, device_id text NOT NULL, attribution text NOT NULL,
 digest text NOT NULL, body jsonb NOT NULL, received_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(workspace,source_ref,revision)
);
CREATE INDEX IF NOT EXISTS tm_snapshots_workspace ON tm_snapshots(workspace,received_at DESC,id);
CREATE TABLE IF NOT EXISTS tm_audit (
 id bigserial PRIMARY KEY, workspace text NOT NULL, actor text NOT NULL, action text NOT NULL,
 resource text NOT NULL, happened_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS tm_deleted (
 workspace text NOT NULL, source_ref text NOT NULL, deleted_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(workspace, source_ref)
);
