package store

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"
)

func quotaEstimateFixture() ([]NativeQuota, []QuotaUsagePoint, time.Time, time.Time) {
	from := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	to := from.Add(10 * time.Minute)
	reset := to.Add(time.Hour).Unix()
	duration := int64(300)
	obs := []NativeQuota{}
	for _, device := range []string{"alice-device", "bob-device"} {
		for i, at := range []time.Time{from, to} {
			used := []float64{20, 24}[i]
			obs = append(obs, NativeQuota{Device: device, QuotaObservation: c.QuotaObservation{Provider: "codex", Profile: "default", Snapshot: quota.Snapshot{Email: "shared@example.test", Plan: "pro", ObservedAt: at, Windows: []quota.Window{{Bucket: "codex", Name: "primary", UsedPercent: &used, ResetsAt: &reset, WindowDurationMins: &duration}}}}})
		}
	}
	a, b := float64(3), float64(1)
	points := []QuotaUsagePoint{{At: from.Add(4 * time.Minute), Person: "alice", Device: "alice-device", Project: "alpha", Model: "gpt-5", Effort: "high", Client: "codex", Weight: &a}, {At: from.Add(6 * time.Minute), Person: "bob", Device: "bob-device", Project: "beta", Model: "gpt-5", Effort: "high", Client: "codex", Weight: &b}}
	return obs, points, from, to
}

// Removing denominator preservation or summing duplicate device observations must fail this test.
func TestQuotaEstimatesAllocateObservedIncreaseWithoutDuplicateDevices(t *testing.T) {
	obs, points, from, to := quotaEstimateFixture()
	rows := estimateQuotas(obs, points, from, to, UsageFilter{}, true)
	if len(rows) != 1 || rows[0].Status != "estimated" || rows[0].ObservedPercentagePoints == nil || *rows[0].ObservedPercentagePoints != 4 || rows[0].EstimatedPercentagePoints == nil || *rows[0].EstimatedPercentagePoints != 4 {
		t.Fatalf("want one four-point estimate: %+v", rows)
	}
	if len(rows[0].Allocations) != 2 || rows[0].Allocations[0].Person != "alice" || rows[0].Allocations[0].PercentagePoints != 3 || rows[0].Allocations[1].Person != "bob" || rows[0].Allocations[1].PercentagePoints != 1 {
		t.Fatalf("want alice=3 bob=1: %+v", rows[0].Allocations)
	}
	for _, f := range []UsageFilter{{Person: "alice"}, {Project: "alpha"}} {
		filtered := estimateQuotas(obs, points, from, to, f, true)
		if len(filtered) != 1 || filtered[0].EstimatedPercentagePoints == nil || *filtered[0].EstimatedPercentagePoints != 3 || len(filtered[0].Allocations) != 1 {
			t.Fatalf("filtered share must retain other person's denominator: %+v", filtered)
		}
	}
}

