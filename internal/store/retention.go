package store

import (
	"context"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
)

// Expire uses archive receipt time, not a guessed time of human work. It deletes
// all revisions and their attached discussions once a source is wholly expired.
func (s *Store) Expire(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, `SELECT s.workspace,s.source_ref FROM tm_snapshots s JOIN tm_policies p ON p.workspace=s.workspace WHERE (p.body->>'retention_days')::int>0 GROUP BY s.workspace,s.source_ref,p.body HAVING max(s.received_at)<now()-make_interval(days => (p.body->>'retention_days')::int)`)
	if e != nil {
		return e
	}
	var refs [][2]string
	for rows.Next() {
		var r [2]string
		if e = rows.Scan(&r[0], &r[1]); e != nil {
			rows.Close()
			return e
		}
		refs = append(refs, r)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, r := range refs {
		if e = s.DeleteSource(ctx, c.Principal{Workspace: r[0], Person: "retention", Role: "owner"}, r[1]); e != nil {
			return e
		}
	}
	_, e = s.DB.Exec(ctx, "DELETE FROM tm_native_quotas q USING tm_policies p WHERE q.workspace=p.workspace AND (p.body->>'retention_days')::int>0 AND q.received_at<now()-make_interval(days => (p.body->>'retention_days')::int); DELETE FROM tm_chunks WHERE created_at<now()-interval '1 day'; DELETE FROM tm_tokens WHERE expires_at<now(); DELETE FROM tm_invites WHERE expires_at<now(); DELETE FROM tm_analysis_runs WHERE expires_at<now(); DELETE FROM tm_login_attempts WHERE window_start<now()-interval '1 day'")
	return e
}
func (s *Store) DeletionLedger(ctx context.Context, p c.Principal) ([]string, error) {
	if p.Role != "owner" || p.Device != "" {
		return nil, c.ErrForbidden
	}
	rows, e := s.DB.Query(ctx, "SELECT source_ref FROM tm_deleted WHERE workspace=$1 ORDER BY source_ref", p.Workspace)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var ref string
		if e = rows.Scan(&ref); e != nil {
			return nil, e
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}
func (s *Store) RestoreDeletions(ctx context.Context, p c.Principal, refs []string) error {
	if p.Role != "owner" || p.Device != "" {
		return c.ErrForbidden
	}
	for _, ref := range refs {
		if e := s.DeleteSource(ctx, p, ref); e != nil {
			return e
		}
	}
	return nil
}
