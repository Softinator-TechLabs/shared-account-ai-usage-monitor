package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"strings"
	"testing"
)

func TestEmailInvitePasswordLoginAndReplay(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	inv, e := s.InviteUser(ctx, owner(), "pm@example.com", "Project Manager", "manager")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.AcceptUserInvite(ctx, inv, "short"); !errors.Is(e, c.ErrInvalid) {
		t.Fatal("weak password accepted", e)
	}
	if e = s.AcceptUserInvite(ctx, inv, "synthetic long passphrase"); e != nil {
		t.Fatal(e)
	}
	if e = s.AcceptUserInvite(ctx, inv, "a different passphrase"); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("invite replay", e)
	}
	token, e := s.PasswordLogin(ctx, "team", "PM@example.com", "synthetic long passphrase")
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.Authenticate(ctx, token)
	if e != nil || p.Person != "pm@example.com" || p.Role != "manager" {
		t.Fatal("incorrect member", p, e)
	}
	if _, e = s.PasswordLogin(ctx, "team", "pm@example.com", "wrong password"); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("bad password login", e)
	}
	if e = s.Revoke(ctx, owner(), "pm@example.com"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.PasswordLogin(ctx, "team", "pm@example.com", "synthetic long passphrase"); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("revoked login", e)
	}
}

func TestOwnerSetupRecoveryDoesNotCreateAnotherOwner(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	token, e := s.PrepareOwner(ctx, "team", "first@example.com")
	if e != nil || token == "" {
		t.Fatal("initial", e)
	}
	token, e = s.PrepareOwner(ctx, "team", "first@example.com")
	if e != nil || token == "" {
		t.Fatal("recover", e)
	}
	if _, e = s.PrepareOwner(ctx, "team", "intruder@example.com"); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("new owner after bootstrap", e)
	}
	if e = s.AcceptUserInvite(ctx, token, "synthetic owner password"); e != nil {
		t.Fatal(e)
	}
	token, e = s.PrepareOwner(ctx, "team", "first@example.com")
	if e != nil || token != "" {
		t.Fatal("reset configured password during startup", e)
	}
}

func TestExplicitBootstrapTokenIsSingleUseAndCannotResetPassword(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	seed := strings.Repeat("a1", 32)
	tok, e := s.PrepareOwner(ctx, "team", "admin@example.com", seed)
	if e != nil || tok != seed {
		t.Fatal("explicit bootstrap", e)
	}
	tok, e = s.PrepareOwner(ctx, "team", "admin@example.com", seed)
	if e != nil || tok != seed {
		t.Fatal("restart before setup", e)
	}
	if e = s.AcceptUserInvite(ctx, seed, "synthetic password example"); e != nil {
		t.Fatal(e)
	}
	tok, e = s.PrepareOwner(ctx, "team", "admin@example.com", seed)
	if e != nil || tok != "" {
		t.Fatal("restart reset password", e)
	}
	if e = s.AcceptUserInvite(ctx, seed, "different synthetic password"); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("replayed bootstrap", e)
	}
}
