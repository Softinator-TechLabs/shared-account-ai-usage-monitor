package agentsview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// All transcript and tool text in these HTTP fixtures is synthetic.
func TestUsageActivityProjection(t *testing.T) {
	messages := []map[string]any{
		{"ordinal": 0, "role": "user", "timestamp": "2026-09-26T01:00:00Z", "content": "PRIVATE SYNTHETIC PROMPT"},
		{"ordinal": 1, "role": "user", "timestamp": "2026-09-26T01:00:01Z", "is_system": true},
		{"ordinal": 2, "role": "assistant", "timestamp": "2026-09-26T01:00:02Z", "model": "m", "reasoning_effort": "high", "tool_calls": []any{
			map[string]any{"tool_name": "Write", "tool_use_id": "a", "input_json": `{"file_path":"private.txt","content":"PRIVATE ONE\nPRIVATE TWO\n"}`},
			map[string]any{"tool_name": "Write", "tool_use_id": "a", "input_json": `{"content":"PRIVATE ONE\nPRIVATE TWO\n"}`},
			map[string]any{"tool_name": "Bash", "input_json": `{"command":"echo PRIVATE"}`, "result_content": "+PRIVATE\n+PRIVATE"},
		}},
		{"ordinal": 3, "role": "user", "timestamp": "2026-09-26T01:00:03Z", "prompt_source": "system"},
		{"ordinal": 4, "role": "user", "timestamp": "2026-09-26T01:00:04Z", "is_automated": true},
		{"ordinal": 5, "role": "user", "timestamp": "2026-09-26T01:00:05Z", "source_subtype": "tool_result"},
		{"ordinal": 6, "role": "user", "timestamp": "2026-09-26T01:00:06Z", "model": "explicit"},
	}
	capture := collectSyntheticActivity(t, messages, "")
	if capture.ActivityCoverage != "reported" || len(capture.Activity) != 3 {
		t.Fatalf("activity: %+v", capture)
	}
	if capture.Points[0].Effort != "high" || capture.Points[1].Effort != "" {
		t.Fatalf("effort must join message_ordinal only: %+v", capture.Points)
	}
	if a := capture.Activity[0]; a.Prompts == nil || *a.Prompts != 1 || a.Model != "m" || a.Effort != "high" {
		t.Fatalf("prompt %+v", a)
	}
	if a := capture.Activity[1]; a.GeneratedLines == nil || *a.GeneratedLines != 2 {
		t.Fatalf("lines %+v", a)
	}
	if a := capture.Activity[2]; a.Model != "explicit" || a.Effort != "" {
		t.Fatalf("explicit %+v", a)
	}
	body, _ := json.Marshal(capture)
	if strings.Contains(string(body), "PRIVATE") || strings.Contains(string(body), "private.txt") || strings.Contains(string(body), "input_json") {
		t.Fatal("raw text retained")
	}
}

