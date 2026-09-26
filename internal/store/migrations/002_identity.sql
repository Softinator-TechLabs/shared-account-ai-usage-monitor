CREATE TABLE IF NOT EXISTS tm_members (
 workspace text NOT NULL, person text NOT NULL, role text NOT NULL CHECK(role IN('owner','manager','member')),
 oidc_subject text, active boolean NOT NULL DEFAULT true, PRIMARY KEY(workspace,person), UNIQUE(workspace,oidc_subject)
);
CREATE TABLE IF NOT EXISTS tm_invites (
 digest text PRIMARY KEY, workspace text NOT NULL, person text NOT NULL, kind text NOT NULL,
 expires_at timestamptz NOT NULL, consumed boolean NOT NULL DEFAULT false
);
CREATE TABLE IF NOT EXISTS tm_tokens (
 digest text PRIMARY KEY, workspace text NOT NULL, person text NOT NULL, kind text NOT NULL,
 device text NOT NULL DEFAULT '', ack_version integer NOT NULL DEFAULT 0,
 enrolled_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL
);
