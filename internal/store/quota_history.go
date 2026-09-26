package store

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"time"
)

// Unlike the account cards, allocation needs the interval history, including
// failed observations. Never calculate from a silently truncated history.
func (s *Store) quotaHistory(ctx context.Context, p c.Principal, all bool, start, end time.Time) ([]NativeQuota, bool, error) {
	rows, e := s.DB.Query(ctx, `SELECT body,person,device,received_at FROM tm_native_quotas WHERE workspace=$1 AND ($2 OR person=$3) AND observed_at >= $4 AND observed_at <= $5 ORDER BY observed_at,event_id LIMIT 100001`, p.Workspace, all, p.Person, start.Add(-15*time.Minute), end)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []NativeQuota{}
	for rows.Next() {
		var v NativeQuota
		var b []byte
		if e = rows.Scan(&b, &v.Person, &v.Device, &v.ReceivedAt); e != nil {
			return nil, false, e
		}
		if e = json.Unmarshal(b, &v.QuotaObservation); e != nil {
			return nil, false, e
		}
		out = append(out, v)
		if len(out) > 100000 {
			return nil, true, nil
		}
	}
	return out, false, rows.Err()
}
