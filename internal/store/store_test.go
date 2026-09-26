package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL is required; DB integration tests must not silently skip")
	}
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 8)
	rand.Read(b)
	schema := "test_" + hex.EncodeToString(b)
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close(); admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	s := &Store{DB: db}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.SetPolicy(ctx, owner(), c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team", RetentionDays: 90}); err != nil {
		t.Fatal(err)
	}
	return s
}
func owner() c.Principal { return c.Principal{Workspace: "team", Person: "owner", Role: "owner"} }
func alice() c.Principal {
	return c.Principal{Workspace: "team", Person: "alice", Role: "member", Device: "mac", EnrolledAt: time.Now()}
}
func fixture() c.Snapshot {
	return c.Snapshot{SchemaVersion: 1, SourceRef: "codex:demo", Revision: "r1", PolicyVersion: 1, Client: "codex", Project: "demo", AccountMethod: "unknown", StartedAt: "2020-01-01T00:00:00Z", Coverage: "partial", Messages: []c.Message{{Ordinal: 0, Role: "user", Content: strings.Repeat("Hinglish में समझाओ ", 2000), Model: "fixture-model"}}}
}
func TestReplayAndRevisionConflict(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	p := alice()
	v := fixture()
	first, err := s.Accept(ctx, p, v)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Accept(ctx, p, v)
	if err != nil || first.ID != second.ID {
		t.Fatal("replay changed identity", err)
	}
	v.Messages[0].Content = "changed under same revision"
	if _, err = s.Accept(ctx, p, v); !errors.Is(err, c.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	rows, err := s.List(ctx, owner(), c.Filter{})
	if err != nil || len(rows) != 1 {
		t.Fatalf("duplicate count: %d %v", len(rows), err)
	}
}
func TestFullNoneAndHistoricalAttribution(t *testing.T) {
	s := testStore(t)
	v := fixture()
	row, err := s.Accept(context.Background(), alice(), v)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), owner(), row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Messages[0].Content != v.Messages[0].Content {
		t.Fatal("content changed")
	}
	if got.Attribution != "unknown_historical" {
		t.Fatal("historical actor invented", got.Attribution)
	}
}
func TestVisibilityDirectIDAndSearch(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	row, err := s.Accept(ctx, alice(), fixture())
	if err != nil {
		t.Fatal(err)
	}
	bob := c.Principal{Workspace: "team", Person: "bob", Role: "member"}
	if _, err = s.Get(ctx, bob, row.ID); err != nil {
		t.Fatal("team visibility", err)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "self_managers", RetentionDays: 90})
	if _, err = s.Get(ctx, bob, row.ID); !errors.Is(err, c.ErrNotFound) {
		t.Fatal("direct access leak", err)
	}
	rows, err := s.List(ctx, bob, c.Filter{Query: "Hinglish"})
	if err != nil || len(rows) != 0 {
		t.Fatal("search leak", err)
	}
	other := owner()
	other.Workspace = "other"
	if _, err = s.Get(ctx, other, row.ID); !errors.Is(err, c.ErrNotFound) {
		t.Fatal("cross workspace leak", err)
	}
}
func TestUnknownPolicyAndMetadataMode(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	v := fixture()
	v.PolicyVersion = 99
	if _, err := s.Accept(ctx, alice(), v); !errors.Is(err, c.ErrForbidden) {
		t.Fatal("old policy accepted", err)
	}
	s.SetPolicy(ctx, owner(), c.Policy{Version: 2, Content: "metadata", Redaction: "none", Visibility: "team", RetentionDays: 90})
	v.PolicyVersion = 2
	row, err := s.Accept(ctx, alice(), v)
	if err != nil {
		t.Fatal(err)
	}
	if row.Messages[0].Content != "" || len(row.Raw) != 0 || len(row.Messages[0].Raw) != 0 {
		t.Fatal("metadata leaked content")
	}
}

func TestOutOfOrderSnapshotsAndPriorDiscussionsRemainDiscoverable(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newer := fixture()
	newer.Revision = "new"
	newer.Messages = append(newer.Messages, c.Message{Ordinal: 1, Role: "assistant", Content: "new response"})
	n, e := s.Accept(ctx, alice(), newer)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddReview(ctx, owner(), Review{SessionID: n.ID, Ordinal: 0, Kind: "comment", ActorKind: "human", Body: "Synthetic coaching"}); e != nil {
		t.Fatal(e)
	}
	older := fixture()
	older.Revision = "old"
	if _, e = s.Accept(ctx, alice(), older); e != nil {
		t.Fatal(e)
	}
	rows, e := s.List(ctx, owner(), c.Filter{Query: "new response"})
	if e != nil || len(rows) != 1 || rows[0].ID != n.ID {
		t.Fatal("offline older receipt hid newer searchable evidence", e, len(rows))
	}
	rows, e = s.List(ctx, owner(), c.Filter{})
	if e != nil || len(rows) != 2 {
		t.Fatal("unordered revisions must remain discoverable", e, len(rows))
	}
	found := false
	for _, row := range rows {
		reviews, e := s.Reviews(ctx, owner(), row.ID)
		if e != nil {
			t.Fatal(e)
		}
		if len(reviews) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("prior coaching no longer discoverable")
	}
}

func TestActivityUsesBoundedPreviewWhileFullCaptureIsPreserved(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	v := fixture()
	v.Messages[0].Content = strings.Repeat("हिंग्लिश synthetic full text ", 5000)
	saved, e := s.Accept(ctx, alice(), v)
	if e != nil {
		t.Fatal(e)
	}
	rows, e := s.List(ctx, owner(), c.Filter{})
	if e != nil {
		t.Fatal(e)
	}
	body, e := json.Marshal(rows)
	if e != nil {
		t.Fatal(e)
	}
	if len(body) > 8000 {
		t.Fatal("activity downloaded full transcript instead of preview", len(body))
	}
	full, e := s.Get(ctx, owner(), saved.ID)
	if e != nil || full.Messages[0].Content != v.Messages[0].Content {
		t.Fatal("preview truncated archive", e)
	}
}
