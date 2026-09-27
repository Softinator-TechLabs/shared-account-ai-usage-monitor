package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestUsageRollingHoursExactBoundsAndBuckets(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 26, 16, 7, 0, 0, time.UTC)
	for _, hours := range []int{1, 12, 24, 48} {
		start := now.Add(-time.Duration(hours) * time.Hour)
		v := c.UsageCapture{SourceRef: "rolling", Revision: "1", PolicyVersion: 1, Client: "claude", ObservedAt: time.Now().UTC(), Coverage: "reported", Points: []c.UsagePoint{}}
		for _, at := range []time.Time{start.Add(-time.Nanosecond), start, start.Add(time.Minute), now.Add(-time.Nanosecond), now, now.Add(time.Minute)} {
			v.Points = append(v.Points, c.UsagePoint{Timestamp: at.Format(time.RFC3339Nano), OutputTokens: amount(1)})
		}
		if e := s.ObserveUsage(ctx, alice(), v); e != nil {
			t.Fatal(e)
		}
		d, e := s.Analytics(ctx, owner(), UsageFilter{Hours: hours}, now)
		want := hours
		if hours == 1 {
			want = 12
		}
		if e != nil || d.Totals.OutputTokens == nil || *d.Totals.OutputTokens != 3 || len(d.Series) != want || len(d.Daily) != 0 {
			t.Fatalf("hours=%d data=%+v err=%v", hours, d, e)
		}
		if d.Start != start.Format(time.RFC3339Nano) || d.End != now.Format(time.RFC3339Nano) {
			t.Fatal("not rolling bounds", d.Start, d.End)
		}
	}
	for _, f := range []UsageFilter{{Hours: 2}, {Hours: 24, Days: 7}, {Hours: -1}} {
		if _, e := s.Analytics(ctx, owner(), f, now); !errors.Is(e, c.ErrInvalid) {
			t.Fatal("invalid period accepted", f, e)
		}
	}
}

// Old missing-usage history must not contaminate unrelated observed intervals;
// incomplete/conflicting source bounds must never hide a possible denominator.
func TestUsageQuotaBlockersRespectAllSourceTimeBounds(t *testing.T) {
	for _, tc := range []struct {
		name, kind string
		available  bool
	}{
		{"old_unavailable", "unavailable", true},
		{"old_undated", "undated", true},
		{"old_conflicts", "conflict", true},
		{"overlapping_unavailable", "overlap", false},
		{"missing_end", "missing_end", false},
		{"invalid_end", "invalid_end", false},
		{"inverted_bounds", "inverted", false},
		{"dated_point_outside_old_bounds", "outside_point", false},
		{"activity_outside_old_bounds", "outside_activity", false},
		{"older_copy_overlaps", "conflict_overlap", false},
		{"older_copy_unknown", "conflict_unknown", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			obs, points, from, to := quotaEstimateFixture()
			for i, o := range obs {
				o.EventID = fmt.Sprintf("event-%d", i)
				o.PolicyVersion = 1
				if err := s.ObserveQuota(ctx, c.Principal{Workspace: "team", Person: "alice", Device: o.Device}, o.QuotaObservation); err != nil {
					t.Fatal(err)
				}
			}
			for i, p := range points {
				v := c.UsageCapture{SourceRef: fmt.Sprintf("valid-%d", i), Revision: "1", PolicyVersion: 1, Client: "codex", Project: p.Project, ObservedAt: time.Now(), Coverage: "reported", Points: []c.UsagePoint{{Timestamp: p.At.Format(time.RFC3339), Model: "gpt-6-sol", InputTokens: amount(int64(*p.Weight * 1000000)), OutputTokens: amount(0), CacheReadTokens: amount(0), CacheWriteTokens: amount(0)}}}
				if err := s.ObserveUsage(ctx, c.Principal{Workspace: "team", Person: p.Person, Device: p.Device}, v); err != nil {
					t.Fatal(err)
				}
			}
			oldStart, oldEnd := from.Add(-48*time.Hour).Format(time.RFC3339), from.Add(-47*time.Hour).Format(time.RFC3339)
			// Decode the wire contract to exercise compatibility with pre-ended_at captures.
			capture := func(start, end string) c.UsageCapture {
				wire, _ := json.Marshal(map[string]any{"source_ref": "missing-history", "revision": "1", "policy_version": 1, "client": "codex", "project": "excluded-project", "observed_at": time.Now(), "started_at": start, "ended_at": end, "coverage": "unavailable"})
				var v c.UsageCapture
				if err := json.Unmarshal(wire, &v); err != nil {
					t.Fatal(err)
				}
				return v
			}
			v := capture(oldStart, oldEnd)
			switch tc.kind {
			case "undated":
				v.Coverage = "reported"
				v.Points = []c.UsagePoint{{Timestamp: "invalid", Model: "gpt-6-sol"}}
			case "overlap":
				v = capture(from.Format(time.RFC3339), to.Format(time.RFC3339))
			case "missing_end":
				v = capture(oldStart, "")
			case "invalid_end":
				v = capture(oldStart, "invalid")
			case "inverted":
				v = capture(oldEnd, oldStart)
			case "outside_point":
				v.Points = []c.UsagePoint{{Timestamp: from.Add(time.Minute).Format(time.RFC3339)}}
			case "outside_activity":
				v.Activity = []c.UsageActivity{{Timestamp: from.Add(time.Minute).Format(time.RFC3339)}}
			case "conflict_overlap":
				v = capture(from.Format(time.RFC3339), to.Format(time.RFC3339))
			case "conflict_unknown":
				v = capture("", "")
			}
			principal := alice()
			if err := s.ObserveUsage(ctx, principal, v); err != nil {
				t.Fatal(err)
			}
			if tc.kind == "conflict" || tc.kind == "conflict_overlap" || tc.kind == "conflict_unknown" {
				v = capture(oldStart, oldEnd)
				v.Client = "claude"
				v.ObservedAt = v.ObservedAt.Add(time.Second)
				principal.Device = "latest-copy"
				if err := s.ObserveUsage(ctx, principal, v); err != nil {
					t.Fatal(err)
				}
			}
			d, err := s.Analytics(ctx, owner(), UsageFilter{Hours: 1, Person: "alice", Project: "alpha"}, to.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			if len(d.QuotaEstimates) != 1 {
				t.Fatalf("missing interval: %+v", d.QuotaEstimates)
			}
			got := d.QuotaEstimates[0]
			if tc.available {
				if got.Status != "estimated" || got.EstimatedPercentagePoints == nil || *got.EstimatedPercentagePoints != 3 {
					t.Fatalf("irrelevant old source blocked intact 3/4 denominator share: %+v", got)
				}
			} else if got.Status != "unavailable" || got.EstimatedPercentagePoints != nil || len(got.Allocations) != 0 {
				t.Fatalf("uncertain source excluded from denominator: %+v", got)
			}
		})
	}
}
