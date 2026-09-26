package store

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/policy"
	"sort"
	"time"
)

// ObserveUsage replaces a compact source projection, never adds revision totals.
func (s *Store) ObserveUsage(ctx context.Context, p c.Principal, v c.UsageCapture) error {
	if p.Workspace == "" || p.Person == "" || p.Device == "" {
		return c.ErrForbidden
	}
	if v.SourceRef == "" || len(v.SourceRef) > 512 || v.Revision == "" || len(v.Revision) > 256 || len(v.Project) > 1024 || len(v.Branch) > 1024 || v.Client == "" || len(v.Client) > 64 || v.Messages < 0 || v.Prompts < 0 || v.ObservedAt.IsZero() || v.ObservedAt.After(time.Now().Add(time.Minute)) || len(v.Points) > 100000 || (v.Coverage != "reported" && v.Coverage != "unavailable") {
		return c.ErrInvalid
	}
	for _, pt := range v.Points {
		if len(pt.Model) > 256 || len(pt.Timestamp) > 64 {
			return c.ErrInvalid
		}
		for _, n := range []*int64{pt.InputTokens, pt.OutputTokens, pt.CacheReadTokens, pt.CacheWriteTokens} {
			if n != nil && (*n < 0 || *n > 1000000000000) {
				return c.ErrInvalid
			}
		}
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var pb []byte
	var pol c.Policy
	if e = tx.QueryRow(ctx, "SELECT body FROM tm_policies WHERE workspace=$1 FOR SHARE", p.Workspace).Scan(&pb); e != nil {
		return e
	}
	if e = json.Unmarshal(pb, &pol); e != nil {
		return e
	}
	if pol.Version != v.PolicyVersion {
		return c.ErrForbidden
	}
	var b []byte
	v, b, e = policy.ApplyUsage(v, pol)
	if e != nil || len(b) > 16<<20 {
		return c.ErrInvalid
	}
	var deleted bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tm_deleted WHERE workspace=$1 AND source_ref=$2)", p.Workspace, v.SourceRef).Scan(&deleted); e != nil {
		return e
	}
	if deleted {
		return c.ErrDeleted
	}
	_, e = tx.Exec(ctx, `INSERT INTO tm_usage(workspace,person,device,source_ref,observed_at,body) VALUES($1,$2,$3,$4,$5,$6)
 ON CONFLICT(workspace,person,device,source_ref) DO UPDATE SET observed_at=excluded.observed_at,received_at=CASE WHEN (tm_usage.body - 'observed_at' - 'revision' - 'policy_version') IS DISTINCT FROM (excluded.body - 'observed_at' - 'revision' - 'policy_version') THEN now() ELSE tm_usage.received_at END,body=excluded.body WHERE tm_usage.observed_at<excluded.observed_at`, p.Workspace, p.Person, p.Device, v.SourceRef, v.ObservedAt, b)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

type UsageFilter struct {
	Days                    int
	Person, Project, Client string
}
type UsageCounters struct {
	InputTokens      *int64 `json:"input_tokens"`
	OutputTokens     *int64 `json:"output_tokens"`
	CacheReadTokens  *int64 `json:"cache_read_tokens"`
	CacheWriteTokens *int64 `json:"cache_write_tokens"`
	Points           int    `json:"points"`
}

func (v *UsageCounters) add(p c.UsagePoint) {
	for _, pair := range []struct {
		dst **int64
		src *int64
	}{{&v.InputTokens, p.InputTokens}, {&v.OutputTokens, p.OutputTokens}, {&v.CacheReadTokens, p.CacheReadTokens}, {&v.CacheWriteTokens, p.CacheWriteTokens}} {
		if pair.src != nil {
			if *pair.dst == nil {
				n := int64(0)
				*pair.dst = &n
			}
			**pair.dst += *pair.src
		}
	}
	v.Points++
}

type UsageGroup struct {
	UsageCounters
	Day       string   `json:"day,omitempty"`
	Person    string   `json:"person,omitempty"`
	Project   string   `json:"project,omitempty"`
	Client    string   `json:"client,omitempty"`
	Model     string   `json:"model,omitempty"`
	Sources   int      `json:"sources"`
	People    []string `json:"people,omitempty"`
	sourceSet map[string]bool
	personSet map[string]bool
}

func (g *UsageGroup) add(p c.UsagePoint, source, person string) {
	g.UsageCounters.add(p)
	if g.sourceSet == nil {
		g.sourceSet = map[string]bool{}
		g.personSet = map[string]bool{}
	}
	g.sourceSet[source] = true
	g.Sources = len(g.sourceSet)
	if person != "" {
		g.personSet[person] = true
	}
}

type UsageCoverage struct {
	Sources              int        `json:"sources"`
	UnavailableSources   int        `json:"unavailable_sources"`
	UndatedPoints        int        `json:"undated_points"`
	ContentConflicts     int        `json:"content_conflicts"`
	AttributionConflicts int        `json:"attribution_conflicts"`
	LastObservedAt       *time.Time `json:"last_observed_at"`
}
type UsageAnalytics struct {
	Start       string        `json:"start"`
	End         string        `json:"end"`
	Timezone    string        `json:"timezone"`
	Attribution string        `json:"attribution"`
	Coverage    UsageCoverage `json:"coverage"`
	Totals      UsageCounters `json:"totals"`
	Daily       []UsageGroup  `json:"daily"`
	People      []UsageGroup  `json:"people"`
	Projects    []UsageGroup  `json:"projects"`
	Clients     []UsageGroup  `json:"clients"`
	Models      []UsageGroup  `json:"models"`
}

