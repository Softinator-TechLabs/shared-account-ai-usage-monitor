CREATE TABLE IF NOT EXISTS tm_reviews (
 id text PRIMARY KEY, session_id text NOT NULL REFERENCES tm_snapshots(id) ON DELETE CASCADE,
 ordinal integer NOT NULL, actor text NOT NULL, actor_kind text NOT NULL,
 kind text NOT NULL CHECK(kind IN('comment','prompt_rating','work_rating','analysis')),
 body text NOT NULL, score integer NOT NULL DEFAULT 0, evidence text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS tm_reviews_session ON tm_reviews(session_id,created_at,id);
