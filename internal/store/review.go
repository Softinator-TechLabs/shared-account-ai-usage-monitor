package store

import (
	"context"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"time"
)

type Review struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Ordinal   int       `json:"ordinal"`
	Kind      string    `json:"kind"`
	Body      string    `json:"body"`
	Score     int       `json:"score"`
	Evidence  string    `json:"evidence"`
	ActorKind string    `json:"actor_kind"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) AddReview(ctx context.Context, p c.Principal, r Review) (Review, error) {
	if p.Device != "" {
		return r, c.ErrForbidden
	}
	session, err := s.Get(ctx, p, r.SessionID)
	if err != nil {
		return r, err
	}
	found := false
	for _, m := range session.Messages {
		if m.Ordinal == r.Ordinal {
			found = true
		}
	}
	if !found || len(r.Body) == 0 || len(r.Body) > 100000 || r.ActorKind != "human" {
		return r, c.ErrInvalid
	}
	switch r.Kind {
	case "comment":
		if r.Score != 0 {
			return r, c.ErrInvalid
		}
	case "prompt_rating", "work_rating":
		if r.Score < 1 || r.Score > 5 {
			return r, c.ErrInvalid
		}
		if r.Kind == "work_rating" && r.Evidence == "" {
			return r, c.ErrInvalid
		}
	default:
		return r, c.ErrInvalid
	}
	r.ID = ID()
	r.Actor = p.Person
	err = s.DB.QueryRow(ctx, `INSERT INTO tm_reviews(id,session_id,ordinal,actor,actor_kind,kind,body,score,evidence) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING created_at`, r.ID, r.SessionID, r.Ordinal, r.Actor, r.ActorKind, r.Kind, r.Body, r.Score, r.Evidence).Scan(&r.CreatedAt)
	return r, err
}
func (s *Store) Reviews(ctx context.Context, p c.Principal, id string) ([]Review, error) {
	if _, err := s.Get(ctx, p, id); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `SELECT id,session_id,ordinal,actor,actor_kind,kind,body,score,evidence,created_at FROM tm_reviews WHERE session_id=$1 ORDER BY created_at,id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Review{}
	for rows.Next() {
		var r Review
		if err = rows.Scan(&r.ID, &r.SessionID, &r.Ordinal, &r.Actor, &r.ActorKind, &r.Kind, &r.Body, &r.Score, &r.Evidence, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Store) DeleteSource(ctx context.Context, p c.Principal, source string) error {
	if p.Role != "owner" || p.Device != "" || source == "" {
		return c.ErrForbidden
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Serialize with ingest policy read locks so deletion cannot race an accepted snapshot.
	if _, err = tx.Exec(ctx, "SELECT workspace FROM tm_policies WHERE workspace=$1 FOR UPDATE", p.Workspace); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO tm_deleted(workspace,source_ref) VALUES($1,$2) ON CONFLICT DO NOTHING", p.Workspace, source); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM tm_snapshots WHERE workspace=$1 AND source_ref=$2", p.Workspace, source); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO tm_audit(workspace,actor,action,resource) VALUES($1,$2,'source.delete',$3)", p.Workspace, p.Person, source); err != nil {
		return err
	}
	// Unassembled chunks cannot be assigned safely to one source, so clear this
	// workspace's staging area; device queues retain unacknowledged originals.
	if _, err = tx.Exec(ctx, "DELETE FROM tm_chunks WHERE workspace=$1", p.Workspace); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