// Each mutation removes one piece of evidence required for a safe conditional allocation.
func TestQuotaEstimatesRejectUnreliableIntervals(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*[]NativeQuota, *[]QuotaUsagePoint, *time.Time, *time.Time)
		full   bool
	}{
		{"restricted_scope", func(_ *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {}, false},
		{"unweighted_point", func(_ *[]NativeQuota, p *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) { (*p)[1].Weight = nil }, true},
		{"unknown_model", func(_ *[]NativeQuota, p *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) { (*p)[1].Model = "unknown" }, true},
		{"nonfinite_weight", func(_ *[]NativeQuota, p *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			x := math.Inf(1)
			(*p)[1].Weight = &x
		}, true},
		{"unknown_device", func(_ *[]NativeQuota, p *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			(*p)[1].Device = "unobserved"
		}, true},
		{"account_switch", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			(*o)[3].Email = "other@example.test"
		}, true},
		{"ambiguous_profiles", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			extra := (*o)[0]
			extra.Profile = "second"
			*o = append(*o, extra)
		}, true},
		{"same_time_conflict", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			x := 25.0
			(*o)[3].Windows[0].UsedPercent = &x
		}, true},
		{"reset_transition", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			for _, i := range []int{1, 3} {
				x := *(*o)[i].Windows[0].ResetsAt + 3600
				(*o)[i].Windows[0].ResetsAt = &x
			}
		}, true},
		{"percentage_drop", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			for _, i := range []int{1, 3} {
				x := 19.0
				(*o)[i].Windows[0].UsedPercent = &x
			}
		}, true},
		{"plan_change", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			(*o)[1].Plan = "plus"
			(*o)[3].Plan = "plus"
		}, true},
		{"missing_reset", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) {
			(*o)[1].Windows[0].ResetsAt = nil
			(*o)[3].Windows[0].ResetsAt = nil
		}, true},
		{"missing_reading", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, _ *time.Time) { *o = (*o)[:1] }, true},
		{"large_gap", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, _ *time.Time, to *time.Time) {
			*to = to.Add(time.Hour)
			(*o)[1].ObservedAt = *to
			(*o)[3].ObservedAt = *to
		}, true},
		{"partial_requested_interval", func(_ *[]NativeQuota, _ *[]QuotaUsagePoint, from *time.Time, _ *time.Time) {
			*from = from.Add(time.Minute)
		}, true},
		{"observation_error", func(o *[]NativeQuota, _ *[]QuotaUsagePoint, from *time.Time, _ *time.Time) {
			*o = append(*o, NativeQuota{Device: "alice-device", QuotaObservation: c.QuotaObservation{Provider: "codex", Profile: "default", Error: "unavailable", Snapshot: quota.Snapshot{ObservedAt: from.Add(5 * time.Minute)}}})
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs, points, from, to := quotaEstimateFixture()
			tt.mutate(&obs, &points, &from, &to)
			rows := estimateQuotas(obs, points, from, to, UsageFilter{}, tt.full)
			if len(rows) == 0 {
				t.Fatal("missing unavailable evidence summary")
			}
			for _, row := range rows {
				if row.Status != "unavailable" || row.EstimatedPercentagePoints != nil || len(row.Allocations) != 0 || row.Reason == "" {
					t.Fatalf("unsafe estimate: %+v", row)
				}
			}
		})
	}
}

func TestQuotaEstimatesMissingHistoryDoesNotUseCurrentIdentity(t *testing.T) {
	obs, points, from, to := quotaEstimateFixture()
	for i := range obs {
		obs[i].ObservedAt = to.Add(time.Hour)
	}
	rows := estimateQuotas(obs, points, from, to, UsageFilter{}, true)
	if len(rows) == 0 {
		t.Fatal("missing coverage summary")
	}
	for _, row := range rows {
		if row.Status != "unavailable" || len(row.Allocations) > 0 {
			t.Fatalf("current identity attributed history: %+v", row)
		}
	}
}

func TestQuotaEstimatesUndatedCapturedUsageBlocksAllocation(t *testing.T) {
	obs, points, from, to := quotaEstimateFixture()
	points[1].At = time.Time{}
	rows := estimateQuotas(obs, points, from, to, UsageFilter{}, true)
	for _, row := range rows {
		if row.Status != "unavailable" || len(row.Allocations) != 0 {
			t.Fatalf("undated usage silently removed from denominator: %+v", row)
		}
	}
}

func TestQuotaEstimatesOtherClientsDoNotPolluteCodexDenominator(t *testing.T) {
	obs, points, from, to := quotaEstimateFixture()
	points = append(points, QuotaUsagePoint{At: from.Add(time.Minute), Client: "claude"})
	rows := estimateQuotas(obs, points, from, to, UsageFilter{}, true)
	if len(rows) != 1 || rows[0].Status != "estimated" || rows[0].EstimatedPercentagePoints == nil || *rows[0].EstimatedPercentagePoints != 4 {
		t.Fatalf("unrelated provider blocked Codex: %+v", rows)
	}
}

