package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/jackc/pgx/v5"
	"regexp"
	"time"
)

var safeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]{0,127}$`)

func (s *Store) SetMember(ctx context.Context, p c.Principal, person, role, subject string) error {
	if p.Device != "" || p.Role != "owner" {
		return c.ErrForbidden
	}
	if p.Person == person && role != "owner" {
		return c.ErrForbidden
	}
	if !safeID.MatchString(person) || (role != "owner" && role != "manager" && role != "member") {
		return c.ErrInvalid
	}
	var sub any
	if subject != "" {
		sub = subject
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO tm_members(workspace,person,role,oidc_subject) VALUES($1,$2,$3,$4) ON CONFLICT(workspace,person) DO UPDATE SET role=excluded.role,oidc_subject=excluded.oidc_subject,active=true`, p.Workspace, person, role, sub)
	return err
}
func (s *Store) Members(ctx context.Context, p c.Principal) ([]map[string]any, error) {
	if !p.Manager() {
		return nil, c.ErrForbidden
	}
	rows, err := s.DB.Query(ctx, "SELECT person,role,active FROM tm_members WHERE workspace=$1 ORDER BY person", p.Workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var person, role string
		var active bool
		if err = rows.Scan(&person, &role, &active); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"person": person, "role": role, "active": active})
	}
	return out, rows.Err()
}
func (s *Store) Invite(ctx context.Context, p c.Principal, person, kind string, expires time.Time) (string, error) {
	if p.Device != "" || p.Role != "owner" {
		return "", c.ErrForbidden
	}
	if !safeID.MatchString(person) || kind != "device" {
		return "", c.ErrInvalid
	}
	if _, err := s.DB.Exec(ctx, "INSERT INTO tm_members(workspace,person,role) VALUES($1,$2,'member') ON CONFLICT DO NOTHING", p.Workspace, person); err != nil {
		return "", err
	}
	token := ID()
	_, err := s.DB.Exec(ctx, "INSERT INTO tm_invites(digest,workspace,person,kind,expires_at) VALUES($1,$2,$3,$4,$5)", Hash([]byte(token)), p.Workspace, person, kind, expires)
	return token, err
}
func (s *Store) Enroll(ctx context.Context, invitation, device string, version int) (string, error) {
	if !safeID.MatchString(device) {
		return "", c.ErrInvalid
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var workspace, person string
	err = tx.QueryRow(ctx, `SELECT i.workspace,i.person FROM tm_invites i JOIN tm_members m ON m.workspace=i.workspace AND m.person=i.person JOIN tm_policies p ON p.workspace=i.workspace WHERE i.digest=$1 AND i.kind='device' AND NOT i.consumed AND i.expires_at>now() AND m.active AND p.version=$2 FOR UPDATE OF i`, Hash([]byte(invitation)), version).Scan(&workspace, &person)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", c.ErrForbidden
	}
	if err != nil {
		return "", err
	}
	token := ID()
	_, err = tx.Exec(ctx, "INSERT INTO tm_tokens(digest,workspace,person,kind,device,ack_version,expires_at) VALUES($1,$2,$3,'device',$4,$5,$6)", Hash([]byte(token)), workspace, person, device, version, time.Now().AddDate(0, 3, 0))
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, "UPDATE tm_invites SET consumed=true WHERE digest=$1", Hash([]byte(invitation))); err != nil {
		return "", err
	}
	return token, tx.Commit(ctx)
}
func (s *Store) Authenticate(ctx context.Context, token string) (c.Principal, error) {
	var p c.Principal
	err := s.DB.QueryRow(ctx, `SELECT t.workspace,t.person,m.role,t.device,t.enrolled_at,t.kind,t.label FROM tm_tokens t JOIN tm_members m ON m.workspace=t.workspace AND m.person=t.person LEFT JOIN tm_policies p ON p.workspace=t.workspace WHERE t.digest=$1 AND t.expires_at>now() AND m.active AND (t.kind<>'device' OR t.ack_version=p.version)`, Hash([]byte(token))).Scan(&p.Workspace, &p.Person, &p.Role, &p.Device, &p.EnrolledAt, &p.TokenKind, &p.AgentLabel)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, c.ErrForbidden
	}
	return p, err
}
func (s *Store) Acknowledge(ctx context.Context, token string, version int) error {
	result, err := s.DB.Exec(ctx, `UPDATE tm_tokens t SET ack_version=$2 FROM tm_policies p,tm_members m WHERE t.digest=$1 AND t.kind='device' AND t.expires_at>now() AND p.workspace=t.workspace AND p.version=$2 AND m.workspace=t.workspace AND m.person=t.person AND m.active`, Hash([]byte(token)), version)
	if err == nil && result.RowsAffected() == 0 {
		return c.ErrForbidden
	}
	return err
}
func (s *Store) Revoke(ctx context.Context, p c.Principal, person string) error {
	if p.Device != "" || p.Role != "owner" || p.Person == person {
		return c.ErrForbidden
	}
	_, err := s.DB.Exec(ctx, "UPDATE tm_members SET active=false WHERE workspace=$1 AND person=$2", p.Workspace, person)
	return err
}
func (s *Store) HumanToken(ctx context.Context, workspace, subject string) (string, error) {
	var person string
	err := s.DB.QueryRow(ctx, "SELECT person FROM tm_members WHERE workspace=$1 AND oidc_subject=$2 AND active", workspace, subject).Scan(&person)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", c.ErrForbidden
	}
	if err != nil {
		return "", err
	}
	return s.IssueHumanToken(ctx, workspace, person)
}

// IssueHumanToken is called only after verified OIDC/password identity or loopback synthetic demo login.
func (s *Store) IssueHumanToken(ctx context.Context, workspace, person string) (string, error) {
	token := ID()
	result, err := s.DB.Exec(ctx, `INSERT INTO tm_tokens(digest,workspace,person,kind,expires_at) SELECT $1,workspace,person,'human',$4 FROM tm_members WHERE workspace=$2 AND person=$3 AND active`, Hash([]byte(token)), workspace, person, time.Now().Add(8*time.Hour))
	if err == nil && result.RowsAffected() != 1 {
		return "", c.ErrForbidden
	}
	return token, err
}
func (s *Store) Logout(ctx context.Context, token string) error {
	_, err := s.DB.Exec(ctx, "DELETE FROM tm_tokens WHERE digest=$1", Hash([]byte(token)))
	return err
}

// DevicePolicy remains readable after a policy changes, before acknowledgement.
func (s *Store) DevicePolicy(ctx context.Context, token string) (c.Policy, error) {
	var workspace string
	err := s.DB.QueryRow(ctx, `SELECT t.workspace FROM tm_tokens t JOIN tm_members m ON m.workspace=t.workspace AND m.person=t.person WHERE t.digest=$1 AND t.kind='device' AND t.expires_at>now() AND m.active`, Hash([]byte(token))).Scan(&workspace)
	if errors.Is(err, pgx.ErrNoRows) {
		return c.Policy{}, c.ErrForbidden
	}
	if err != nil {
		return c.Policy{}, err
	}
	return s.Policy(ctx, workspace)
}

// ReadToken is scoped to GET routes for read-only MCP clients.
func (s *Store) ReadToken(ctx context.Context, p c.Principal) (string, error) {
	if p.Device != "" {
		return "", c.ErrForbidden
	}
	token := ID()
	_, e := s.DB.Exec(ctx, `INSERT INTO tm_tokens(digest,workspace,person,kind,expires_at) VALUES($1,$2,$3,'read',$4)`, Hash([]byte(token)), p.Workspace, p.Person, time.Now().Add(8*time.Hour))
	return token, e
}
