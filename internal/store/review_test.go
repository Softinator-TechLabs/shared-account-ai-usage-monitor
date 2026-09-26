package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
)

func TestReviewEvidenceAndAccess(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	row, err := s.Accept(ctx, alice(), fixture())
	if err != nil {
		t.Fatal(err)
	}
	r := Review{SessionID: row.ID, Ordinal: 0, Kind: "comment", Body: "Hinglish mein acceptance criteria samjhao", ActorKind: "human"}
	if _, err = s.AddReview(ctx, owner(), r); err != nil {
		t.Fatal(err)
	}
	r.Kind = "work_rating"
	r.Score = 4
	if _, err = s.AddReview(ctx, owner(), r); !errors.Is(err, c.ErrInvalid) {
		t.Fatal("work rating without evidence accepted", err)
	}
	r.Evidence = "https://github.com/example/synthetic/pull/1"
	if _, err = s.AddReview(ctx, owner(), r); err != nil {
		t.Fatal(err)
	}
	r.Kind = "prompt_rating"
	r.Score = 6
	if _, err = s.AddReview(ctx, owner(), r); !errors.Is(err, c.ErrInvalid) {
		t.Fatal("invalid score", err)
	}
	r.Kind = "comment"
	r.Score = 0
	r.ActorKind = "agent"
	if _, err = s.AddReview(ctx, owner(), r); !errors.Is(err, c.ErrInvalid) {
		t.Fatal("human forged agent", err)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "self_managers"})
	bob := c.Principal{Workspace: "team", Person: "bob", Role: "member"}
	if _, err = s.Reviews(ctx, bob, row.ID); !errors.Is(err, c.ErrNotFound) {
		t.Fatal("review leak", err)
	}
}
func TestDeletePurgesDerivedAndBlocksReplay(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	row, err := s.Accept(ctx, alice(), fixture())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddReview(ctx, owner(), Review{SessionID: row.ID, Ordinal: 0, Kind: "comment", Body: "quoted prompt", ActorKind: "human"}); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteSource(ctx, owner(), row.SourceRef); err != nil {
		t.Fatal(err)
	}
	var count int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM tm_reviews").Scan(&count)
	if count != 0 {
		t.Fatal("derived text survived")
	}
	if _, err = s.Accept(ctx, alice(), fixture()); !errors.Is(err, c.ErrForbidden) {
		t.Fatal("deleted source replayed", err)
	}
}
