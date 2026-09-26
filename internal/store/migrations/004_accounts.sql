CREATE TABLE IF NOT EXISTS tm_accounts(workspace text NOT NULL,alias text NOT NULL,body jsonb NOT NULL,updated_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(workspace,alias));
CREATE TABLE IF NOT EXISTS tm_quotas(id bigserial PRIMARY KEY,workspace text NOT NULL,account text NOT NULL,body jsonb NOT NULL,observed_at timestamptz NOT NULL,FOREIGN KEY(workspace,account) REFERENCES tm_accounts(workspace,alias) ON DELETE CASCADE);
CREATE INDEX IF NOT EXISTS tm_quotas_account ON tm_quotas(workspace,account,observed_at DESC);
