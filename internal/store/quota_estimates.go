package store

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"
)

// QuotaUsagePoint is captured usage, not a provider quota debit. Weight is an
// explicit model/counter weight; nil preserves incomplete denominator evidence.
// Effort is a recorded grouping field, not an invented quota multiplier.
type QuotaUsagePoint struct {
	At                                             time.Time
	Person, Device, Project, Model, Effort, Client string
	Weight                                         *float64
}

type QuotaAllocation struct {
	Person           string  `json:"person"`
	Project          string  `json:"project"`
	Model            string  `json:"model"`
	Effort           string  `json:"effort"`
	PercentagePoints float64 `json:"percentage_points"`
}

type QuotaEstimate struct {
	Provider                  string            `json:"provider"`
	Email                     string            `json:"email"`
	Bucket                    string            `json:"bucket"`
	Window                    string            `json:"window"`
	From                      string            `json:"from"`
	To                        string            `json:"to"`
	ResetsAt                  *int64            `json:"resets_at"`
	ObservedPercentagePoints  *float64          `json:"observed_percentage_points"`
	EstimatedPercentagePoints *float64          `json:"estimated_percentage_points"`
	Status                    string            `json:"status"`
	Reason                    string            `json:"reason"`
	Allocations               []QuotaAllocation `json:"allocations"`
}

const quotaIdentityGap = 15 * time.Minute
const quotaEstimateAssumption = "Conditional estimate proportional to captured model/counter weights; captured scope does not prove complete provider usage or actual employee/project quota debits."

type quotaEstimateKey struct{ provider, email, bucket, window string }
type quotaEstimateReading struct {
	at       time.Time
	plan     string
	window   quota.Window
	conflict bool
}
type quotaBoundPoint struct {
	point         QuotaUsagePoint
	email, reason string
}

