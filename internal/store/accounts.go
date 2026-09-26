package store

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"math"
	"time"
)

type Account struct {
	Alias    string   `json:"alias"`
	Provider string   `json:"provider"`
	Plan     string   `json:"plan"`
	Assigned []string `json:"assigned"`
	Quotas   []Quota  `json:"quotas,omitempty"`
}
type Quota struct {
	Account     string    `json:"account"`
	Window      string    `json:"window"`
	UsedPercent float64   `json:"used_percent"`
	ObservedAt  time.Time `json:"observed_at"`
	ResetsAt    time.Time `json:"resets_at"`
	Source      string    `json:"source"`
	Evidence    string    `json:"evidence"`
}

func (s *Store) PutAccount(ctx context.Context, p c.Principal, a Account) error {
	if !p.Manager() {
		return c.ErrForbidden
	}
	if !safeID.MatchString(a.Alias) || a.Provider == "" || len(a.Provider) > 64 || len(a.Plan) > 128 || len(a.Assigned) > 100 {
		return c.ErrInvalid
	}
	for _, person := range a.Assigned {
		if !safeID.MatchString(person) {
			return c.ErrInvalid
		}
	}
	a.Quotas = nil
	b, _ := json.Marshal(a)
	_, e := s.DB.Exec(ctx, `INSERT INTO tm_accounts(workspace,alias,body) VALUES($1,$2,$3) ON CONFLICT(workspace,alias) DO UPDATE SET body=excluded.body,updated_at=now()`, p.Workspace, a.Alias, b)
	return e
}
func (s *Store) AddQuota(ctx context.Context, p c.Principal, q Quota) error {
	if !p.Manager() {
		return c.ErrForbidden
	}
	if !safeID.MatchString(q.Account) || q.Window == "" || len(q.Window) > 64 || math.IsNaN(q.UsedPercent) || math.IsInf(q.UsedPercent, 0) || q.UsedPercent < 0 || q.UsedPercent > 100 || q.Source != "manual" || q.ObservedAt.IsZero() || q.ObservedAt.After(time.Now().Add(time.Minute)) || !q.ResetsAt.After(q.ObservedAt) || len(q.Evidence) > 4096 {
		return c.ErrInvalid
	}
	b, _ := json.Marshal(q)
	_, e := s.DB.Exec(ctx, "INSERT INTO tm_quotas(workspace,account,body,observed_at) VALUES($1,$2,$3,$4)", p.Workspace, q.Account, b, q.ObservedAt)
	return e
}
func (s *Store) Accounts(ctx context.Context, p c.Principal) ([]Account, error) {
	if p.Device != "" {
		return nil, c.ErrForbidden
	}
	policy, e := s.Policy(ctx, p.Workspace)
	if e != nil {
		return nil, e
	}
	rows, e := s.DB.Query(ctx, "SELECT body FROM tm_accounts WHERE workspace=$1 ORDER BY alias", p.Workspace)
	if e != nil {
		return nil, e
	}
	var out []Account
	for rows.Next() {
		var b []byte
		var a Account
		if e = rows.Scan(&b); e != nil {
			rows.Close()
			return nil, e
		}
		if e = json.Unmarshal(b, &a); e != nil {
			rows.Close()
			return nil, e
		}
		allowed := p.Manager() || policy.Visibility == "team"
		for _, person := range a.Assigned {
			allowed = allowed || person == p.Person
		}
		if allowed {
			out = append(out, a)
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if out == nil {
		out = []Account{}
	}
	for i := range out {
		rs, e := s.DB.Query(ctx, `SELECT body FROM tm_quotas WHERE workspace=$1 AND account=$2 ORDER BY observed_at DESC,id DESC LIMIT 100`, p.Workspace, out[i].Alias)
		if e != nil {
			return nil, e
		}
		out[i].Quotas = []Quota{}
		for rs.Next() {
			var b []byte
			var q Quota
			if e = rs.Scan(&b); e != nil {
				rs.Close()
				return nil, e
			}
			if e = json.Unmarshal(b, &q); e != nil {
				rs.Close()
				return nil, e
			}
			out[i].Quotas = append(out[i].Quotas, q)
		}
		e = rs.Err()
		rs.Close()
		if e != nil {
			return nil, e
		}
	}
	return out, nil
}
