package store

import (
	"context"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
	"time"
)

func TestDashboardDeduplicatesCapturesAndEnforcesVisibility(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	v := fixture()
	v.StartedAt = "2026-09-25T20:00:00Z"
	v.Messages = []c.Message{{Ordinal: 0, Role: "user", Content: "synthetic"}, {Ordinal: 1, Role: "assistant", Content: "synthetic", Model: "fixture-model"}}
	if _, e := s.Accept(ctx, alice(), v); e != nil {
		t.Fatal(e)
	}
	v.Revision = "longer"
	v.Messages = append(v.Messages, c.Message{Ordinal: 2, Role: "assistant", Content: "synthetic"})
	row, e := s.Accept(ctx, alice(), v)
	if e != nil {
		t.Fatal(e)
	}
	v.Revision = "late-shorter"
	v.Messages = v.Messages[:1]
	s.Accept(ctx, alice(), v)
	now := time.Date(2026, 9, 26, 5, 0, 0, 0, time.UTC)
	d, e := s.Dashboard(ctx, owner(), 14, now)
	if e != nil {
		t.Fatal(e)
	}
	if d.Sessions != 1 || d.Prompts != 1 || d.Models["fixture-model"] != 1 || d.Models["Unknown"] != 1 || d.Days["2026-09-26"] != 1 {
		t.Fatalf("duplicate/invented evidence: %+v", d)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "self_managers"})
	d, e = s.Dashboard(ctx, c.Principal{Workspace: "team", Person: "bob", Role: "member"}, 14, now)
	if e != nil || d.Sessions != 0 || d.IndexedSources != 0 {
		t.Fatal("cross-person aggregate leak", d, e)
	}
	// An existing archive without metrics can be upgraded without importing again.
	s.DB.Exec(ctx, "DELETE FROM tm_snapshot_metrics WHERE snapshot_id=$1", row.ID)
	if e = s.BackfillMetrics(ctx, 10); e != nil {
		t.Fatal(e)
	}
	d, e = s.Dashboard(ctx, owner(), 14, now)
	if e != nil || d.Models["Unknown"] != 1 {
		t.Fatal("backfill", d, e)
	}
	s.DeleteSource(ctx, owner(), row.SourceRef)
	var n int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM tm_snapshot_metrics").Scan(&n)
	if n != 0 {
		t.Fatal("deleted content left derived metrics", n)
	}
}