func groups(m map[string]*UsageGroup) []UsageGroup {
	out := []UsageGroup{}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		v.People = []string{}
		for person := range v.personSet {
			v.People = append(v.People, person)
		}
		sort.Strings(v.People)
		out = append(out, *v)
	}
	return out
}
func group(m map[string]*UsageGroup, k string) *UsageGroup {
	if m[k] == nil {
		m[k] = &UsageGroup{}
	}
	return m[k]
}
func (s *Store) Analytics(ctx context.Context, p c.Principal, f UsageFilter, now time.Time) (UsageAnalytics, error) {
	d := UsageAnalytics{Timezone: "Asia/Kolkata", Attribution: "device_observed_not_verified_author", Daily: []UsageGroup{}, People: []UsageGroup{}, Projects: []UsageGroup{}, Clients: []UsageGroup{}, Models: []UsageGroup{}}
	if p.Device != "" || p.Workspace == "" || p.Person == "" {
		return d, c.ErrForbidden
	}
	if f.Days != 7 && f.Days != 14 && f.Days != 30 && f.Days != 90 || len(f.Person) > 128 || len(f.Project) > 1024 || len(f.Client) > 64 {
		return d, c.ErrInvalid
	}
	policy, e := s.Policy(ctx, p.Workspace)
	if e != nil {
		return d, e
	}
	all := p.Manager() || policy.Visibility == "team"
	if !all && f.Person != "" && f.Person != p.Person {
		return d, c.ErrForbidden
	}
	zone, _ := time.LoadLocation(d.Timezone)
	today := now.In(zone)
	d.End = today.Format("2006-01-02")
	d.Start = today.AddDate(0, 0, 1-f.Days).Format("2006-01-02")
	daily, people, projects, clients, models := map[string]*UsageGroup{}, map[string]*UsageGroup{}, map[string]*UsageGroup{}, map[string]*UsageGroup{}, map[string]*UsageGroup{}
	for i := 0; i < f.Days; i++ {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		group(daily, day).Day = day
	}
	// Conflicts are computed before person filtering: copies cannot silently become two employees' work.
	rows, e := s.DB.Query(ctx, `SELECT DISTINCT ON(u.source_ref) u.body,u.person,(SELECT count(DISTINCT person) FROM tm_usage x WHERE x.workspace=u.workspace AND x.source_ref=u.source_ref),(SELECT count(DISTINCT (body - 'observed_at' - 'revision' - 'policy_version')) FROM tm_usage x WHERE x.workspace=u.workspace AND x.source_ref=u.source_ref) FROM tm_usage u WHERE u.workspace=$1 AND ($2 OR u.person=$3) ORDER BY u.source_ref,u.observed_at DESC,u.person,u.device`, p.Workspace, all, p.Person)
	if e != nil {
		return d, e
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		var person string
		var owners, contents int
		var v c.UsageCapture
		if e = rows.Scan(&b, &person, &owners, &contents); e != nil {
			return d, e
		}
		if e = json.Unmarshal(b, &v); e != nil {
			return d, e
		}
		if v.Project == "" {
			v.Project = "Unknown project"
		}
		if f.Project != "" && v.Project != f.Project || f.Client != "" && v.Client != f.Client {
			continue
		}
		conflict := owners > 1
		if f.Person != "" && (person != f.Person || conflict) {
			continue
		}
		if conflict {
			d.Coverage.AttributionConflicts++
			person = ""
		}
		d.Coverage.Sources++
		if d.Coverage.LastObservedAt == nil || v.ObservedAt.After(*d.Coverage.LastObservedAt) {
			t := v.ObservedAt
			d.Coverage.LastObservedAt = &t
		}
		if contents > 1 {
			d.Coverage.ContentConflicts++
			continue
		}
		if v.Coverage != "reported" {
			d.Coverage.UnavailableSources++
		}
		for _, pt := range v.Points {
			at, err := time.Parse(time.RFC3339Nano, pt.Timestamp)
			if err != nil {
				d.Coverage.UndatedPoints++
				continue
			}
			day := at.In(zone).Format("2006-01-02")
			if day < d.Start || day > d.End {
				continue
			}
			if !all && conflict {
				continue
			}
			d.Totals.add(pt)
			g := group(daily, day)
			g.Day = day
			g.add(pt, v.SourceRef, person)
			if person != "" {
				g = group(people, person)
				g.Person = person
				g.add(pt, v.SourceRef, person)
			}
			project := v.Project
			if project == "" {
				project = "Unknown project"
			}
			g = group(projects, project)
			g.Project = project
			g.add(pt, v.SourceRef, person)
			g = group(clients, v.Client)
			g.Client = v.Client
			g.add(pt, v.SourceRef, person)
			model := pt.Model
			if model == "" {
				model = "Unknown model"
			}
			g = group(models, model)
			g.Model = model
			g.add(pt, v.SourceRef, person)
		}
	}
	if e = rows.Err(); e != nil {
		return d, e
	}
	d.Daily = groups(daily)
	d.People = groups(people)
	d.Projects = groups(projects)
	d.Clients = groups(clients)
	d.Models = groups(models)
	e = s.Audit(ctx, p, "analytics.read", "aggregate")
	return d, e
}
