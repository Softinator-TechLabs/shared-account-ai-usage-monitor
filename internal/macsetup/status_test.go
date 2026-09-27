package macsetup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStatusDoesNotConfuseLaunchdWithSuccessfulSync(t *testing.T) {
	home := t.TempDir()
	write(filepath.Join(Root(home), "team-agent.json"), []byte(`{"token":"synthetic-secret"}`), 0600)
	for _, name := range serviceOrder(true) {
		write(serviceFile(home, name), []byte("synthetic"), 0600)
	}
	probe := func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] == "print-disabled" {
			return []byte("disabled services = {}"), nil
		}
		return []byte("service = {\n\tstate = running\n\tpid = 42\n\tlast exit code = 9\n}"), nil
	}
	s := readStatus(context.Background(), home, probe, time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC))
	if !s.Configured || s.State != "running" || s.Action != "pause" {
		t.Fatalf("status %+v", s)
	}
	for _, v := range s.Services {
		if v.State != "running" || v.PID != 42 {
			t.Fatalf("service %+v", v)
		}
	}
	if s.Analytics.State != "never_synced" || s.Analytics.LastSuccessAt != "" {
		t.Fatalf("invented sync success: %+v", s.Analytics)
	}
	data, _ := json.Marshal(s)
	if strings.Contains(string(data), "synthetic-secret") {
		t.Fatal("secret leaked")
	}
}

func TestLaunchdStatusStates(t *testing.T) {
	for _, tc := range []struct {
		name, output        string
		disabled, installed bool
		err                 error
		want                string
	}{
		{"running", "job = {\n\tstate = running\n\tpid = 42\n}", false, true, nil, "running"},
		{"waiting", "job = {\n\tstate = waiting\n\tlast exit code = 0\n}", false, true, nil, "loaded"},
		{"failed", "job = {\n\tstate = waiting\n\tlast exit code = 78\n}", false, true, nil, "error"},
		{"paused", "", true, true, errServiceNotLoaded, "paused"},
		{"disabledButRunning", "job = {\n\tstate = running\n\tpid = 42\n}", true, true, nil, "running"},
		{"off", "", false, true, errServiceNotLoaded, "off"},
		{"missing", "", false, false, errServiceNotLoaded, "not_installed"},
		{"probeFailed", "", false, true, errors.New("synthetic-secret"), "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := launchStatus("analytics", tc.installed, tc.disabled, []byte(tc.output), tc.err)
			if got.State != tc.want {
				t.Fatalf("got %+v want %s", got, tc.want)
			}
		})
	}
}

func TestSyncStatusPreservesSuccessOnErrorAndRejectsInvalidEvidence(t *testing.T) {
	p := filepath.Join(t.TempDir(), "status.json")
	os.WriteFile(p, []byte(`{"state":"error","last_attempt_at":"2026-09-27T00:02:00Z","last_success_at":"2026-09-27T00:01:00Z","last_error_at":"2026-09-27T00:02:00Z","error":"secret"}`), 0600)
	s := readSyncStatus(p)
	if s.State != "error" || s.LastSuccessAt != "2026-09-27T00:01:00Z" {
		t.Fatalf("%+v", s)
	}
	b, _ := json.Marshal(s)
	if strings.Contains(string(b), "secret") {
		t.Fatal("leaked error")
	}
	for _, v := range []string{`{"state":"ok"}`, `{"state":"ok","last_success_at":"not-a-time"}`, `{broken`} {
		os.WriteFile(p, []byte(v), 0600)
		if got := readSyncStatus(p); got.State != "unknown" {
			t.Fatalf("accepted invalid evidence: %+v", got)
		}
	}
}

func TestStatusPausedRequiresEveryInstalledServiceStopped(t *testing.T) {
	home := t.TempDir()
	write(filepath.Join(Root(home), "team-agent.json"), []byte(`{}`), 0600)
	for _, name := range serviceOrder(true) {
		write(serviceFile(home, name), []byte("synthetic"), 0600)
	}
	probe := func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] == "print-disabled" {
			return []byte(`disabled services = {
 "com.softinator.ai-usage.agentsview" => true
 "com.softinator.ai-usage.analytics" => true
 "com.softinator.ai-usage.quota" => true
 "com.softinator.ai-usage.companion" => true
}`), nil
		}
		return nil, errServiceNotLoaded
	}
	s := readStatus(context.Background(), home, probe, time.Now())
	if s.State != "paused" || s.Action != "resume" {
		t.Fatalf("%+v", s)
	}
}

func TestStatusProbeFailureDoesNotClaimPausedOrOfferResume(t *testing.T) {
	home := t.TempDir()
	write(filepath.Join(Root(home), "team-agent.json"), []byte(`{}`), 0600)
	for _, name := range serviceOrder(true) {
		write(serviceFile(home, name), []byte("synthetic"), 0600)
	}
	s := readStatus(context.Background(), home, func(context.Context, ...string) ([]byte, error) { return nil, errors.New("permission denied") }, time.Now())
	if s.State != "unknown" || s.Action != "" {
		t.Fatalf("%+v", s)
	}
}

func TestSyncStatusReadsAcknowledgedSuccessAndPreservesItDuringNextAttempt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "status.json")
	for _, state := range []string{"ok", "syncing"} {
		os.WriteFile(p, []byte(`{"state":"`+state+`","last_attempt_at":"2026-09-27T00:02:00.123Z","last_success_at":"2026-09-27T00:01:00Z"}`), 0600)
		got := readSyncStatus(p)
		if got.State != state || got.LastSuccessAt != "2026-09-27T00:01:00Z" {
			t.Fatalf("%+v", got)
		}
	}
}

func TestSyncStatusReportsUploadProgressWithoutClaimingCompletedSync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	for _, stamp := range []string{"2026-09-27T00:01:30.123456789Z", "invalid"} {
		os.WriteFile(path, []byte(`{"state":"syncing","last_attempt_at":"2026-09-27T00:01:00Z","last_upload_at":"`+stamp+`"}`), 0600)
		got := readSyncStatus(path)
		data, _ := json.Marshal(got)
		var fields map[string]any
		json.Unmarshal(data, &fields)
		if stamp == "invalid" {
			if got.State != "unknown" {
				t.Fatalf("accepted invalid upload receipt: %+v", got)
			}
		} else if got.State != "syncing" || got.LastSuccessAt != "" || fields["last_upload_at"] != stamp {
			t.Fatalf("lost partial upload progress or invented success: %s", data)
		}
	}
}