// estimateQuotas allocates account-window increases only under a conditional
// captured-usage assumption. Observations bracket each point on its own device
// and profile; a present-day account is never copied onto historical usage.
// Callers must pass ALL captured points in the authorized scope before applying
// person/project filters. Restricted scopes cannot establish a denominator.
func estimateQuotas(observations []NativeQuota, points []QuotaUsagePoint, from, to time.Time, f UsageFilter, fullScope bool) []QuotaEstimate {
	result := []QuotaEstimate{}
	orderedObservations := append([]NativeQuota(nil), observations...)
	sort.Slice(orderedObservations, func(i, j int) bool {
		return orderedObservations[i].ObservedAt.Before(orderedObservations[j].ObservedAt)
	})
	timelines := map[quotaEstimateKey][]quotaEstimateReading{}
	devices := map[string][]NativeQuota{}
	for _, o := range observations {
		if o.Provider != "codex" || o.ObservedAt.IsZero() {
			continue
		}
		devices[o.Device] = append(devices[o.Device], o)
		if o.Error != "" || o.Email == "" {
			continue
		}
		for _, w := range o.Windows {
			key := quotaEstimateKey{o.Provider, strings.ToLower(o.Email), w.Bucket, w.Name}
			timelines[key] = append(timelines[key], quotaEstimateReading{at: o.ObservedAt, plan: o.Plan, window: w})
		}
	}
	for device := range devices {
		sort.Slice(devices[device], func(i, j int) bool { return devices[device][i].ObservedAt.Before(devices[device][j].ObservedAt) })
	}
	bound := []quotaBoundPoint{}
	undated := false
	for _, p := range points {
		if !fullScope || !strings.EqualFold(p.Client, "codex") {
			continue
		}
		if p.At.IsZero() {
			undated = true
			continue
		}
		email, reason := quotaPointIdentity(p, devices[p.Device])
		bound = append(bound, quotaBoundPoint{p, email, reason})
	}
	sort.SliceStable(bound, func(i, j int) bool { return bound[i].point.At.Before(bound[j].point.At) })
	keys := make([]quotaEstimateKey, 0, len(timelines))
	for k := range timelines {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.provider != b.provider {
			return a.provider < b.provider
		}
		if a.email != b.email {
			return a.email < b.email
		}
		if a.bucket != b.bucket {
			return a.bucket < b.bucket
		}
		return a.window < b.window
	})
	summary := func(key quotaEstimateKey, reason string) QuotaEstimate {
		return QuotaEstimate{Provider: key.provider, Email: key.email, Bucket: key.bucket, Window: key.window, From: from.UTC().Format(time.RFC3339Nano), To: to.UTC().Format(time.RFC3339Nano), Status: "unavailable", Reason: reason, Allocations: []QuotaAllocation{}}
	}
	for _, key := range keys {
		if !fullScope {
			result = append(result, summary(key, "incomplete_authorized_scope"))
			continue
		}
		readings := timelines[key]
		sort.Slice(readings, func(i, j int) bool { return readings[i].at.Before(readings[j].at) })
		canonical := make([]quotaEstimateReading, 0, len(readings))
		for _, r := range readings {
			n := len(canonical)
			if n > 0 && canonical[n-1].at.Equal(r.at) {
				if !sameQuotaReading(canonical[n-1], r) {
					canonical[n-1].conflict = true
				}
			} else {
				canonical = append(canonical, r)
			}
		}
		emitted := false
		for i := 1; i < len(canonical); i++ {
			left, right := canonical[i-1], canonical[i]
			if !right.at.After(from) || !left.at.Before(to) {
				continue
			}
			row := summary(key, "")
			row.From = left.at.UTC().Format(time.RFC3339Nano)
			row.To = right.at.UTC().Format(time.RFC3339Nano)
			row.ResetsAt = right.window.ResetsAt
			emitted = true
			reason := quotaIntervalProblem(left, right, from, to)
			if reason == "" {
				delta := *right.window.UsedPercent - *left.window.UsedPercent
				row.ObservedPercentagePoints = &delta
			}
			if undated {
				reason = "undated_captured_usage"
			}
			if reason == "" {
				// A failed read has no account identity. Conservatively withhold intervals
				// overlapping it instead of assuming the failed profile used another account.
				start := sort.Search(len(orderedObservations), func(i int) bool { return !orderedObservations[i].ObservedAt.Before(left.at) })
				for j := start; j < len(orderedObservations) && !orderedObservations[j].ObservedAt.After(right.at); j++ {
					o := orderedObservations[j]
					if o.Provider != "codex" {
						continue
					}
					if o.Error != "" {
						reason = "quota_observation_error"
						break
					}
					if strings.EqualFold(o.Email, key.email) {
						found := false
						for _, w := range o.Windows {
							// Captured usage has no provider bucket binding. It cannot
							// safely supply a separate denominator for parallel buckets.
							if w.Bucket != key.bucket {
								reason = "ambiguous_quota_bucket"
							}
							if w.Bucket == key.bucket && w.Name == key.window {
								found = true
							}
						}
						if reason != "" {
							break
						}
						if !found {
							reason = "missing_window_observation"
							break
						}
					}
				}
			}
			if reason == "" {
				reason = allocateQuotaInterval(&row, key, left.at, right.at, bound, f)
			}
			if reason != "" {
				row.Reason = reason
			} else {
				row.Status = "estimated"
				row.Reason = quotaEstimateAssumption
			}
			result = append(result, row)
		}
		if !emitted {
			reason := "missing_quota_bounds"
			if !fullScope {
				reason = "incomplete_authorized_scope"
			}
			result = append(result, summary(key, reason))
		}
	}
	if len(result) == 0 {
		reason := "missing_quota_bounds"
		for _, o := range observations {
			if o.Provider == "codex" && o.Error != "" {
				reason = "quota_observation_error"
				break
			}
		}
		if !fullScope {
			reason = "incomplete_authorized_scope"
		}
		result = append(result, summary(quotaEstimateKey{provider: "codex"}, reason))
	}
	return result
}

func quotaIntervalProblem(a, b quotaEstimateReading, from, to time.Time) string {
	if a.conflict || b.conflict {
		return "conflicting_simultaneous_observations"
	}
	if a.at.Before(from) || b.at.After(to) || !to.After(from) {
		return "partial_interval_bounds"
	}
	if b.at.Sub(a.at) > quotaIdentityGap {
		return "quota_observation_gap"
	}
	if a.plan == "" || b.plan == "" {
		return "unknown_plan"
	}
	if a.plan != b.plan {
		return "plan_changed"
	}
	x, y := a.window, b.window
	if x.UsedPercent == nil || y.UsedPercent == nil || x.ResetsAt == nil || y.ResetsAt == nil || x.WindowDurationMins == nil || y.WindowDurationMins == nil {
		return "missing_window_bounds"
	}
	if !validQuotaPercent(*x.UsedPercent) || !validQuotaPercent(*y.UsedPercent) || *x.WindowDurationMins <= 0 || *y.WindowDurationMins <= 0 {
		return "invalid_window_observation"
	}
	if *x.ResetsAt != *y.ResetsAt || *x.WindowDurationMins != *y.WindowDurationMins || !b.at.Before(time.Unix(*x.ResetsAt, 0)) {
		return "reset_or_window_transition"
	}
	if *y.UsedPercent < *x.UsedPercent {
		return "quota_percentage_dropped"
	}
	return ""
}

