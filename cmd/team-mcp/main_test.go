package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestAnalyticsReadToolBoundary(t *testing.T) {
	p, e := readToolPath("usage_analytics", map[string]string{"days": "30", "project": "A & B", "person": "alice", "client": "claude", "key": "secret"})
	if e != nil {
		t.Fatal(e)
	}
	u, _ := url.Parse(p)
	if u.Path != "/api/v1/analytics" || u.Query().Get("project") != "A & B" || u.Query().Get("days") != "30" || u.Query().Get("key") != "" {
		t.Fatal(p)
	}
	for _, d := range []string{"-1", "365", "bad"} {
		if _, e := readToolPath("usage_analytics", map[string]string{"days": d}); e == nil {
			t.Fatal(d)
		}
	}
	for _, n := range []string{"get_viewer_key", "delete_source", "set_policy"} {
		if _, e := readToolPath(n, nil); e == nil {
			t.Fatal(n)
		}
	}
	for _, id := range []string{"", "..", "../viewer-key", "x/y"} {
		if _, e := readToolPath("get_session", map[string]string{"id": id}); e == nil {
			t.Fatal(id)
		}
	}
	b, _ := json.Marshal(readTools())
	for _, n := range []string{"usage_analytics", "list_quota_observations", "list_device_viewers"} {
		if !strings.Contains(string(b), n) {
			t.Fatal(n)
		}
	}
}
