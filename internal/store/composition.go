package store

import (
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/usageweights"
	"sort"
	"time"
)

type CompositionRow struct {
	Client           string   `json:"client"`
	Model            string   `json:"model"`
	Effort           string   `json:"effort"`
	Project          string   `json:"project"`
	Branch           string   `json:"branch"`
	Weight           *float64 `json:"weight"`
	Prompts          *int64   `json:"prompts"`
	GeneratedLines   *int64   `json:"generated_lines"`
	Points           int      `json:"points"`
	UnweightedPoints int      `json:"unweighted_points"`
}
type UsageComposition struct {
	RateVersion                string           `json:"rate_version"`
	Rows                       []CompositionRow `json:"rows"`
	ActivitySources            int              `json:"activity_sources"`
	ActivityUnavailableSources int              `json:"activity_unavailable_sources"`
}
type compositionBuilder map[string]*CompositionRow

func (b compositionBuilder) row(v c.UsageCapture, model, effort string) *CompositionRow {
	key, _ := json.Marshal([]string{v.Client, model, effort, v.Project, v.Branch})
	if b[string(key)] == nil {
		b[string(key)] = &CompositionRow{Client: v.Client, Model: model, Effort: effort, Project: v.Project, Branch: v.Branch}
	}
	return b[string(key)]
}
func addActivity(dst **int64, value *int64) {
	if value != nil {
		if *dst == nil {
			n := int64(0)
			*dst = &n
		}
		**dst += *value
	}
}
func (b compositionBuilder) point(v c.UsageCapture, p c.UsagePoint) {
	r := b.row(v, p.Model, p.Effort)
	r.Points++
	w := usageweights.Weight(v.Client, p)
	if w == nil {
		r.UnweightedPoints++
		return
	}
	if r.Weight == nil {
		n := 0.
		r.Weight = &n
	}
	*r.Weight += *w
}
func (b compositionBuilder) activity(v c.UsageCapture, a c.UsageActivity) {
	r := b.row(v, a.Model, a.Effort)
	addActivity(&r.Prompts, a.Prompts)
	addActivity(&r.GeneratedLines, a.GeneratedLines)
}
func (b compositionBuilder) finish() []CompositionRow {
	keys := make([]string, 0, len(b))
	for k := range b {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]CompositionRow, 0, len(b))
	for _, k := range keys {
		out = append(out, *b[k])
	}
	return out
}
func usageWindow(f UsageFilter, now time.Time, zone *time.Location) (time.Time, time.Time) {
	if f.Hours > 0 {
		return now.Add(-time.Duration(f.Hours) * time.Hour), now
	}
	d := now.In(zone)
	start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, zone)
	if f.Period != "today" {
		start = start.AddDate(0, 0, 1-f.Days)
	}
	return start, now
}
func inUsageWindow(timestamp string, start, end time.Time) bool {
	at, e := time.Parse(time.RFC3339Nano, timestamp)
	return e == nil && !at.Before(start) && at.Before(end)
}

// Merge source rows after counting. A missing counter in any contributing
// source stays unknown, including sources with identical grouping dimensions.
func (b compositionBuilder) merge(source compositionBuilder, complete bool) {
	for key, row := range source {
		if !complete {
			row.Prompts = nil
			row.GeneratedLines = nil
		}
		existing, ok := b[key]
		if !ok {
			b[key] = row
			continue
		}
		if row.Weight != nil {
			if existing.Weight == nil {
				n := 0.
				existing.Weight = &n
			}
			*existing.Weight += *row.Weight
		}
		existing.Points += row.Points
		existing.UnweightedPoints += row.UnweightedPoints
		for _, pair := range []struct {
			dst **int64
			src *int64
		}{{&existing.Prompts, row.Prompts}, {&existing.GeneratedLines, row.GeneratedLines}} {
			if *pair.dst == nil || pair.src == nil {
				*pair.dst = nil
			} else {
				**pair.dst += *pair.src
			}
		}
	}
}
