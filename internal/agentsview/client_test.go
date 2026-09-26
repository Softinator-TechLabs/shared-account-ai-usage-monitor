package agentsview

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestReadAllPagesAndFullText(t *testing.T) {
	text := strings.Repeat("Hinglish समझाओ ", 2000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/api/v1/sessions" {
			for _, k := range []string{"include_one_shot", "include_automated", "include_children"} {
				if r.URL.Query().Get(k) != "true" {
					t.Error("missing complete enumeration", k)
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"sessions": []any{map[string]any{"id": "codex:demo", "agent": "codex", "project": "demo"}}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/messages") {
			start, _ := strconv.Atoi(r.URL.Query().Get("from"))
			msgs := []any{}
			for i := start; i < start+100 && i < 201; i++ {
				msgs = append(msgs, map[string]any{"ordinal": i, "role": "user", "content": text, "model": "fixture-model", "vendor_extension": true})
			}
			json.NewEncoder(w).Encode(map[string]any{"messages": msgs, "count": len(msgs)})
			return
		}
		w.WriteHeader(404)
	}))
	defer server.Close()
	client, err := New(server.URL, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := client.Collect(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(rows[0].Messages) != 201 {
		t.Fatal("incomplete enumeration")
	}
	if rows[0].Messages[200].Content != text || !strings.Contains(string(rows[0].Messages[0].Raw), "vendor_extension") {
		t.Fatal("lost content")
	}
}
func TestRejectRemoteAndRedirect(t *testing.T) {
	if _, err := New("http://example.org", "secret"); err == nil {
		t.Fatal("remote upstream accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "http://example.org", 302) }))
	defer server.Close()
	client, _ := New(server.URL, "secret")
	if _, err := client.Collect(context.Background(), 1); err == nil {
		t.Fatal("redirect followed")
	}
}

func TestRejectInvalidUTF8BeforeJSONRepair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/sessions" {
			w.Write([]byte(`{"sessions":[{"id":"x","agent":"codex"}]}`))
			return
		}
		w.Write(append([]byte(`{"messages":[{"ordinal":0,"role":"user","content":"bad`), append([]byte{0xff}, []byte(`text"}]}`)...)...))
	}))
	defer server.Close()
	client, _ := New(server.URL, "fixture")
	if _, err := client.Collect(context.Background(), 1); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("expected explicit UTF-8 fidelity failure, got %v", err)
	}
}

func TestStreamingEmitsBeforeReadingNextSource(t *testing.T) {
	emitted := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/sessions" {
			w.Write([]byte(`{"sessions":[{"id":"one","agent":"codex"},{"id":"two","agent":"codex"}]}`))
			return
		}
		if strings.Contains(r.URL.Path, "two") && emitted != 1 {
			t.Error("second source read before first durable handoff")
		}
		w.Write([]byte(`{"messages":[{"ordinal":0,"role":"user","content":"synthetic"}]}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "")
	if err := client.CollectEach(context.Background(), 1, nil, func(_ c.Snapshot) error { emitted++; return nil }); err != nil {
		t.Fatal(err)
	}
	if emitted != 2 {
		t.Fatal("missing handoff", emitted)
	}
}
func TestChangingTranscriptRevisionIsNotAcknowledged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sessions":
			w.Write([]byte(`{"sessions":[{"id":"one","agent":"codex","transcript_revision":"1"}]}`))
		case "/api/v1/sessions/one":
			w.Write([]byte(`{"id":"one","transcript_revision":"2"}`))
		default:
			w.Write([]byte(`{"messages":[{"ordinal":0,"role":"user","content":"synthetic"}]}`))
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "")
	emitted := false
	err := client.CollectEach(context.Background(), 1, nil, func(_ c.Snapshot) error { emitted = true; return nil })
	if err == nil || emitted {
		t.Fatal("changing source silently archived", err, emitted)
	}
}
