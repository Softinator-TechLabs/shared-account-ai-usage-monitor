package policy

import (
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"strings"
	"testing"
)

func TestToolEvidenceHonorsContentPolicy(t *testing.T) {
	secret := "sk-syntheticcredential0123456789"
	snapshot := c.Snapshot{SchemaVersion: 1, SourceRef: "synthetic", Revision: "one", Client: "codex", Coverage: "full", ToolCalls: []json.RawMessage{json.RawMessage(`{"tool_name":"Write","input_json":"{\"content\":\"` + secret + `\"}"}`)}}
	for _, p := range []c.Policy{{Content: "metadata", Redaction: "none"}, {Content: "full", Redaction: "secrets"}, {Content: "full", Redaction: "none"}} {
		out, b, err := Apply(snapshot, p)
		if err != nil {
			t.Fatal(err)
		}
		if p.Content == "metadata" && (len(out.ToolCalls) != 0 || out.ToolCoverage != "withheld_by_policy") {
			t.Fatal("tool content leaked into metadata capture")
		}
		wantSecret := p.Content == "full" && p.Redaction == "none"
		if strings.Contains(string(b), secret) != wantSecret {
			t.Fatalf("incorrect tool redaction for %v", p)
		}
	}
	if !strings.Contains(string(snapshot.ToolCalls[0]), secret) {
		t.Fatal("mutated caller snapshot")
	}
}
