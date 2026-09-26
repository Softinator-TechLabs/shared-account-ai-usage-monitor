package main

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyticsOnceIndependentOfTranscriptQueueAndLock(t *testing.T) {
	uploads := 0
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/device/policy":
			w.Write([]byte(`{"version":1}`))
		case "/api/v1/device/usage":
			uploads++
			w.Write([]byte(`{"id":"ack"}`))
		default:
			t.Errorf("analytics touched transcript route %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer central.Close()
	av := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/sessions" {
			w.Write([]byte(`{"sessions":[{"id":"synthetic","agent":"codex"}]}`))
		} else {
			w.WriteHeader(404)
		}
	}))
	defer av.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := save(path, config{Server: central.URL, Upstream: av.URL, Policy: c.Policy{Version: 1}}); err != nil {
		t.Fatal(err)
	}
	os.Mkdir(path+".lock", 0700)
	os.Mkdir(path+".queue", 0700)
	os.WriteFile(filepath.Join(path+".queue", "broken.json"), []byte(`invalid transcript spool`), 0600)
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "analytics-once", "--config", path}
	if err := run(); err != nil {
		t.Fatal(err)
	}
	if uploads != 1 {
		t.Fatalf("uploads=%d", uploads)
	}
	if _, err := os.Stat(path + ".analytics.checkpoint.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".lock"); err != nil {
		t.Fatal("transcript lock was touched")
	}
}

func TestPolicyAckMustPauseAnalyticsCollector(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := save(path, config{Server: "http://127.0.0.1:1", Policy: c.Policy{Version: 1}}); err != nil {
		t.Fatal(err)
	}
	os.Mkdir(path+".analytics.lock", 0700)
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "ack", "--config", path, "--ack-version", "2"}
	err := run()
	if err == nil || err.Error() != "pause all collectors before changing policy or discarding queues" {
		t.Fatalf("expected collector lock error, got %v", err)
	}
}