func validQuotaPercent(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 100
}
func sameQuotaReading(a, b quotaEstimateReading) bool {
	sameFloat := func(x, y *float64) bool { return x == nil && y == nil || x != nil && y != nil && *x == *y }
	sameInt := func(x, y *int64) bool { return x == nil && y == nil || x != nil && y != nil && *x == *y }
	return a.plan == b.plan && sameFloat(a.window.UsedPercent, b.window.UsedPercent) && sameInt(a.window.ResetsAt, b.window.ResetsAt) && sameInt(a.window.WindowDurationMins, b.window.WindowDurationMins)
}

func quotaPointIdentity(p QuotaUsagePoint, obs []NativeQuota) (string, string) {
	if p.Device == "" || p.At.IsZero() {
		return "", "missing_point_identity"
	}
	start := sort.Search(len(obs), func(i int) bool { return !obs[i].ObservedAt.Before(p.At.Add(-quotaIdentityGap)) })
	var before, after *NativeQuota
	profile, email := "", ""
	for i := start; i < len(obs) && !obs[i].ObservedAt.After(p.At.Add(quotaIdentityGap)); i++ {
		o := &obs[i]
		if o.Error != "" {
			return "", "quota_observation_error"
		}
		if o.Profile == "" || o.Email == "" {
			return "", "missing_profile_identity"
		}
		if profile != "" && (profile != o.Profile || !strings.EqualFold(email, o.Email)) {
			return "", "ambiguous_profile_account"
		}
		profile, email = o.Profile, strings.ToLower(o.Email)
		if o.ObservedAt.Before(p.At) {
			before = o
		} else if after == nil {
			after = o
		}
	}
	if before == nil || after == nil {
		return "", "missing_bracketing_identity"
	}
	return email, ""
}

func allocateQuotaInterval(row *QuotaEstimate, key quotaEstimateKey, from, to time.Time, points []quotaBoundPoint, f UsageFilter) string {
	start := sort.Search(len(points), func(i int) bool { return points[i].point.At.After(from) })
	selected := []QuotaUsagePoint{}
	total := 0.0
	for i := start; i < len(points) && !points[i].point.At.After(to); i++ {
		b := points[i]
		if b.reason != "" {
			return b.reason
		}
		if b.email != key.email {
			continue
		}
		p := b.point
		if p.Person == "" {
			return "unknown_person"
		}
		if p.Weight == nil || math.IsNaN(*p.Weight) || math.IsInf(*p.Weight, 0) || *p.Weight <= 0 || strings.TrimSpace(p.Model) == "" || strings.EqualFold(p.Model, "unknown") {
			return "unweighted_or_unknown_model_usage"
		}
		total += *p.Weight
		selected = append(selected, p)
	}
	if len(selected) == 0 || total <= 0 || math.IsInf(total, 0) {
		return "missing_weighted_usage"
	}
	type allocationKey struct{ person, project, model, effort string }
	shares := map[allocationKey]float64{}
	estimated := 0.0
	for _, p := range selected {
		if f.Person != "" && f.Person != p.Person || f.Project != "" && f.Project != p.Project || f.Client != "" && !strings.EqualFold(f.Client, p.Client) {
			continue
		}
		share := *row.ObservedPercentagePoints * (*p.Weight / total)
		shares[allocationKey{p.Person, p.Project, p.Model, p.Effort}] += share
		estimated += share
	}
	for k, v := range shares {
		row.Allocations = append(row.Allocations, QuotaAllocation{Person: k.person, Project: k.project, Model: k.model, Effort: k.effort, PercentagePoints: v})
	}
	sort.Slice(row.Allocations, func(i, j int) bool {
		a, b := row.Allocations[i], row.Allocations[j]
		if a.Person != b.Person {
			return a.Person < b.Person
		}
		if a.Project != b.Project {
			return a.Project < b.Project
		}
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		return a.Effort < b.Effort
	})
	row.EstimatedPercentagePoints = &estimated
	return ""
}
