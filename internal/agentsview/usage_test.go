package agentsview

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUsageProjectionPaginationAndUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sessions":
			for _, key := range []string{"include_one_shot", "include_automated", "include_children", "include_source"} {
				if r.URL.Query().Get(key) != "true" {
					t.Errorf("missing %s", key)
				}
			}
			if r.URL.Query().Get("limit") != "100" {
				t.Error("unbounded enumeration")
			}
			if r.URL.Query().Get("cursor") == "next" {
				w.Write([]byte(`{"sessions":[{"id":"unsupported","agent":"other"}]}`))
				return
			}
			w.Write([]byte(`{"sessions":[{"id":"codex:synthetic","agent":"codex","project":"demo","git_branch":"main","message_count":4,"user_message_count":2,"transcript_revision":"r1","first_message":"PRIVATE PROMPT"}],"next_cursor":"next"}`))
		case "/api/v1/sessions/codex:synthetic/usage":
			if r.URL.Query().Get("breakdown") != "true" || r.URL.Query().Has("subagents") || r.URL.Query().Has("rollup") {
				t.Error("wrong usage query")
			}
			w.Write([]byte(`{"breakdown_count":2,"has_token_data":true,"cost_usd":99,"breakdown":[{"timestamp":"2026-09-26T10:00:00+05:30","model":"synthetic","input_tokens":11,"output_tokens":2,"cache_creation_input_tokens":3,"cache_read_input_tokens":4,"label":"PRIVATE PROMPT"},{"timestamp":"","model":"unknown","output_tokens":0}]}`))
		case "/api/v1/sessions/codex:synthetic":
			w.Write([]byte(`{"transcript_revision":"r1"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	cl, _ := New(server.URL, "")
	var captures []c.UsageCapture
	err := cl.CollectUsageEach(context.Background(), 2, nil, func(v c.UsageCapture) error { captures = append(captures, v); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(captures) != 2 {
		t.Fatalf("captures %d", len(captures))
	}
	got := captures[0]
	if got.Messages != 4 || got.Prompts != 2 || got.Coverage != "reported" || got.PolicyVersion != 2 || len(got.Points) != 2 {
		t.Fatalf("bad metadata %+v", got)
	}
	p := got.Points[0]
	if *p.InputTokens != 11 || *p.OutputTokens != 2 || *p.CacheWriteTokens != 3 || *p.CacheReadTokens != 4 || p.Timestamp != "2026-09-26T10:00:00+05:30" {
		t.Fatalf("bad counters %+v", p)
	}
	if got.Points[1].InputTokens != nil || got.Points[1].OutputTokens == nil || *got.Points[1].OutputTokens != 0 {
		t.Fatal("unknown turned into zero")
	}
	body, _ := json.Marshal(captures)
	if strings.Contains(string(body), "PRIVATE") || strings.Contains(string(body), "cost") || strings.Contains(string(body), "raw") {
		t.Fatal("private or price fields retained")
	}
	if captures[1].Coverage != "unavailable" || len(captures[1].Points) != 0 {
		t.Fatal("missing endpoint presented as reported")
	}
}

func TestUsageMovingRevisionAndIncompleteBreakdownContinue(t *testing.T) {
	for _, bad := range []string{"moving", "incomplete"} {
		t.Run(bad, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/sessions":
					w.Write([]byte(`{"sessions":[{"id":"bad","agent":"codex","transcript_revision":"one"},{"id":"good","agent":"claude"}]}`))
				case "/api/v1/sessions/bad/usage":
					if bad == "incomplete" {
						w.Write([]byte(`{"breakdown_count":2,"breakdown":[]}`))
					} else {
						w.Write([]byte(`{"breakdown_count":0,"breakdown":[]}`))
					}
				case "/api/v1/sessions/bad":
					w.Write([]byte(`{"transcript_revision":"two"}`))
				default:
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			cl, _ := New(server.URL, "")
			var got []string
			err := cl.CollectUsageEach(context.Background(), 1, nil, func(v c.UsageCapture) error { got = append(got, v.SourceRef); return nil })
			if err == nil || len(got) != 1 || got[0] != "good" {
				t.Fatalf("error %v, emitted %v", err, got)
			}
		})
	}
}