func collectSyntheticActivity(t *testing.T, messages []map[string]any, mode string) c.UsageCapture {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sessions":
			relationship := ""
			if mode == "subagent" || mode == "fork" {
				relationship = mode
			}
			fmt.Fprintf(w, `{"sessions":[{"id":"synthetic","agent":"codex","message_count":%d,"transcript_revision":"r1","relationship_type":%q,"parent_session_id":"parent","is_automated":false}]}`, len(messages), relationship)
		case "/api/v1/sessions/synthetic/usage":
			w.Write([]byte(`{"breakdown_count":2,"breakdown":[{"ordinal":0,"message_ordinal":2,"model":"m","input_tokens":7},{"ordinal":2,"model":"m","input_tokens":5}]}`))
		case "/api/v1/sessions/synthetic/messages":
			if mode == "unsupported" {
				w.WriteHeader(404)
				return
			}
			if mode == "failure" {
				w.WriteHeader(500)
				return
			}
			if mode == "missing" {
				w.Write([]byte(`{}`))
				return
			}
			from := 0
			fmt.Sscan(r.URL.Query().Get("from"), &from)
			if r.URL.Query().Get("direction") != "asc" || r.URL.Query().Get("limit") != "100" {
				t.Error("unbounded message query")
			}
			page := []map[string]any{}
			for _, m := range messages {
				if m["ordinal"].(int) >= from && len(page) < 100 {
					page = append(page, m)
				}
			}
			if mode == "stalled" {
				page = messages[:1]
			}
			if mode == "truncated" {
				page = []map[string]any{}
			}
			rev := "r1"
			if mode == "moving" {
				rev = "r2"
			}
			json.NewEncoder(w).Encode(map[string]any{"messages": page, "count": len(page), "transcript_revision": rev})
		case "/api/v1/sessions/synthetic":
			w.Write([]byte(`{"transcript_revision":"r1"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "")
	var capture c.UsageCapture
	if err := client.CollectUsageEach(context.Background(), 1, nil, func(v c.UsageCapture) error { capture = v; return nil }); err != nil {
		t.Fatal(err)
	}
	return capture
}

func TestUsageActivityPaginationFailuresKeepTokens(t *testing.T) {
	messages := []map[string]any{}
	for i := 0; i < 101; i++ {
		messages = append(messages, map[string]any{"ordinal": i, "role": "user", "timestamp": "2026-09-26T01:00:00Z"})
	}
	capture := collectSyntheticActivity(t, messages, "")
	if len(capture.Activity) != 101 || capture.ActivityCoverage != "reported" {
		t.Fatalf("pagination %+v", capture)
	}
	for _, mode := range []string{"unsupported", "failure", "missing", "stalled", "truncated", "moving"} {
		t.Run(mode, func(t *testing.T) {
			capture := collectSyntheticActivity(t, messages, mode)
			if len(capture.Points) != 2 || capture.Coverage != "reported" || capture.ActivityCoverage != "unavailable" || len(capture.Activity) != 0 || capture.Points[0].Effort != "" {
				t.Fatalf("must preserve tokens without partial enrichment %+v", capture)
			}
		})
	}
}

func TestUsageProposedLines(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		lines       int64
		ok          bool
	}{
		{"Write", `{"file_path":"x","content":"a\nb\n"}`, 2, true},
		{"Edit", `{"old_string":"a\nb","new_string":"a\nc\nd"}`, 3, true},
		{"MultiEdit", `{"edits":[{"new_string":"a"},{"new_string":"b\nc"}]}`, 3, true},
		{"apply_patch", `{"patch":"*** Begin Patch\n*** Add File: x\n+a\n+b\n*** Update File: y\n@@\n-old\n+new\n*** End Patch"}`, 3, true},
		{"apply_patch", `"*** Begin Patch\n*** Add File: x\n+a\n*** End Patch"`, 1, true},
		{"apply_patch", `{"patch":"stdout\n+a\n+b"}`, 0, false},
		{"Bash", `{"command":"echo '+x'"}`, 0, false},
		{"Write", `{"content":14}`, 0, false},
		{"Edit", `{"new_string":""}`, 0, true},
	} {
		t.Run(tc.name+tc.input, func(t *testing.T) {
			n, ok := proposedLines(tc.name, tc.input)
			if n != tc.lines || ok != tc.ok {
				t.Fatalf("got %d %v, want %d %v", n, ok, tc.lines, tc.ok)
			}
		})
	}
}

func TestUsageActivityMissingTimeAndHumanBoundaries(t *testing.T) {
	messages := []map[string]any{
		{"ordinal": 0, "role": "user", "timestamp": "invalid"},
		{"ordinal": 1, "role": "user", "timestamp": "2026-09-26T01:00:01Z"},
		{"ordinal": 2, "role": "user", "timestamp": "2026-09-26T01:00:02Z"},
		{"ordinal": 3, "role": "assistant", "timestamp": "2026-09-26T01:00:03Z", "model": "m", "reasoning_effort": "high"},
		{"ordinal": 4, "role": "user", "timestamp": "2026-09-26T01:00:04Z", "prompt_source": "sdk"},
		{"ordinal": 5, "role": "user", "timestamp": "2026-09-26T01:00:05Z", "source": "automated"},
	}
	got := collectSyntheticActivity(t, messages, "")
	if got.ActivityCoverage != "partial" || len(got.Activity) != 2 {
		t.Fatalf("unknown timestamp/system counted %+v", got)
	}
	if got.Activity[0].Model != "" || got.Activity[1].Model != "m" {
		t.Fatalf("association crossed next human prompt %+v", got.Activity)
	}
}

func TestUsageActivityRequiresRevisionAndBound(t *testing.T) {
	client, _ := New("http://127.0.0.1:1", "")
	for _, session := range []usageSession{{Messages: 1}, {NativeRevision: "r1", Messages: 100001}} {
		activity, efforts, coverage := client.collectUsageActivity(context.Background(), session)
		if coverage != "unavailable" || len(activity) != 0 || len(efforts) != 0 {
			t.Fatalf("unbounded or unguarded activity %v", coverage)
		}
	}
}

func TestUsageVersionedSkipPreservesSuppression(t *testing.T) {
	for _, revision := range []string{"", "r1"} {
		t.Run(revision, func(t *testing.T) {
			s := usageSession{ID: "synthetic", Agent: "codex", NativeRevision: revision}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/sessions" {
					t.Error("suppressed source read")
					w.WriteHeader(500)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"sessions": []usageSession{s}})
			}))
			defer server.Close()
			client, _ := New(server.URL, "")
			metadata, _ := json.Marshal(s)
			sum := sha256.Sum256(append([]byte("usage-v2:"), metadata...))
			want := hex.EncodeToString(sum[:])
			err := client.CollectUsageEach(context.Background(), 1, func(id, got string) (bool, error) {
				if got != want {
					t.Fatalf("revision %s != usage-v2 %s", got, want)
				}
				return true, nil
			}, func(c.UsageCapture) error { t.Fatal("suppressed source emitted"); return nil })
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUsageActivitySubagentIsNotHumanPrompt(t *testing.T) {
	messages := []map[string]any{
		{"ordinal": 0, "role": "user", "timestamp": "2026-09-26T01:00:00Z"},
		{"ordinal": 2, "role": "assistant", "timestamp": "2026-09-26T01:00:01Z", "model": "m", "reasoning_effort": "high", "tool_calls": []any{
			map[string]any{"tool_name": "Write", "tool_use_id": "child-write", "input_json": `{"content":"synthetic line"}`},
		}},
	}
	for _, relationship := range []string{"subagent", "fork"} {
		t.Run(relationship, func(t *testing.T) {
			got := collectSyntheticActivity(t, messages, relationship)
			prompts, lines := int64(0), int64(0)
			for _, activity := range got.Activity {
				if activity.Prompts != nil {
					prompts += *activity.Prompts
				}
				if activity.GeneratedLines != nil {
					lines += *activity.GeneratedLines
				}
			}
			wantPrompts := int64(1)
			if relationship == "subagent" {
				wantPrompts = 0
			}
			if prompts != wantPrompts {
				t.Fatalf("%s human prompts = %d, want %d", relationship, prompts, wantPrompts)
			}
			if lines != 1 || len(got.Points) != 2 || got.Points[0].Effort != "high" || got.ActivityCoverage != "reported" {
				t.Fatalf("child token/line evidence lost: %+v", got)
			}
		})
	}
}
