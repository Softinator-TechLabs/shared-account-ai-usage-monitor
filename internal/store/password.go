package store

import (
	"context"
	"encoding/hex"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"time"
)

func emailID(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, e := mail.ParseAddress(value)
	if e != nil || address.Address != value || len(value) > 128 {
		return "", c.ErrInvalid
	}
	return value, nil
}
func (s *Store) InviteUser(ctx context.Context, p c.Principal, email, name, role string) (string, error) {
	if p.Role != "owner" || p.Device != "" {
		return "", c.ErrForbidden
	}
	email, e := emailID(email)
	if e != nil {
		return "", e
	}
	if len(name) > 160 || (role != "owner" && role != "manager" && role != "member") {
		return "", c.ErrInvalid
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	// Creating an invitation must not silently reactivate or change an existing member.
	_, e = tx.Exec(ctx, "INSERT INTO tm_members(workspace,person,role,display_name) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", p.Workspace, email, role, name)
	if e != nil {
		return "", e
	}
	token := ID()
	_, e = tx.Exec(ctx, "INSERT INTO tm_invites(digest,workspace,person,kind,expires_at) VALUES($1,$2,$3,'human',$4)", Hash([]byte(token)), p.Workspace, email, time.Now().Add(24*time.Hour))
	if e != nil {
		return "", e
	}
	return token, tx.Commit(ctx)
}
func (s *Store) AcceptUserInvite(ctx context.Context, token, password string) error {
	if len(password) < 12 || len(password) > 72 {
		return c.ErrInvalid
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var workspace, person string
	e = tx.QueryRow(ctx, `SELECT i.workspace,i.person FROM tm_invites i JOIN tm_members m ON m.workspace=i.workspace AND m.person=i.person WHERE i.digest=$1 AND i.kind='human' AND NOT i.consumed AND i.expires_at>now() AND m.active FOR UPDATE OF i`, Hash([]byte(token))).Scan(&workspace, &person)
	if errors.Is(e, pgx.ErrNoRows) {
		return c.ErrForbidden
	}
	if e != nil {
		return e
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(password), 12)
	if e != nil {
		return c.ErrInvalid
	}
	if _, e = tx.Exec(ctx, `INSERT INTO tm_passwords(workspace,person,hash) VALUES($1,$2,$3) ON CONFLICT(workspace,person) DO UPDATE SET hash=excluded.hash`, workspace, person, hash); e != nil {
		return e
	}
	// Password reset invalidates all browser/read sessions and sibling invitation links.
	if _, e = tx.Exec(ctx, "DELETE FROM tm_tokens WHERE workspace=$1 AND person=$2 AND kind<>'device'", workspace, person); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE tm_invites SET consumed=true WHERE workspace=$1 AND person=$2 AND kind='human'", workspace, person); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM tm_login_attempts WHERE workspace=$1 AND person=$2", workspace, person); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Store) PasswordLogin(ctx context.Context, workspace, email, password string) (string, error) {
	email, e := emailID(email)
	if e != nil || len(password) > 72 {
		return "", c.ErrForbidden
	}
	// Persistent per-account attempt budget; no permanent account lockout.
	var attempts int
	e = s.DB.QueryRow(ctx, `INSERT INTO tm_login_attempts(workspace,person,attempts) VALUES($1,$2,1) ON CONFLICT(workspace,person) DO UPDATE SET attempts=CASE WHEN tm_login_attempts.window_start<now()-interval '15 minutes' THEN 1 ELSE tm_login_attempts.attempts+1 END,window_start=CASE WHEN tm_login_attempts.window_start<now()-interval '15 minutes' THEN now() ELSE tm_login_attempts.window_start END RETURNING attempts`, workspace, email).Scan(&attempts)
	if e != nil {
		return "", e
	}
	if attempts > 10 {
		return "", c.ErrForbidden
	}
	var hash []byte
	e = s.DB.QueryRow(ctx, `SELECT p.hash FROM tm_passwords p JOIN tm_members m ON m.workspace=p.workspace AND m.person=p.person WHERE p.workspace=$1 AND p.person=$2 AND m.active`, workspace, email).Scan(&hash)
	if e != nil || bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
		return "", c.ErrForbidden
	}
	if _, e = s.DB.Exec(ctx, "DELETE FROM tm_login_attempts WHERE workspace=$1 AND person=$2", workspace, email); e != nil {
		return "", e
	}
	return s.IssueHumanToken(ctx, workspace, email)
}

// PrepareOwner is a local administrator bootstrap/recovery operation. It never
// creates a second owner or resets an existing password merely on server restart.
func (s *Store) PrepareOwner(ctx context.Context, workspace, email string, setupToken ...string) (string, error) {
	email, e := emailID(email)
	if e != nil {
		return "", e
	}
	if len(setupToken) > 1 {
		return "", c.ErrInvalid
	}
	tok := ID()
	if len(setupToken) == 1 && setupToken[0] != "" {
		decoded, err := hex.DecodeString(setupToken[0])
		if err != nil || len(decoded) != 32 {
			return "", c.ErrInvalid
		}
		tok = setupToken[0]
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", "bootstrap:"+workspace); e != nil {
		return "", e
	}
	var count int
	if e = tx.QueryRow(ctx, "SELECT count(*) FROM tm_members WHERE workspace=$1", workspace).Scan(&count); e != nil {
		return "", e
	}
	if count == 0 {
		if _, e = tx.Exec(ctx, "INSERT INTO tm_members(workspace,person,role,display_name) VALUES($1,$2,'owner','Owner')", workspace, email); e != nil {
			return "", e
		}
	}
	var role string
	var active, hasPassword bool
	e = tx.QueryRow(ctx, `SELECT m.role,m.active,EXISTS(SELECT 1 FROM tm_passwords WHERE workspace=m.workspace AND person=m.person) FROM tm_members m WHERE m.workspace=$1 AND m.person=$2`, workspace, email).Scan(&role, &active, &hasPassword)
	if e != nil || role != "owner" || !active {
		return "", c.ErrForbidden
	}
	if hasPassword {
		return "", tx.Commit(ctx)
	}
	if _, e = tx.Exec(ctx, "UPDATE tm_invites SET consumed=true WHERE workspace=$1 AND person=$2 AND kind='human'", workspace, email); e != nil {
		return "", e
	}
	result, e := tx.Exec(ctx, `INSERT INTO tm_invites(digest,workspace,person,kind,expires_at) VALUES($1,$2,$3,'human',$4) ON CONFLICT(digest) DO UPDATE SET consumed=false,expires_at=excluded.expires_at WHERE tm_invites.workspace=excluded.workspace AND tm_invites.person=excluded.person AND tm_invites.kind='human'`, Hash([]byte(tok)), workspace, email, time.Now().Add(24*time.Hour))
	if e == nil && result.RowsAffected() != 1 {
		return "", c.ErrForbidden
	}
	if e != nil {
		return "", e
	}
	return tok, tx.Commit(ctx)
}
