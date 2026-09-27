package companion

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func readStatusFields(t *testing.T, path string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]string
	if err = json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	for key, value := range fields {
		switch key {
		case "state":
		case "last_attempt_at", "last_success_at", "last_error_at", "last_upload_at":
			parsed, err := time.Parse(time.RFC3339Nano, value)
			if err != nil || parsed.Location() != time.UTC {
				t.Fatalf("non-UTC timestamp %s=%s", key, value)
			}
		default:
			t.Fatalf("status leaked unexpected field %s", key)
		}
	}
	return fields
}

func TestSyncStatusPreservesLastSuccessAcrossAttemptAndFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.analytics-status.json")
	writer := SyncStatusWriter{Path: path}
	if err := writer.Begin(); err != nil {
		t.Fatal(err)
	}
	if got := readStatusFields(t, path); got["state"] != "syncing" || got["last_success_at"] != "" {
		t.Fatal(got)
	}
	if err := writer.Uploaded(); err != nil {
		t.Fatal(err)
	}
	progress := readStatusFields(t, path)
	if progress["state"] != "syncing" || progress["last_upload_at"] == "" || progress["last_success_at"] != "" {
		t.Fatal(progress)
	}
	if err := writer.Finish(nil); err != nil {
		t.Fatal(err)
	}
	good := readStatusFields(t, path)
	if good["state"] != "ok" || good["last_success_at"] == "" {
		t.Fatal(good)
	}
	// A new process must preserve the receipt from the earlier completed cycle.
	writer = SyncStatusWriter{Path: path}
	if err := writer.Begin(); err != nil {
		t.Fatal(err)
	}
	if got := readStatusFields(t, path); got["last_success_at"] != good["last_success_at"] || got["state"] != "syncing" {
		t.Fatal(got)
	}
	if err := writer.Finish(errors.New("synthetic-secret /private/profile@example.invalid")); err != nil {
		t.Fatal(err)
	}
	failed := readStatusFields(t, path)
	if failed["state"] != "error" || failed["last_success_at"] != good["last_success_at"] || failed["last_error_at"] == "" {
		t.Fatal(failed)
	}
}

func TestSyncStatusAtomicPrivateReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.quota-status.json")
	if err := os.WriteFile(path, []byte(`{"state":"ok","last_success_at":"2026-09-26T00:00:00Z","token":"synthetic-secret"}`), 0644); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer previous.Close()
	if err := (SyncStatusWriter{Path: path}).Begin(); err != nil {
		t.Fatal(err)
	}
	got := readStatusFields(t, path)
	if got["last_success_at"] != "2026-09-26T00:00:00Z" {
		t.Fatal(got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	oldInfo, err := previous.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(oldInfo, info) {
		t.Fatal("status updated in place instead of atomic replacement")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files retained: %v %v", entries, err)
	}
}

func TestSyncStatusRejectsCorruptionWithoutErasingPreviousReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	body := []byte(`{"last_success_at":`)
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := (SyncStatusWriter{Path: path}).Begin(); err == nil {
		t.Fatal("corruption silently overwritten")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(body) {
		t.Fatalf("receipt changed: %s %v", got, err)
	}
}
