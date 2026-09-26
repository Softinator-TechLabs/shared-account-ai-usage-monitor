package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
)

func TestRetentionAndRestoreLedger(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	row, e := s.Accept(ctx, alice(), fixture())
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Exec(ctx, "UPDATE tm_snapshots SET received_at=now()-interval '100 days'")
	if e = s.Expire(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Get(ctx, owner(), row.ID); !errors.Is(e, c.ErrNotFound) {
		t.Fatal("expired session survived", e)
	}
	ledger, e := s.DeletionLedger(ctx, owner())
	if e != nil || len(ledger) != 1 {
		t.Fatal("missing restore ledger", e, ledger)
	}
	fresh := testStore(t)
	if e = fresh.RestoreDeletions(ctx, owner(), ledger); e != nil {
		t.Fatal(e)
	}
	if _, e = fresh.Accept(ctx, alice(), fixture()); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("restore resurrected deleted source", e)
	}
}
