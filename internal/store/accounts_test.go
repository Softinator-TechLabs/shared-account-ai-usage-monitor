package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
	"time"
)

func TestQuotaObservationsAreNotProjectAllocations(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a := Account{Alias: "shared-1", Provider: "codex", Plan: "subscription", Assigned: []string{"alice", "bob"}}
	if e := s.PutAccount(ctx, owner(), a); e != nil {
		t.Fatal(e)
	}
	q := Quota{Account: "shared-1", Window: "weekly", UsedPercent: 75, ObservedAt: time.Now().UTC(), ResetsAt: time.Now().Add(time.Hour).UTC(), Source: "manual"}
	if e := s.AddQuota(ctx, owner(), q); e != nil {
		t.Fatal(e)
	}
	q.UsedPercent = 101
	if e := s.AddQuota(ctx, owner(), q); !errors.Is(e, c.ErrInvalid) {
		t.Fatal("bad percent", e)
	}
	rows, e := s.Accounts(ctx, owner())
	if e != nil || len(rows) != 1 {
		t.Fatal(e, rows)
	}
	if len(rows[0].Quotas) != 1 || rows[0].Quotas[0].UsedPercent != 75 {
		t.Fatal("lost observation")
	}
}
