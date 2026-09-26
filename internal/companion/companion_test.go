package companion

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/spool"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestPolicyChangeBlocksOldQueue(t *testing.T) {
	q := &spool.Queue{Dir: filepath.Join(t.TempDir(), "queue")}
	b, _ := json.Marshal(c.Snapshot{PolicyVersion: 1})
	q.Append(b)
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			posts++
			w.Write([]byte(`{"id":"ack"}`))
			return
		}
		w.Write([]byte(`{"version":2,"content":"metadata","redaction":"none","visibility":"team"}`))
	}))
	defer server.Close()
	cl, _ := New(server.URL, "test")
	if err := cl.Deliver(context.Background(), q, 1); err == nil {
		t.Fatal("old policy sent")
	}
	rows, _ := q.Pending()
	if len(rows) != 1 || posts != 0 {
		t.Fatal("queue discarded or uploaded")
	}
}
func TestDeliveryReplaysAfterFailure(t *testing.T) {
	q := &spool.Queue{Dir: t.TempDir()}
	b, _ := json.Marshal(c.Snapshot{PolicyVersion: 1})
	q.Append(b)
	fail := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte(`{"version":1}`))
			return
		}
		if fail {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"id":"ack"}`))
	}))
	defer server.Close()
	cl, _ := New(server.URL, "test")
	if err := cl.Deliver(context.Background(), q, 1); err == nil {
		t.Fatal("failure hidden")
	}
	rows, _ := q.Pending()
	if len(rows) != 1 {
		t.Fatal("lost queue")
	}
	fail = false
	if err := cl.Deliver(context.Background(), q, 1); err != nil {
		t.Fatal(err)
	}
	rows, _ = q.Pending()
	if len(rows) != 0 {
		t.Fatal("ack not removed")
	}
}
func TestRefuseInsecureRemote(t *testing.T) {
	if _, e := New("http://example.com", "secret"); e == nil {
		t.Fatal("plaintext remote allowed")
	}
}

func TestDeletedSourceDoesNotBlockQueue(t *testing.T) {
	q := &spool.Queue{Dir: t.TempDir()}
	for _, ref := range []string{"deleted", "live"} {
		b, _ := json.Marshal(c.Snapshot{SourceRef: ref, PolicyVersion: 1})
		q.Append(b)
	}
	live := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte(`{"version":1}`))
			return
		}
		var s c.Snapshot
		json.NewDecoder(r.Body).Decode(&s)
		if s.SourceRef == "deleted" {
			w.WriteHeader(410)
			w.Write([]byte(`{"error":"source_deleted"}`))
			return
		}
		live++
		w.Write([]byte(`{"id":"accepted"}`))
	}))
	defer server.Close()
	cl, _ := New(server.URL, "fixture")
	if err := cl.Deliver(context.Background(), q, 1); err != nil {
		t.Fatal(err)
	}
	pending, _ := q.Pending()
	if live != 1 || len(pending) != 0 {
		t.Fatal("queue blocked", live, len(pending))
	}
}
