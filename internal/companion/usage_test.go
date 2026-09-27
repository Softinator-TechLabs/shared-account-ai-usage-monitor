package companion

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/agentsview"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUsageCycleRedactsMetadataAndCheckpointsOriginalSource(t *testing.T) {
	secret := "sk-syntheticcredential0123456789"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/sessions" {
			json.NewEncoder(w).Encode(map[string]any{"sessions": []map[string]any{{"id": secret, "agent": "codex", "project": secret, "git_branch": secret}}})
		} else {
			json.NewEncoder(w).Encode(map[string]any{"breakdown_count": 1, "breakdown": []map[string]any{{"timestamp": "2026-09-26T00:00:00Z", "model": secret, "input_tokens": 9}}})
		}
	}))
	defer upstream.Close()
	uploads := 0
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/device/policy" {
			json.NewEncoder(w).Encode(c.Policy{Version: 1, Content: "metadata", Redaction: "secrets"})
			return
		}
		uploads++
		var capture c.UsageCapture
		if err := json.NewDecoder(r.Body).Decode(&capture); err != nil {
			t.Error(err)
		}
		body, _ := json.Marshal(capture)
		if strings.Contains(string(body), secret) {
			t.Error("secret metadata reached the central upload")
		}
		if len(capture.Points) != 1 || capture.Points[0].InputTokens == nil || *capture.Points[0].InputTokens != 9 {
			t.Error("redaction changed usage counters")
		}
		json.NewEncoder(w).Encode(map[string]string{"id": capture.SourceRef})
	}))
	defer central.Close()
	cl, _ := New(central.URL, "")
	av, _ := agentsview.New(upstream.URL, "")
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if _, err := cl.UsageCycle(context.Background(), av, path, 1); err != nil {
		t.Fatal(err)
	}
	cp, err := loadUsageCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cp[secret]; !ok {
		t.Fatal("checkpoint lost original upstream identity")
	}
	if stats, err := cl.UsageCycle(context.Background(), av, path, 1); err != nil || stats.Skipped != 1 || uploads != 1 {
		t.Fatalf("redacted source was not skipped: %+v %v", stats, err)
	}
}

