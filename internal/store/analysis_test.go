package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
)

func TestScopedAgentDraftAndPolicyChange(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.SetMember(ctx, owner(), "owner", "owner", "")
	row, e := s.Accept(ctx, alice(), fixture())
	if e != nil {
		t.Fatal(e)
	}
	run, e := s.BeginAnalysis(ctx, owner(), row.ID, 0, "external-native")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinishAnalysis(ctx, run.Token, "Draft: acceptance criteria clear; outcome unknown."); e != nil {
		t.Fatal(e)
	}
	reviews, e := s.Reviews(ctx, owner(), row.ID)
	if e != nil || len(reviews) != 1 || reviews[0].ActorKind != "agent" || reviews[0].Kind != "analysis" {
		t.Fatal("agent attribution", e, reviews)
	}
	if e = s.FinishAnalysis(ctx, run.Token, "overwrite"); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("result replay allowed", e)
	}
	run, e = s.BeginAnalysis(ctx, owner(), row.ID, 0, "external-native")
	if e != nil {
		t.Fatal(e)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "self_managers"})
	if e = s.FinishAnalysis(ctx, run.Token, "after change"); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("stale visibility accepted", e)
	}
}
