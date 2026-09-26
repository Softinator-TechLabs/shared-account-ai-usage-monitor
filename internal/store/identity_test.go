package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
	"time"
)

func TestEnrollmentAckAndRevocation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	invite, err := s.Invite(ctx, owner(), "alice", "device", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.Enroll(ctx, invite, "laptop", 1)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Authenticate(ctx, token)
	if err != nil || p.Person != "alice" || p.Device != "laptop" {
		t.Fatal("wrong identity", p, err)
	}
	if _, err = s.Enroll(ctx, invite, "other", 1); !errors.Is(err, c.ErrForbidden) {
		t.Fatal("invitation replay", err)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "team"})
	if _, err = s.Authenticate(ctx, token); !errors.Is(err, c.ErrForbidden) {
		t.Fatal("stale acknowledgement", err)
	}
	if err = s.Acknowledge(ctx, token, 2); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authenticate(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err = s.Revoke(ctx, owner(), "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authenticate(ctx, token); !errors.Is(err, c.ErrForbidden) {
		t.Fatal("revoked token", err)
	}
}
func TestInvitationExpiryAndRole(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if _, err := s.Invite(ctx, alice(), "bob", "device", time.Now().Add(time.Hour)); !errors.Is(err, c.ErrForbidden) {
		t.Fatal(err)
	}
	token, err := s.Invite(ctx, owner(), "alice", "device", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Enroll(ctx, token, "pc", 1); !errors.Is(err, c.ErrForbidden) {
		t.Fatal("expired invitation", err)
	}
}