func TestUsageCycleCheckpointReplayPolicyAndDeletion(t *testing.T) {
	uploads, reads := 0, 0
	serverPolicy := 1
	status := 500
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sessions":
			w.Write([]byte(`{"sessions":[{"id":"synthetic","agent":"codex","message_count":1}]}`))
		case "/api/v1/sessions/synthetic/usage":
			reads++
			w.Write([]byte(`{"breakdown_count":1,"breakdown":[{"timestamp":"2026-09-26T00:00:00Z","input_tokens":9}]}`))
		default:
			t.Errorf("unexpected upstream %s", r.URL.Path)
		}
	}))
	defer upstream.Close()
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/device/policy":
			json.NewEncoder(w).Encode(map[string]int{"version": serverPolicy})
		case "/api/v1/device/usage":
			uploads++
			w.WriteHeader(status)
			if status == 200 {
				w.Write([]byte(`{"id":"durable"}`))
			} else if status == 410 {
				w.Write([]byte(`{"error":"source_deleted"}`))
			}
		default:
			t.Errorf("unexpected central %s", r.URL.Path)
		}
	}))
	defer central.Close()
	cl, _ := New(central.URL, "")
	av, _ := agentsview.New(upstream.URL, "")
	path := filepath.Join(t.TempDir(), "analytics.json")
	cycle := func(version int) (UsageStats, error) { return cl.UsageCycle(context.Background(), av, path, version) }
	if _, err := cycle(1); err == nil {
		t.Fatal("failed upload accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("failed upload checkpointed")
	}
	status = 200
	if stats, err := cycle(1); err != nil || stats.Uploaded != 1 {
		t.Fatalf("retry: %+v %v", stats, err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("checkpoint is not private")
	}
	if stats, err := cycle(1); err != nil || stats.Skipped != 1 || uploads != 2 || reads != 2 {
		t.Fatalf("unchanged source reread: %+v %v uploads=%d reads=%d", stats, err, uploads, reads)
	}
	// A changed policy must stop upstream reads and must invalidate old checkpoints.
	serverPolicy = 2
	if _, err := cycle(1); err == nil || reads != 2 {
		t.Fatal("policy change read source")
	}
	if stats, err := cycle(2); err != nil || stats.Uploaded != 1 || reads != 3 {
		t.Fatalf("policy checkpoint not invalidated: %+v %v", stats, err)
	}
	// Missing native revision still reconciles periodically.
	cp, err := loadUsageCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := cp["synthetic"]
	entry.AcknowledgedAt = time.Now().Add(-2 * time.Hour)
	cp["synthetic"] = entry
	if err = saveUsageCheckpoint(path, cp); err != nil {
		t.Fatal(err)
	}
	status = 410
	if stats, err := cycle(2); err != nil || stats.Suppressed != 1 {
		t.Fatalf("deletion: %+v %v", stats, err)
	}
	before := reads
	if _, err := cycle(2); err != nil || reads != before {
		t.Fatal("deleted source recollected")
	}
}

func TestUsageCycleRefusesMissingAckAndCorruptCheckpoint(t *testing.T) {
	reads := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		if r.URL.Path == "/api/v1/sessions" {
			w.Write([]byte(`{"sessions":[{"id":"s","agent":"codex"}]}`))
		} else {
			w.WriteHeader(404)
		}
	}))
	defer upstream.Close()
	central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/device/policy" {
			w.Write([]byte(`{"version":1}`))
		} else {
			w.Write([]byte(`{}`))
		}
	}))
	defer central.Close()
	cl, _ := New(central.URL, "")
	av, _ := agentsview.New(upstream.URL, "")
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if _, err := cl.UsageCycle(context.Background(), av, path, 1); err == nil {
		t.Fatal("missing acknowledgement accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("missing acknowledgement persisted")
	}
	os.WriteFile(path, []byte(`broken`), 0600)
	before := reads
	if _, err := cl.UsageCycle(context.Background(), av, path, 1); err == nil || reads != before {
		t.Fatal("corrupt checkpoint silently ignored")
	}
}

// Synthetic source yields one capture and then can fail the remainder of a backfill.
type partialUsageSource struct{ err error }

func (s partialUsageSource) CollectUsageEach(ctx context.Context, version int, skip func(string, string) (bool, error), emit func(c.UsageCapture) error) error {
	if _, err := skip("synthetic", "revision"); err != nil {
		return err
	}
	if err := emit(c.UsageCapture{SourceRef: "synthetic", Revision: "revision", PolicyVersion: version, Client: "codex"}); err != nil {
		return err
	}
	return s.err
}

func TestUsageProgressOnlyAfterDurableAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		wantProgress   int
	}{
		{"acknowledged", `{"id":"durable"}`, 1},
		{"missing acknowledgement", `{}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			central := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/device/policy" {
					w.Write([]byte(`{"version":1,"content":"full","redaction":"none"}`))
					return
				}
				w.Write([]byte(tc.response))
			}))
			defer central.Close()
			cl, _ := New(central.URL, "")
			dir := t.TempDir()
			status := SyncStatusWriter{Path: filepath.Join(dir, "status.json")}
			if err := status.Begin(); err != nil {
				t.Fatal(err)
			}
			progress := 0
			stats, err := cl.UsageCycleWithProgress(context.Background(), partialUsageSource{err: errors.New("later source failed")}, filepath.Join(dir, "checkpoint.json"), 1, func() {
				progress++
				if err := status.Uploaded(); err != nil {
					t.Error(err)
				}
				fields := readStatusFields(t, status.Path)
				if fields["state"] != "syncing" || fields["last_success_at"] != "" {
					t.Errorf("partial upload claimed completion: %v", fields)
				}
			})
			if err == nil || progress != tc.wantProgress || stats.Uploaded != tc.wantProgress {
				t.Fatalf("stats=%+v progress=%d err=%v", stats, progress, err)
			}
			if err := status.Finish(err); err != nil {
				t.Fatal(err)
			}
			fields := readStatusFields(t, status.Path)
			if fields["state"] != "error" || fields["last_success_at"] != "" {
				t.Fatal(fields)
			}
		})
	}
}
