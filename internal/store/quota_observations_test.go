package store

import (
	"context"
	"errors"
	"fmt"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"
	"testing"
	"time"
)

func TestNativeQuotaReplayResetAndUnknownAttribution(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	p := c.Principal{Workspace: "w", Person: "alice", Device: "mac"}
	v := 40.0
	reset := time.Now().Add(time.Hour).Unix()
	obs := c.QuotaObservation{EventID: "event-1", PolicyVersion: 1, Profile: "default", Provider: "codex", Snapshot: quota.Snapshot{Email: "synthetic@example.test", Plan: "pro", ObservedAt: time.Now().UTC(), Windows: []quota.Window{{Bucket: "codex", Name: "primary", UsedPercent: &v, ResetsAt: &reset}}}}
	// testStore's workspace uses its own policy; mirror the actual device's workspace.
	if e := s.SetPolicy(ctx, c.Principal{Workspace: "w", Person: "owner", Role: "owner"}, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"}); e != nil {
		t.Fatal(e)
	}
	if e := s.ObserveQuota(ctx, p, obs); e != nil {
		t.Fatal(e)
	}
	if e := s.ObserveQuota(ctx, p, obs); e != nil {
		t.Fatal("replay", e)
	}
	obs.EventID = "event-2"
	obs.ObservedAt = obs.ObservedAt.Add(time.Second)
	v = 48
	if e := s.ObserveQuota(ctx, p, obs); e != nil {
		t.Fatal(e)
	}
	rows, e := s.NativeQuotas(ctx, c.Principal{Workspace: "w", Person: "owner", Role: "owner"})
	if e != nil || len(rows) != 2 {
		t.Fatal("duplicates", len(rows), e)
	}
	if rows[0].Device != "mac" || rows[0].Person != "alice" || rows[0].Profile != "default" {
		t.Fatal("lost collector provenance")
	}
	if *rows[0].Windows[0].UsedPercent-*rows[1].Windows[0].UsedPercent != 8 {
		t.Fatal("wrong delta")
	}
	obs.EventID = "event-1"
	if e := s.ObserveQuota(ctx, p, obs); !errors.Is(e, c.ErrConflict) {
		t.Fatal("conflicting replay", e)
	}
	p.Device = ""
	if e := s.ObserveQuota(ctx, p, obs); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("human upload accepted")
	}
	if _, e := s.NativeQuotas(ctx, c.Principal{Workspace: "w", Person: "alice", Device: "mac"}); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("device read allowed")
	}
}
func TestNativeQuotaPolicyAndStaleErrors(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	p := c.Principal{Workspace: "w", Person: "alice", Device: "pc"}
	s.SetPolicy(ctx, c.Principal{Workspace: "w", Person: "owner", Role: "owner"}, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "self_managers"})
	obs := c.QuotaObservation{EventID: "error-event", Provider: "codex", Profile: "default", PolicyVersion: 1, Error: "unavailable", Snapshot: quota.Snapshot{ObservedAt: time.Now().UTC()}}
	if e := s.ObserveQuota(ctx, p, obs); e != nil {
		t.Fatal(e)
	}
	rows, e := s.NativeQuotas(ctx, c.Principal{Workspace: "w", Person: "bob", Role: "member"})
	if e != nil || len(rows) != 0 {
		t.Fatal("private quota leak")
	}
	rows, e = s.NativeQuotas(ctx, c.Principal{Workspace: "w", Person: "alice", Role: "member"})
	if e != nil || len(rows) != 1 || rows[0].Error != "unavailable" || len(rows[0].Windows) != 0 {
		t.Fatal("missing source error")
	}
	obs.PolicyVersion = 2
	if e := s.ObserveQuota(ctx, p, obs); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("policy bypass")
	}
}

func TestNativeQuotaLastGoodSurvivesRepeatedFailures(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	p := c.Principal{Workspace: "w", Person: "alice", Device: "mac"}
	s.SetPolicy(ctx, c.Principal{Workspace: "w", Person: "owner", Role: "owner"}, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	v := c.QuotaObservation{EventID: "good", Provider: "codex", Profile: "default", PolicyVersion: 1, Snapshot: quota.Snapshot{Email: "synthetic@example.test", ObservedAt: time.Now().Add(-time.Hour)}}
	if e := s.ObserveQuota(ctx, p, v); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 101; i++ {
		v.EventID = fmt.Sprintf("error-%d", i)
		v.Email = ""
		v.Error = "unavailable"
		v.ObservedAt = v.ObservedAt.Add(time.Second)
		if e := s.ObserveQuota(ctx, p, v); e != nil {
			t.Fatal(e)
		}
	}
	rows, e := s.NativeQuotas(ctx, c.Principal{Workspace: "w", Person: "owner", Role: "owner"})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, r := range rows {
		if r.Email != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("last good identity evicted by errors")
	}
}
