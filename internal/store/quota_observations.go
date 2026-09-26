package store

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"
	"math"
	"net/mail"
	"strings"
	"time"
)

type NativeQuota struct {
	c.QuotaObservation
	Person         string    `json:"person"`
	Device         string    `json:"device"`
	ReceivedAt     time.Time `json:"received_at"`
	IdentityMethod string    `json:"identity_method"`
}

func (s *Store) ObserveQuota(ctx context.Context, p c.Principal, v c.QuotaObservation) error {
	if p.Device == "" || p.Person == "" || p.Workspace == "" {
		return c.ErrForbidden
	}
	if !safeID.MatchString(v.EventID) || !safeID.MatchString(v.Profile) || v.Provider != "codex" || v.ObservedAt.IsZero() || v.ObservedAt.After(time.Now().Add(time.Minute)) || len(v.SourceVersion) > 512 || len(v.Plan) > 128 || len(v.Windows) > 32 {
		return c.ErrInvalid
	}
	if v.Error != "" {
		if v.Error != "unavailable" || v.Email != "" || len(v.Windows) > 0 {
			return c.ErrInvalid
		}
	} else {
		parsed, e := mail.ParseAddress(v.Email)
		if e != nil || parsed.Address != v.Email || len(v.Email) > 254 {
			return c.ErrInvalid
		}
	}
	if v.Windows == nil {
		v.Windows = []quota.Window{}
	}
	v.Email = strings.ToLower(v.Email)
	for _, w := range v.Windows {
		if !safeID.MatchString(w.Bucket) || (w.Name != "primary" && w.Name != "secondary") || w.UsedPercent != nil && (math.IsNaN(*w.UsedPercent) || math.IsInf(*w.UsedPercent, 0) || *w.UsedPercent < 0 || *w.UsedPercent > 100) || w.WindowDurationMins != nil && *w.WindowDurationMins <= 0 || w.ResetsAt != nil && *w.ResetsAt <= 0 {
			return c.ErrInvalid
		}
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var version int
	if e = tx.QueryRow(ctx, "SELECT version FROM tm_policies WHERE workspace=$1 FOR SHARE", p.Workspace).Scan(&version); e != nil || version != v.PolicyVersion {
		return c.ErrForbidden
	}
	b, e := json.Marshal(v)
	if e != nil {
		return c.ErrInvalid
	}
	digest := Hash(b)
	_, e = tx.Exec(ctx, `INSERT INTO tm_native_quotas(workspace,device,event_id,person,profile,observed_at,digest,body) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, p.Workspace, p.Device, v.EventID, p.Person, v.Profile, v.ObservedAt, digest, b)
	if e != nil {
		return e
	}
	var saved string
	e = tx.QueryRow(ctx, "SELECT digest FROM tm_native_quotas WHERE workspace=$1 AND device=$2 AND event_id=$3", p.Workspace, p.Device, v.EventID).Scan(&saved)
	if e != nil {
		return e
	}
	if saved != digest {
		return c.ErrConflict
	}
	return tx.Commit(ctx)
}
func (s *Store) NativeQuotas(ctx context.Context, p c.Principal) ([]NativeQuota, error) {
	if p.Device != "" || p.Person == "" {
		return nil, c.ErrForbidden
	}
	policy, e := s.Policy(ctx, p.Workspace)
	if e != nil {
		return nil, e
	}
	all := p.Manager() || policy.Visibility == "team"
	rows, e := s.DB.Query(ctx, `SELECT body,person,device,received_at FROM (SELECT *,row_number() OVER(PARTITION BY device,profile,body->>'email' ORDER BY observed_at DESC,event_id DESC) AS n FROM tm_native_quotas WHERE workspace=$1 AND ($2 OR person=$3)) q WHERE n<=100 ORDER BY observed_at DESC,event_id DESC LIMIT 5000`, p.Workspace, all, p.Person)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []NativeQuota{}
	for rows.Next() {
		var v NativeQuota
		var b []byte
		if e = rows.Scan(&b, &v.Person, &v.Device, &v.ReceivedAt); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &v.QuotaObservation); e != nil {
			return nil, e
		}
		v.IdentityMethod = "observed_profile_email_not_session_binding"
		out = append(out, v)
	}
	return out, rows.Err()
}
