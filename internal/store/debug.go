package store

import (
	"context"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"strings"
	"time"
)

type DebugLink struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Store) CreateDebugLink(ctx context.Context, p c.Principal, label string) (DebugLink, string, error) {
	if p.Role != "owner" || p.TokenKind != "human" || p.Device != "" {
		return DebugLink{}, "", c.ErrForbidden
	}
	label = strings.TrimSpace(label)
	if label == "" || len(label) > 80 || strings.ContainsAny(label, "\r\n\t") {
		return DebugLink{}, "", c.ErrInvalid
	}
	tok := ID()
	v := DebugLink{ID: Hash([]byte(tok)), Label: label, ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
	_, e := s.DB.Exec(ctx, `INSERT INTO tm_tokens(digest,workspace,person,kind,label,expires_at) VALUES($1,$2,$3,'debug',$4,$5)`, v.ID, p.Workspace, p.Person, v.Label, v.ExpiresAt)
	if e != nil {
		return DebugLink{}, "", e
	}
	if e = s.Audit(ctx, p, "debug.create", v.ID); e != nil {
		return DebugLink{}, "", e
	}
	return v, tok, nil
}
func (s *Store) DebugLinks(ctx context.Context, p c.Principal) ([]DebugLink, error) {
	if p.Role != "owner" || p.TokenKind != "human" || p.Device != "" {
		return nil, c.ErrForbidden
	}
	rows, e := s.DB.Query(ctx, `SELECT digest,label,expires_at FROM tm_tokens WHERE workspace=$1 AND kind='debug' AND expires_at>now() ORDER BY expires_at DESC`, p.Workspace)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []DebugLink{}
	for rows.Next() {
		var v DebugLink
		if e = rows.Scan(&v.ID, &v.Label, &v.ExpiresAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) RevokeDebugLink(ctx context.Context, p c.Principal, id string) error {
	if p.Role != "owner" || p.TokenKind != "human" || p.Device != "" {
		return c.ErrForbidden
	}
	_, e := s.DB.Exec(ctx, `DELETE FROM tm_tokens WHERE workspace=$1 AND kind='debug' AND digest=$2`, p.Workspace, id)
	if e != nil {
		return e
	}
	return s.Audit(ctx, p, "debug.revoke", id)
}
func (s *Store) DebugLogin(ctx context.Context, workspace, token string) (int, error) {
	p, e := s.Authenticate(ctx, token)
	if e != nil || p.Workspace != workspace || p.TokenKind != "debug" {
		return 0, c.ErrForbidden
	}
	var expiry time.Time
	if e = s.DB.QueryRow(ctx, "SELECT expires_at FROM tm_tokens WHERE digest=$1", Hash([]byte(token))).Scan(&expiry); e != nil {
		return 0, e
	}
	age := int(time.Until(expiry).Seconds())
	if age < 1 {
		return 0, c.ErrForbidden
	}
	if e = s.Audit(ctx, p, "debug.login", Hash([]byte(token))); e != nil {
		return 0, e
	}
	return age, nil
}
