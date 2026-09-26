package store

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/jackc/pgx/v5"
	"time"
	_ "time/tzdata"
)

type captureMetrics struct {
	Day      string         `json:"day"`
	Project  string         `json:"project"`
	Client   string         `json:"client"`
	Messages int            `json:"messages"`
	Prompts  int            `json:"prompts"`
	Models   map[string]int `json:"models"`
}

func summarize(v c.Snapshot) captureMetrics {
	m := captureMetrics{Project: v.Project, Client: v.Client, Messages: len(v.Messages), Models: map[string]int{}}
	zone, _ := time.LoadLocation("Asia/Kolkata")
	if d, e := time.Parse(time.RFC3339Nano, v.StartedAt); e == nil {
		m.Day = d.In(zone).Format("2006-01-02")
	}
	for _, v := range v.Messages {
		if v.Role == "user" {
			m.Prompts++
		}
		if v.Role == "assistant" {
			name := v.Model
			if name == "" {
				name = "Unknown"
			}
			m.Models[name]++
		}
	}
	return m
}
func putMetrics(ctx context.Context, tx pgx.Tx, id string, v c.Snapshot) error {
	b, e := json.Marshal(summarize(v))
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "INSERT INTO tm_snapshot_metrics(snapshot_id,body) VALUES($1,$2) ON CONFLICT DO NOTHING", id, b)
	return e
}

// Backfill compact summaries in bounded batches. Transcripts never reach dashboard responses.
func (s *Store) BackfillMetrics(ctx context.Context, limit int) error {
	for i := 0; i < limit; i++ {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return e
		}
		var id string
		var b []byte
		e = tx.QueryRow(ctx, `SELECT s.id,jsonb_build_object('project',s.body->'project','client',s.body->'client','started_at',s.body->'started_at','messages',coalesce((SELECT jsonb_agg(jsonb_build_object('role',m->'role','model',m->'model')) FROM jsonb_array_elements(s.body->'messages') m),'[]'::jsonb)) FROM tm_snapshots s LEFT JOIN tm_snapshot_metrics m ON m.snapshot_id=s.id WHERE m.snapshot_id IS NULL ORDER BY s.received_at DESC LIMIT 1 FOR UPDATE OF s SKIP LOCKED`).Scan(&id, &b)
		if e == pgx.ErrNoRows {
			tx.Rollback(ctx)
			return nil
		}
		var v c.Snapshot
		if e == nil {
			e = json.Unmarshal(b, &v)
		}
		if e == nil {
			e = putMetrics(ctx, tx, id, v)
		}
		if e != nil {
			tx.Rollback(ctx)
			return e
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
	}
	return nil
}

type Dashboard struct {
	Sessions        int            `json:"sessions"`
	Prompts         int            `json:"prompts"`
	IndexedSources  int            `json:"indexed_sources"`
	PendingCaptures int            `json:"pending_captures"`
	UnknownDates    int            `json:"unknown_dates"`
	Days            map[string]int `json:"days"`
	Projects        map[string]int `json:"projects"`
	Clients         map[string]int `json:"clients"`
	Models          map[string]int `json:"models"`
	Start           string         `json:"start"`
	End             string         `json:"end"`
	Timezone        string         `json:"timezone"`
}

func (s *Store) Dashboard(ctx context.Context, p c.Principal, days int, now time.Time) (Dashboard, error) {
	d := Dashboard{Days: map[string]int{}, Projects: map[string]int{}, Clients: map[string]int{}, Models: map[string]int{}, Timezone: "Asia/Kolkata"}
	if p.Device != "" || p.Workspace == "" {
		return d, c.ErrForbidden
	}
	if days != 7 && days != 14 && days != 30 && days != 90 {
		return d, c.ErrInvalid
	}
	zone, _ := time.LoadLocation(d.Timezone)
	now = now.In(zone)
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	d.End = end.Format("2006-01-02")
	d.Start = end.AddDate(0, 0, 1-days).Format("2006-01-02")
	for i := 0; i < days; i++ {
		d.Days[end.AddDate(0, 0, -i).Format("2006-01-02")] = 0
	}
	rows, e := s.DB.Query(ctx, `SELECT DISTINCT ON(s.source_ref) m.body FROM tm_snapshots s JOIN tm_snapshot_metrics m ON m.snapshot_id=s.id JOIN tm_policies p ON p.workspace=s.workspace WHERE s.workspace=$1 AND ($2 OR s.owner_id=$3 OR p.body->>'visibility'='team') ORDER BY s.source_ref,(m.body->>'messages')::int DESC,s.received_at DESC,s.id DESC`, p.Workspace, p.Manager(), p.Person)
	if e != nil {
		return d, e
	}
	for rows.Next() {
		var b []byte
		var m captureMetrics
		if e = rows.Scan(&b); e != nil {
			break
		}
		if e = json.Unmarshal(b, &m); e != nil {
			break
		}
		d.IndexedSources++
		if m.Day == "" {
			d.UnknownDates++
			continue
		}
		if m.Day < d.Start || m.Day > d.End {
			continue
		}
		d.Sessions++
		d.Prompts += m.Prompts
		d.Days[m.Day]++
		if m.Project == "" {
			m.Project = "Unknown project"
		}
		if m.Client == "" {
			m.Client = "Unknown client"
		}
		d.Projects[m.Project]++
		d.Clients[m.Client]++
		for k, v := range m.Models {
			d.Models[k] += v
		}
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return d, e
	}
	e = s.DB.QueryRow(ctx, `SELECT count(*) FROM tm_snapshots s LEFT JOIN tm_snapshot_metrics m ON m.snapshot_id=s.id JOIN tm_policies p ON p.workspace=s.workspace WHERE m.snapshot_id IS NULL AND s.workspace=$1 AND ($2 OR s.owner_id=$3 OR p.body->>'visibility'='team')`, p.Workspace, p.Manager(), p.Person).Scan(&d.PendingCaptures)
	if e == nil {
		e = s.Audit(ctx, p, "dashboard.read", "aggregate")
	}
	return d, e
}
