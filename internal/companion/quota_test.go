package companion

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/spool"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQuotaQueueSurvivesOutageAndPolicyChange(t *testing.T) {
	outage := true
	version := 1
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/device/policy" {
			json.NewEncoder(w).Encode(c.Policy{Version: version})
			return
		}
		if outage {
			w.WriteHeader(503)
			return
		}
		count++
		json.NewEncoder(w).Encode(map[string]string{"id": "saved"})
	}))
	defer srv.Close()
	cl, _ := New(srv.URL, "synthetic")
	q := &spool.Queue{Dir: t.TempDir(), MaxBytes: 1 << 20}
	v := c.QuotaObservation{PolicyVersion: 1, EventID: "synthetic", Snapshot: quota.Snapshot{ObservedAt: time.Now()}}
	body, _ := json.Marshal(v)
	q.Append(body)
	if cl.QuotaCycle(context.Background(), q, 1, nil) == nil {
		t.Fatal("outage missing")
	}
	rows, _ := q.Pending()
	if len(rows) != 1 {
		t.Fatal("lost durable event")
	}
	outage = false
	version = 2
	if cl.QuotaCycle(context.Background(), q, 1, nil) == nil || count != 0 {
		t.Fatal("policy bypass")
	}
	version = 1
	if e := cl.QuotaCycle(context.Background(), q, 1, nil); e != nil {
		t.Fatal(e)
	}
	rows, _ = q.Pending()
	if len(rows) != 0 || count != 1 {
		t.Fatal("not delivered")
	}
}
func TestOfflineQuotaCaptureUsesAcknowledgedPolicyButDenialStopsIt(t *testing.T) {
	status := 503
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
	defer srv.Close()
	cl, _ := New(srv.URL, "synthetic")
	q := &spool.Queue{Dir: t.TempDir(), MaxBytes: 1 << 20}
	profiles := []CodexProfile{{Label: "test", Executable: "/missing/synthetic-codex", Home: t.TempDir()}}
	if e := cl.QuotaCycle(context.Background(), q, 1, profiles); !Offline(e) {
		t.Fatal(e)
	}
	rows, _ := q.Pending()
	if len(rows) != 1 {
		t.Fatal("outage dropped observed profile error")
	}
	status = 403
	cl.QuotaCycle(context.Background(), q, 1, profiles)
	rows, _ = q.Pending()
	if len(rows) != 1 {
		t.Fatal("denial allowed capture")
	}
}

func TestQuotaReadFailureDeliveredWithoutClaimingSuccessfulCycle(t *testing.T) {
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/device/policy" {
			json.NewEncoder(w).Encode(c.Policy{Version: 1})
			return
		}
		var observation c.QuotaObservation
		if err := json.NewDecoder(r.Body).Decode(&observation); err != nil {
			t.Error(err)
		}
		if observation.Error != "unavailable" {
			t.Errorf("missing safe read-failure observation: %+v", observation)
		}
		count++
		json.NewEncoder(w).Encode(map[string]string{"id": "saved"})
	}))
	defer srv.Close()
	cl, _ := New(srv.URL, "synthetic")
	q := &spool.Queue{Dir: t.TempDir(), MaxBytes: 1 << 20}
	err := cl.QuotaCycle(context.Background(), q, 1, []CodexProfile{{Label: "synthetic", Executable: "/missing/synthetic-codex", Home: t.TempDir()}})
	if err == nil {
		t.Fatal("delivered error observation falsely marked cycle successful")
	}
	rows, pendingErr := q.Pending()
	if pendingErr != nil || len(rows) != 0 || count != 1 {
		t.Fatalf("error record stranded: queued=%d delivered=%d err=%v", len(rows), count, pendingErr)
	}
}
