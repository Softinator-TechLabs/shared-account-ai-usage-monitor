package store

import (
	"context"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"time"
)

type Device struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	OS            string     `json:"os"`
	EnrolledAt    time.Time  `json:"enrolled_at"`
	LastSeen      *time.Time `json:"last_seen"`
	ExpiresAt     time.Time  `json:"expires_at"`
	PolicyVersion int        `json:"policy_version"`
}
type PersonProject struct {
	Name     string `json:"name"`
	Sessions int    `json:"sessions"`
	LastDay  string `json:"last_day"`
}
type Person struct {
	Projects []PersonProject `json:"projects"`
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Role     string          `json:"role"`
	Active   bool            `json:"active"`
	Sessions int             `json:"sessions"`
	Devices  []Device        `json:"devices"`
}

func (s *Store) People(ctx context.Context, p c.Principal) ([]Person, error) {
	if p.Device != "" {
		return nil, c.ErrForbidden
	}
	policy, e := s.Policy(ctx, p.Workspace)
	if e != nil {
		return nil, e
	}
	rows, e := s.DB.Query(ctx, `SELECT m.person,m.display_name,m.role,m.active,(SELECT count(DISTINCT source_ref) FROM tm_snapshots WHERE workspace=m.workspace AND owner_id=m.person) FROM tm_members m WHERE m.workspace=$1 AND ($2 OR m.person=$3) ORDER BY m.display_name,m.person`, p.Workspace, p.Manager() || policy.Visibility == "team", p.Person)
	if e != nil {
		return nil, e
	}
	out := []Person{}
	for rows.Next() {
		var person Person
		if e = rows.Scan(&person.ID, &person.Name, &person.Role, &person.Active, &person.Sessions); e != nil {
			rows.Close()
			return nil, e
		}
		out = append(out, person)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for i := range out {
		rs, e := s.DB.Query(ctx, `SELECT digest,device,device_os,enrolled_at,last_seen,expires_at,ack_version FROM tm_tokens WHERE workspace=$1 AND person=$2 AND kind='device' ORDER BY enrolled_at`, p.Workspace, out[i].ID)
		if e != nil {
			return nil, e
		}
		out[i].Devices = []Device{}
		for rs.Next() {
			var d Device
			if e = rs.Scan(&d.ID, &d.Name, &d.OS, &d.EnrolledAt, &d.LastSeen, &d.ExpiresAt, &d.PolicyVersion); e != nil {
				rs.Close()
				return nil, e
			}
			out[i].Devices = append(out[i].Devices, d)
		}
		e = rs.Err()
		rs.Close()
		if e != nil {
			return nil, e
		}
	}
	zone, _ := time.LoadLocation("Asia/Kolkata")
	from := time.Now().In(zone).AddDate(0, 0, -13).Format("2006-01-02")
	through := time.Now().In(zone).Format("2006-01-02")
	for i := range out {
		out[i].Projects = []PersonProject{}
		rs, err := s.DB.Query(ctx, `WITH latest AS (
 SELECT DISTINCT ON (s.source_ref) m.body FROM tm_snapshots s
 JOIN tm_snapshot_metrics m ON m.snapshot_id=s.id
 WHERE s.workspace=$1 AND s.owner_id=$2
 ORDER BY s.source_ref,s.received_at DESC,s.id DESC
) SELECT body->>'project',count(*),max(body->>'day') FROM latest
 WHERE body->>'day'>=$3 AND body->>'day'<=$4 AND coalesce(body->>'project','')<>''
 GROUP BY body->>'project' ORDER BY max(body->>'day') DESC,count(*) DESC,body->>'project' LIMIT 5`, p.Workspace, out[i].ID, from, through)
		if err != nil {
			return nil, err
		}
		for rs.Next() {
			var project PersonProject
			if err = rs.Scan(&project.Name, &project.Sessions, &project.LastDay); err != nil {
				rs.Close()
				return nil, err
			}
			out[i].Projects = append(out[i].Projects, project)
		}
		err = rs.Err()
		rs.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (s *Store) Heartbeat(ctx context.Context, token, osName string) error {
	if len(osName) > 64 {
		return c.ErrInvalid
	}
	result, e := s.DB.Exec(ctx, `UPDATE tm_tokens t SET last_seen=now(),device_os=$2 FROM tm_members m WHERE t.digest=$1 AND t.kind='device' AND t.expires_at>now() AND m.workspace=t.workspace AND m.person=t.person AND m.active`, Hash([]byte(token)), osName)
	if e == nil && result.RowsAffected() == 0 {
		return c.ErrForbidden
	}
	return e
}
func (s *Store) RevokeDevice(ctx context.Context, p c.Principal, id string) error {
	if !p.Manager() {
		return c.ErrForbidden
	}
	_, e := s.DB.Exec(ctx, "DELETE FROM tm_tokens WHERE workspace=$1 AND digest=$2 AND kind='device'", p.Workspace, id)
	return e
}
