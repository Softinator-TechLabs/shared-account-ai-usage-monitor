package macsetup

import (
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyticsServiceUsesExistingConfigAndPrivateSeparateLogs(t *testing.T) {
	home := t.TempDir()
	root := Root(home)
	if err := writeAnalyticsService(home); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(serviceFile(home, "analytics"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed any
	if err = xml.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"com.softinator.ai-usage.analytics", "analytics-run", filepath.Join(root, "team-agent.json"), "analytics.stdout.log", "analytics.stderr.log"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q", want)
		}
	}
	for _, file := range []string{serviceFile(home, "analytics"), filepath.Join(root, "analytics.stdout.log"), filepath.Join(root, "analytics.stderr.log")} {
		info, err := os.Stat(file)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("private file missing: %s %v", file, err)
		}
	}
	// Regeneration preserves log content and never creates or alters enrollment.
	log := filepath.Join(root, "analytics.stderr.log")
	os.WriteFile(log, []byte("synthetic diagnostic"), 0600)
	if err := writeAnalyticsService(home); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	if string(data) != "synthetic diagnostic" {
		t.Fatal("log erased")
	}
	if _, err := os.Stat(filepath.Join(root, "team-agent.json")); !os.IsNotExist(err) {
		t.Fatal("enrollment created")
	}
}

func TestUpgradeAndUninstallPreserveEnrollmentQueuesAndLogs(t *testing.T) {
	home := t.TempDir()
	root := Root(home)
	resources := t.TempDir()
	cfg := filepath.Join(root, "team-agent.json")
	queue := filepath.Join(root, "team-agent.json.queue", "capture.json")
	if err := write(cfg, []byte(`{"token":"synthetic-private"}`), 0600); err != nil {
		t.Fatal(err)
	}
	write(queue, []byte("synthetic queued data"), 0600)
	write(filepath.Join(root, "bin", "team-agent"), []byte("old synthetic binary"), 0700)
	write(filepath.Join(resources, "team-agent"), []byte("new synthetic binary"), 0700)
	for _, name := range []string{"agentsview", "companion", "quota"} {
		write(serviceFile(home, name), []byte("synthetic service"), 0600)
	}
	var calls []string
	service := func(_ context.Context, _ string, name string, start bool) error {
		action := "stop:"
		if start {
			action = "start:"
		}
		calls = append(calls, action+name)
		return nil
	}
	if err := upgrade(context.Background(), home, resources, service); err != nil {
		t.Fatal(err)
	}
	want := []string{"stop:analytics", "stop:quota", "stop:companion", "start:agentsview", "start:analytics", "start:quota", "start:companion"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("service order %v", calls)
	}
	binary, _ := os.ReadFile(filepath.Join(root, "bin", "team-agent"))
	if string(binary) != "new synthetic binary" {
		t.Fatal("binary not upgraded")
	}
	calls = nil
	if err := uninstall(context.Background(), home, service); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"analytics", "quota", "companion", "agentsview"} {
		if _, err := os.Stat(serviceFile(home, name)); !os.IsNotExist(err) {
			t.Fatalf("service remained %s", name)
		}
	}
	for path, want := range map[string]string{cfg: `{"token":"synthetic-private"}`, queue: "synthetic queued data"} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != want {
			t.Fatalf("private state changed %s", path)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "analytics.stderr.log")); err != nil {
		t.Fatal("diagnostic removed")
	}
}

func TestUpgradeMissingBundleDoesNotStopCollectors(t *testing.T) {
	home := t.TempDir()
	write(filepath.Join(Root(home), "team-agent.json"), []byte(`{}`), 0600)
	calls := 0
	err := upgrade(context.Background(), home, t.TempDir(), func(context.Context, string, string, bool) error { calls++; return nil })
	if err == nil || calls != 0 {
		t.Fatalf("invalid upgrade affected services: %v calls=%d", err, calls)
	}
}
