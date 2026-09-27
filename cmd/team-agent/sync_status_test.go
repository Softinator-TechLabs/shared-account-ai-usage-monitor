package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/companion"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/spool"
)

func collectorStatus(t *testing.T, path string) map[string]string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var status map[string]string
	if err = json.Unmarshal(body, &status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestAnalyticsCLITracksAttemptSuccessFailureWithoutChangingConfig(t *testing.T) {
	centralStatus := 200
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(centralStatus)
		w.Write([]byte(`{"version":1}`))
	}))
	defer central.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	av := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := collectorStatus(t, path+".analytics-status.json")
		if status["state"] != "syncing" || status["last_attempt_at"] == "" {
			t.Errorf("source read before attempt recorded: %v", status)
		}
		w.Write([]byte(`{"sessions":[]}`))
	}))
	defer av.Close()
	if err := save(path, config{Server: central.URL, Upstream: av.URL, Policy: c.Policy{Version: 1}}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "analytics-once", "--config", path}
	if err := run(); err != nil {
		t.Fatal(err)
	}
	good := collectorStatus(t, path+".analytics-status.json")
	if good["state"] != "ok" || good["last_success_at"] == "" {
		t.Fatal(good)
	}
	centralStatus = 503
	if run() == nil {
		t.Fatal("offline analytics cycle succeeded")
	}
	failed := collectorStatus(t, path+".analytics-status.json")
	if failed["state"] != "error" || failed["last_success_at"] != good["last_success_at"] || failed["last_error_at"] == "" {
		t.Fatal(failed)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("status tracking modified configuration")
	}
}

func TestQuotaCLIReportsReadFailureAfterDrainingObservation(t *testing.T) {
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/device/policy" {
			w.Write([]byte(`{"version":1}`))
			return
		}
		w.Write([]byte(`{"id":"saved"}`))
	}))
	defer central.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := save(path, config{Server: central.URL, Policy: c.Policy{Version: 1}, CodexProfiles: []companion.CodexProfile{{Label: "synthetic", Executable: "/missing/synthetic", Home: t.TempDir()}}}); err != nil {
		t.Fatal(err)
	}
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "quota-once", "--config", path}
	if run() == nil {
		t.Fatal("read failure claimed success")
	}
	status := collectorStatus(t, path+".quota-status.json")
	if status["state"] != "error" || status["last_success_at"] != "" || status["last_error_at"] == "" {
		t.Fatal(status)
	}
	rows, err := (&spool.Queue{Dir: path + ".quota.queue"}).Pending()
	if err != nil || len(rows) != 0 {
		t.Fatalf("queue not drained: %d %v", len(rows), err)
	}
}

func TestStatusWriteFailureDoesNotStopAnalyticsAndLogsNoPrivatePath(t *testing.T) {
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"version":1}`)) }))
	defer central.Close()
	reads := 0
	av := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reads++; w.Write([]byte(`{"sessions":[]}`)) }))
	defer av.Close()
	path := filepath.Join(t.TempDir(), "synthetic-private-config.json")
	if err := save(path, config{Server: central.URL, Upstream: av.URL, Policy: c.Policy{Version: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path+".analytics-status.json", 0700); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	previousLog := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previousLog)
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"team-agent", "analytics-once", "--config", path}
	if err := run(); err != nil {
		t.Fatal(err)
	}
	if reads == 0 || !strings.Contains(logs.String(), "collector status could not be saved") || strings.Contains(logs.String(), path) {
		t.Fatalf("missing generic status warning or collection blocked: reads=%d logs=%s", reads, logs.String())
	}
}
