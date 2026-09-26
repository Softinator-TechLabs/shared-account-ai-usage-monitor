package store

import (
	"context"
	"errors"
	"fmt"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"strings"
	"testing"
	"time"
)

func TestCompositionTodayUsesDatedActivityAndKeepsUnknowns(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 26, 16, 30, 0, 0, time.UTC)
	start := time.Date(2026, 9, 25, 18, 30, 0, 0, time.UTC)
	p := c.UsagePoint{Timestamp: start.Format(time.RFC3339), Model: "gpt-6-sol", Effort: "high", InputTokens: amount(1000000), OutputTokens: amount(0), CacheReadTokens: amount(0), CacheWriteTokens: amount(0)}
	v := c.UsageCapture{SourceRef: "composition", Revision: "one", PolicyVersion: 1, Client: "codex", Project: "repo", Branch: "main", ObservedAt: time.Now(), Coverage: "reported", ActivityCoverage: "reported", Points: []c.UsagePoint{p}, Activity: []c.UsageActivity{{Timestamp: start.Format(time.RFC3339), Model: p.Model, Effort: p.Effort, Prompts: amount(2), GeneratedLines: amount(7)}, {Timestamp: start.Add(-time.Second).Format(time.RFC3339), Model: p.Model, Prompts: amount(99)}}}
	q := p
	q.Timestamp = now.Format(time.RFC3339)
	v.Points = append(v.Points, q)
	if e := s.ObserveUsage(ctx, alice(), v); e != nil {
		t.Fatal(e)
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Period: "today"}, now)
	if e != nil || d.Start != start.Format(time.RFC3339) || d.End != now.Format(time.RFC3339) || len(d.Composition.Rows) != 1 {
		t.Fatal(d, e)
	}
	r := d.Composition.Rows[0]
	if r.Weight == nil || *r.Weight != 50 || r.Prompts == nil || *r.Prompts != 2 || r.GeneratedLines == nil || *r.GeneratedLines != 7 || r.Effort != "high" {
		t.Fatal(r)
	}
	if _, e = s.Analytics(ctx, owner(), UsageFilter{Period: "today", Hours: 24}, now); !errors.Is(e, c.ErrInvalid) {
		t.Fatal(e)
	}
	v.Activity[0].Prompts = amount(-1)
	if e = s.ObserveUsage(ctx, alice(), v); !errors.Is(e, c.ErrInvalid) {
		t.Fatal("invalid activity", e)
	}
}

func TestCompositionQuotaDenominatorSurvivesPersonFilter(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	obs, points, from, to := quotaEstimateFixture()
	for i, o := range obs {
		o.EventID = fmt.Sprintf("event-%d", i)
		o.PolicyVersion = 1
		person := "alice"
		if strings.HasPrefix(o.Device, "bob") {
			person = "bob"
		}
		if e := s.ObserveQuota(ctx, c.Principal{Workspace: "team", Person: person, Device: o.Device}, o.QuotaObservation); e != nil {
			t.Fatal(e)
		}
	}
	for i, p := range points {
		v := c.UsageCapture{SourceRef: fmt.Sprintf("source-%d", i), Revision: "1", PolicyVersion: 1, Client: "codex", Project: p.Project, ObservedAt: time.Now(), Coverage: "reported", Points: []c.UsagePoint{{Timestamp: p.At.Format(time.RFC3339), Model: "gpt-6-sol", Effort: "high", InputTokens: amount(int64(*p.Weight * 1000000)), OutputTokens: amount(0), CacheReadTokens: amount(0), CacheWriteTokens: amount(0)}}}
		if e := s.ObserveUsage(ctx, c.Principal{Workspace: "team", Person: p.Person, Device: p.Device}, v); e != nil {
			t.Fatal(e)
		}
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Hours: 1, Person: "alice"}, to.Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	if len(d.QuotaEstimates) != 1 || d.QuotaEstimates[0].EstimatedPercentagePoints == nil || *d.QuotaEstimates[0].EstimatedPercentagePoints != 3 {
		t.Fatal(d.QuotaEstimates)
	}
	if d.QuotaEstimates[0].From != from.Format(time.RFC3339) {
		t.Fatal(d.QuotaEstimates)
	}
}
func TestConflictingSourceWithEmptyLatestCopyBlocksQuotaDenominator(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	obs, points, _, to := quotaEstimateFixture()
	for i, o := range obs {
		o.EventID = fmt.Sprintf("event-%d", i)
		o.PolicyVersion = 1
		if e := s.ObserveQuota(ctx, c.Principal{Workspace: "team", Person: "alice", Device: o.Device}, o.QuotaObservation); e != nil {
			t.Fatal(e)
		}
	}
	for i, p := range points {
		v := c.UsageCapture{SourceRef: fmt.Sprintf("conflict-%d", i), Revision: "1", PolicyVersion: 1, Client: "codex", Project: p.Project, ObservedAt: time.Now(), Coverage: "reported", Points: []c.UsagePoint{{Timestamp: p.At.Format(time.RFC3339), Model: "gpt-6-sol", InputTokens: amount(1000000), OutputTokens: amount(0), CacheReadTokens: amount(0), CacheWriteTokens: amount(0)}}}
		principal := c.Principal{Workspace: "team", Person: p.Person, Device: p.Device}
		if e := s.ObserveUsage(ctx, principal, v); e != nil {
			t.Fatal(e)
		}
		if i == 1 {
			v.ObservedAt = v.ObservedAt.Add(time.Second)
			v.Points = nil
			v.Client = "claude"
			principal.Device = "stale-copy"
			if e := s.ObserveUsage(ctx, principal, v); e != nil {
				t.Fatal(e)
			}
		}
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Hours: 1}, to.Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	for _, r := range d.QuotaEstimates {
		if r.EstimatedPercentagePoints != nil {
			t.Fatal("allocated over hidden conflicting source", r)
		}
	}
}
func TestCompositionUnknownActivitySurvivesIdenticalSourceGroups(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()
	v := c.UsageCapture{SourceRef: "known", Revision: "1", PolicyVersion: 1, Client: "codex", Project: "alpha", ObservedAt: now, Coverage: "reported", ActivityCoverage: "reported", Points: []c.UsagePoint{{Timestamp: now.Add(-time.Minute).Format(time.RFC3339), Model: "gpt-6-sol", InputTokens: amount(10), OutputTokens: amount(0), CacheReadTokens: amount(0), CacheWriteTokens: amount(0)}}, Activity: []c.UsageActivity{{Timestamp: now.Add(-time.Minute).Format(time.RFC3339), Model: "gpt-6-sol", Prompts: amount(3), GeneratedLines: amount(20)}}}
	if e := s.ObserveUsage(ctx, alice(), v); e != nil {
		t.Fatal(e)
	}
	v.SourceRef = "unknown"
	v.Activity = nil
	v.ActivityCoverage = "unavailable"
	if e := s.ObserveUsage(ctx, alice(), v); e != nil {
		t.Fatal(e)
	}
	d, e := s.Analytics(ctx, owner(), UsageFilter{Hours: 1}, now)
	if e != nil || len(d.Composition.Rows) != 1 {
		t.Fatal(d, e)
	}
	r := d.Composition.Rows[0]
	if r.Prompts != nil || r.GeneratedLines != nil || r.Points != 2 {
		t.Fatal("unknown activity erased", r)
	}
}
