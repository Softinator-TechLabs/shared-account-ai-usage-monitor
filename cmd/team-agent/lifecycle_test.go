package main

import (
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/spool"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewRevokedCollector(t *testing.T) {
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer central.Close()
	reads := 0
	av := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		if r.URL.Path == "/api/v1/sessions" {
			w.Write([]byte(`{"sessions":[{"id":"codex:revoked","agent":"codex"}]}`))
			return
		}
		w.Write([]byte(`{"messages":[{"ordinal":0,"role":"user","content":"SYNTHETIC after revocation"}]}`))
	}))
	defer av.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	b, _ := json.Marshal(config{Server: central.URL, Token: "synthetic-revoked", Upstream: av.URL, Policy: c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"}})
	os.WriteFile(path, b, 0600)
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "once", "--config", path}
	run()
	q := spool.Queue{Dir: path + ".queue"}
	pending, _ := q.Pending()
	if reads > 0 || len(pending) > 0 {
		t.Fatalf("explicit central 403 ignored: upstream_reads=%d queued_revisions=%d", reads, len(pending))
	}
}

func TestPendingQueueDrainsWhenUpstreamFails(t *testing.T) {
	posts := 0
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ingest" {
			posts++
			w.Write([]byte(`{"id":"ack"}`))
			return
		}
		w.Write([]byte(`{"version":1}`))
	}))
	defer central.Close()
	av := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer av.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	b, _ := json.Marshal(config{Server: central.URL, Token: "fixture", Upstream: av.URL, Policy: c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"}})
	os.WriteFile(path, b, 0600)
	q := spool.Queue{Dir: path + ".queue"}
	b, _ = json.Marshal(c.Snapshot{PolicyVersion: 1})
	q.Append(b)
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "once", "--config", path}
	run()
	pending, _ := q.Pending()
	if posts != 1 || len(pending) != 0 {
		t.Fatal("upstream failure blocked durable queue", posts, len(pending))
	}
}

func TestHeartbeatPrecedesSlowInitialBackfill(t *testing.T) {
	heartbeats := 0
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/device/heartbeat" {
			heartbeats++
			w.Write([]byte(`{"ok":true}`))
			return
		}
		w.Write([]byte(`{"version":1}`))
	}))
	defer central.Close()
	av := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if heartbeats == 0 {
			t.Error("device appears offline throughout initial backfill")
		}
		w.Write([]byte(`{"sessions":[]}`))
	}))
	defer av.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	b, _ := json.Marshal(config{Server: central.URL, Token: "synthetic", Upstream: av.URL, Policy: c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"}})
	os.WriteFile(path, b, 0600)
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "once", "--config", path}
	if err := run(); err != nil {
		t.Fatal(err)
	}
}
