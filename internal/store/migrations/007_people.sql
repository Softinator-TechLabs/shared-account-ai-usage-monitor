ALTER TABLE tm_members ADD COLUMN IF NOT EXISTS display_name text NOT NULL DEFAULT '';
ALTER TABLE tm_tokens ADD COLUMN IF NOT EXISTS device_os text NOT NULL DEFAULT '';
ALTER TABLE tm_tokens ADD COLUMN IF NOT EXISTS last_seen timestamptz;
CREATE TABLE IF NOT EXISTS tm_passwords(workspace text NOT NULL,person text NOT NULL,hash bytea NOT NULL,PRIMARY KEY(workspace,person),FOREIGN KEY(workspace,person) REFERENCES tm_members(workspace,person));
CREATE TABLE IF NOT EXISTS tm_login_attempts(workspace text NOT NULL,person text NOT NULL,attempts integer NOT NULL DEFAULT 0,window_start timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(workspace,person));