func TestQuotaEstimatesCannotAssignUsageAcrossDifferentBuckets(t *testing.T) {
	obs, points, from, to := quotaEstimateFixture()
	for i := range obs {
		other := obs[i].Windows[0]
		other.Bucket = "separate-model-limit"
		obs[i].Windows = append(obs[i].Windows, other)
	}
	rows := estimateQuotas(obs, points, from, to, UsageFilter{}, true)
	if len(rows) != 2 {
		t.Fatalf("want both bucket summaries: %+v", rows)
	}
	for _, row := range rows {
		if row.Status != "unavailable" || len(row.Allocations) != 0 {
			t.Fatalf("usage lacks a bucket binding: %+v", row)
		}
	}
}

func TestQuotaEstimateEndpointReadingsPreserveMissingBaseline(t *testing.T) {
	obs, points, from, to := quotaEstimateFixture()
	points[0].Weight = nil // Provider readings survive a withheld allocation.
	rows := estimateQuotas(obs, points, from, to, UsageFilter{}, true)
	body, _ := json.Marshal(rows[0])
	var wire map[string]any
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["from_used_percent"] != float64(20) || wire["to_used_percent"] != float64(24) || rows[0].EstimatedPercentagePoints != nil {
		t.Fatalf("missing raw endpoint readings: %s", body)
	}
	// A first reading of 3% cannot create a synthetic zero baseline or delta.
	used := 3.0
	obs[1].Windows[0].UsedPercent = &used
	rows = estimateQuotas(obs[1:2], points, from, to, UsageFilter{}, true)
	if len(rows) != 1 || rows[0].Reason != "missing_quota_bounds" || rows[0].ObservedPercentagePoints != nil || rows[0].EstimatedPercentagePoints != nil {
		t.Fatalf("invented initial three points: %+v", rows)
	}
}

func TestQuotaEstimatesBoundedUnknownSourcesAffectOnlyOverlappingIntervals(t *testing.T) {
	obs, points, from, to := quotaEstimateFixture()
	next := to.Add(10 * time.Minute)
	for _, i := range []int{1, 3} {
		o := obs[i]
		o.ObservedAt = next
		o.Windows = append([]quota.Window(nil), o.Windows...)
		used := 28.0
		o.Windows[0].UsedPercent = &used
		obs = append(obs, o)
	}
	nextPoint := points[0]
	nextPoint.At = to.Add(4 * time.Minute)
	points = append(points, nextPoint)
	// No exact timestamp is invented for this incomplete source. Only the first
	// observed interval overlaps its possible source range.
	points = append(points, QuotaUsagePoint{Client: "codex", RangeStart: from.Add(time.Minute), RangeEnd: from.Add(2 * time.Minute)})
	rows := estimateQuotas(obs, points, from, next, UsageFilter{}, true)
	if len(rows) != 2 || rows[0].Status != "unavailable" || rows[1].Status != "estimated" || rows[1].EstimatedPercentagePoints == nil || *rows[1].EstimatedPercentagePoints != 4 {
		t.Fatalf("source blocker escaped its interval: %+v", rows)
	}
}

func TestQuotaEstimateEndpointReadingsRejectInvalidAndConflictingValues(t *testing.T) {
	for _, mode := range []string{"invalid", "conflicting", "missing"} {
		t.Run(mode, func(t *testing.T) {
			obs, points, from, to := quotaEstimateFixture()
			switch mode {
			case "invalid":
				invalid := -1.0
				obs[1].Windows[0].UsedPercent = &invalid
				obs[3].Windows[0].UsedPercent = &invalid
			case "conflicting":
				other := 25.0
				obs[3].Windows[0].UsedPercent = &other
			case "missing":
				obs[1].Windows[0].UsedPercent = nil
				obs[3].Windows[0].UsedPercent = nil
			}
			rows := estimateQuotas(obs, points, from, to, UsageFilter{}, true)
			if len(rows) != 1 || rows[0].FromUsedPercent == nil || *rows[0].FromUsedPercent != 20 || rows[0].ToUsedPercent != nil || rows[0].ObservedPercentagePoints != nil {
				t.Fatalf("unreliable endpoints emitted: %+v", rows)
			}
		})
	}
}
