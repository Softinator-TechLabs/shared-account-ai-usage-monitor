package store

import (
	"context"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
	"time"
)

func TestPeopleProjectsDeduplicateAndRespectVisibility(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if e := s.SetMember(ctx, owner(), "alice", "member", ""); e != nil {
		t.Fatal(e)
	}
	v := fixture()
	v.StartedAt = time.Now().Format(time.RFC3339)
	v.Project = "synthetic-repo"
	for _, revision := range []string{"one", "two"} {
		v.Revision = revision
		if _, e := s.Accept(ctx, alice(), v); e != nil {
			t.Fatal(e)
		}
	}
	old := v
	old.SourceRef = "codex:old"
	old.StartedAt = time.Now().AddDate(0, 0, -20).Format(time.RFC3339)
	if _, e := s.Accept(ctx, alice(), old); e != nil {
		t.Fatal(e)
	}
	rows, e := s.People(ctx, owner())
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, p := range rows {
		if p.ID == "alice" {
			found = true
			if len(p.Projects) != 1 || p.Projects[0].Name != "synthetic-repo" || p.Projects[0].Sessions != 1 {
				t.Fatalf("project evidence: %+v", p)
			}
		}
	}
	if !found {
		t.Fatal("missing member")
	}
	if e := s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "self_managers"}); e != nil {
		t.Fatal(e)
	}
	rows, e = s.People(ctx, c.Principal{Workspace: "team", Person: "bob", Role: "member"})
	if e != nil || len(rows) != 0 {
		t.Fatal("person evidence leaked", rows, e)
	}
}
