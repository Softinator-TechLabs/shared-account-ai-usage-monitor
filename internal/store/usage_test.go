package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
	"time"
)

func amount(n int64) *int64 { return &n }
func TestUsageRevisionsDatesVisibilityAndDeletion(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	p := alice()
	v := c.UsageCapture{SourceRef: "synthetic", Revision: "r1", PolicyVersion: 1, Client: "claude", Project: "repo", ObservedAt: now, Coverage: "reported", Points: []c.UsagePoint{{Timestamp: now.Format(time.RFC3339), Model: "model", InputTokens: amount(10), OutputTokens: amount(2)}}}
	if e := s.ObserveUsage(ctx, p, v); e != nil {
		t.Fatal(e)
	}
	if e := s.ObserveUsage(ctx, p, v); e != nil {
		t.Fatal(e)
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Days: 14}, now)
	if e != nil || d.Totals.OutputTokens == nil || *d.Totals.OutputTokens != 2 || d.Totals.CacheReadTokens != nil {
		t.Fatal(d, e)
	}
	v.Revision = "r2"
	v.ObservedAt = now.Add(time.Second)
	v.Points[0].OutputTokens = amount(5)
	if e = s.ObserveUsage(ctx, p, v); e != nil {
		t.Fatal(e)
	}
	d, e = s.Analytics(ctx, owner(), UsageFilter{Days: 14}, now)
	if e != nil || *d.Totals.OutputTokens != 5 {
		t.Fatal("summed revisions", d, e)
	}
	v.Points = append(v.Points, c.UsagePoint{Timestamp: "", OutputTokens: amount(100)})
	v.ObservedAt = now.Add(2 * time.Second)
	s.ObserveUsage(ctx, p, v)
	d, e = s.Analytics(ctx, owner(), UsageFilter{Days: 14}, now)
	if e != nil || *d.Totals.OutputTokens != 5 || d.Coverage.UndatedPoints != 1 {
		t.Fatal("dated allocation invented", d, e)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "self_managers"})
	d, e = s.Analytics(ctx, c.Principal{Workspace: "team", Person: "bob", Role: "member"}, UsageFilter{Days: 14, Person: "alice"}, now)
	if !errors.Is(e, c.ErrForbidden) {
		t.Fatal("scope bypass", d, e)
	}
	if e = s.DeleteSource(ctx, owner(), "synthetic"); e != nil {
		t.Fatal(e)
	}
	v.PolicyVersion = 2
	if e = s.ObserveUsage(ctx, p, v); !errors.Is(e, c.ErrDeleted) {
		t.Fatal("resurrected", e)
	}
	d, e = s.Analytics(ctx, owner(), UsageFilter{Days: 14}, now)
	if e != nil || d.Coverage.Sources != 0 {
		t.Fatal(d, e)
	}
}
func TestUsageCopiedSourcesDoNotDoubleCountOrInventPerson(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	v := c.UsageCapture{SourceRef: "copied", Revision: "r", PolicyVersion: 1, Client: "codex", Project: "repo", ObservedAt: now, Coverage: "reported", Points: []c.UsagePoint{{Timestamp: now.Format(time.RFC3339), OutputTokens: amount(4)}}}
	for _, person := range []string{"alice", "bob"} {
		if e := s.ObserveUsage(ctx, c.Principal{Workspace: "team", Person: person, Device: "mac"}, v); e != nil {
			t.Fatal(e)
		}
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Days: 14}, now)
	if e != nil || *d.Totals.OutputTokens != 4 || d.Coverage.AttributionConflicts != 1 || len(d.People) != 0 {
		t.Fatal(d, e)
	}
	d, e = s.Analytics(ctx, owner(), UsageFilter{Days: 14, Person: "alice"}, now)
	if e != nil || d.Totals.OutputTokens != nil {
		t.Fatal("assigned conflicting source", d, e)
	}
}

func TestUsageUnchangedReconciliationExpiresAndDayBoundary(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	local, _ := time.LoadLocation("Asia/Kolkata")
	day := now.In(local)
	boundary := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, local)
	v := c.UsageCapture{SourceRef: "expiry", Revision: "1", PolicyVersion: 1, Client: "codex", ObservedAt: now, Coverage: "reported", Points: []c.UsagePoint{{Timestamp: boundary.Add(-time.Second).Format(time.RFC3339), OutputTokens: amount(2)}, {Timestamp: boundary.Format(time.RFC3339), OutputTokens: amount(0)}}}
	if e := s.ObserveUsage(ctx, alice(), v); e != nil {
		t.Fatal(e)
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Days: 7}, now)
	if e != nil || *d.Daily[5].OutputTokens != 2 || d.Daily[6].OutputTokens == nil || *d.Daily[6].OutputTokens != 0 {
		t.Fatal(d, e)
	}
	s.DB.Exec(ctx, "UPDATE tm_usage SET received_at=now()-interval '100 days'")
	v.ObservedAt = now.Add(time.Second)
	v.Revision = "2"
	if e = s.ObserveUsage(ctx, alice(), v); e != nil {
		t.Fatal(e)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "team", RetentionDays: 90})
	if e = s.Expire(ctx); e != nil {
		t.Fatal(e)
	}
	d, e = s.Analytics(ctx, owner(), UsageFilter{Days: 7}, now)
	if e != nil || d.Coverage.Sources != 0 {
		t.Fatal("unchanged reconciliation pinned source", d, e)
	}
}

func TestUsageConflictingCopyExcludedAndUnknownProject(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	v := c.UsageCapture{SourceRef: "copies", Revision: "r1", PolicyVersion: 1, Client: "codex", ObservedAt: now, Coverage: "reported", Points: []c.UsagePoint{{Timestamp: now.Format(time.RFC3339), OutputTokens: amount(4)}}}
	p := alice()
	s.ObserveUsage(ctx, p, v)
	d, e := s.Analytics(ctx, owner(), UsageFilter{Days: 7, Project: "Unknown project"}, now)
	if e != nil || *d.Totals.OutputTokens != 4 {
		t.Fatal(d, e)
	}
	p.Device = "old-copy"
	v.Points[0].OutputTokens = amount(2)
	v.ObservedAt = now.Add(time.Second)
	s.ObserveUsage(ctx, p, v)
	d, e = s.Analytics(ctx, owner(), UsageFilter{Days: 7}, now)
	if e != nil || d.Totals.OutputTokens != nil || d.Coverage.ContentConflicts != 1 {
		t.Fatal("stale copy became authority", d, e)
	}
}

func TestUsageSecretPolicyEnforcedAtStore(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "secrets", Visibility: "team"})
	secret := "ghp_abcdefghijklmnopqrstuvwxyz1234567890"
	v := c.UsageCapture{SourceRef: "policy-source", Revision: "1", PolicyVersion: 2, Client: "codex", Project: secret, ObservedAt: now, Coverage: "reported", Points: []c.UsagePoint{{Timestamp: now.Format(time.RFC3339), Model: secret, OutputTokens: amount(3)}}}
	if e := s.ObserveUsage(ctx, alice(), v); e != nil {
		t.Fatal(e)
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Days: 7}, now)
	if e != nil || len(d.Projects) != 1 || d.Projects[0].Project == secret || d.Models[0].Model == secret {
		t.Fatal("policy bypass", d, e)
	}
}
