package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"strings"
	"testing"
)

func TestChunkedContentAtomicAndConflict(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	v := fixture()
	v.Messages[0].Content = strings.Repeat("हिंग्लिश payload ", 90000)
	body, _ := json.Marshal(v)
	hash := Hash(body)
	parts := (len(body) + (1 << 20) - 1) / (1 << 20)
	for i := 0; i < parts; i++ {
		end := (i + 1) * (1 << 20)
		if end > len(body) {
			end = len(body)
		}
		if e := s.SaveChunk(ctx, alice(), hash, i, parts, body[i*(1<<20):end]); e != nil {
			t.Fatal(e)
		}
		if i < parts-1 {
			rows, _ := s.List(ctx, owner(), c.Filter{})
			if len(rows) != 0 {
				t.Fatal("partial content visible")
			}
		}
	}
	row, e := s.CommitChunks(ctx, alice(), hash)
	if e != nil {
		t.Fatal(e)
	}
	if len(row.Messages) != 1 {
		t.Fatal("missing assembled message")
	}
	if row.Messages[0].Content != v.Messages[0].Content {
		t.Fatal("chunk loss")
	}
}

func TestChunkedInvalidUTF8RejectedAndDeletionIsTerminal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	v := fixture()
	v.Messages[0].Content = "SYNTHETIC_INVALID_MARKER"
	body, _ := json.Marshal(v)
	body = bytes.Replace(body, []byte("SYNTHETIC_INVALID_MARKER"), []byte{0xff}, 1)
	hash := Hash(body)
	if e := s.SaveChunk(ctx, alice(), hash, 0, 1, body); e != nil {
		t.Fatal(e)
	}
	if _, e := s.CommitChunks(ctx, alice(), hash); !errors.Is(e, c.ErrInvalid) {
		t.Fatal("chunk invalid UTF-8 accepted", e)
	}
	row, e := s.Accept(ctx, alice(), fixture())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.DeleteSource(ctx, owner(), row.SourceRef); e != nil {
		t.Fatal(e)
	}
	body, _ = json.Marshal(fixture())
	hash = Hash(body)
	if e = s.SaveChunk(ctx, alice(), hash, 0, 1, body); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitChunks(ctx, alice(), hash); !errors.Is(e, c.ErrDeleted) {
		t.Fatal("deletion not distinguishable", e)
	}
	var count int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM tm_chunks WHERE hash=$1", hash).Scan(&count)
	if count != 0 {
		t.Fatal("deleted source staging retained")
	}
}
