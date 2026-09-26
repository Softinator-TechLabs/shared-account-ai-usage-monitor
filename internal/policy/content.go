// Package policy applies the same explicit content rules on device and server.
package policy

import (
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"regexp"
	"unicode/utf8"
)

var secretPattern = regexp.MustCompile(`(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[A-Z0-9]{16})`)

func scrub(v any) any {
	switch x := v.(type) {
	case string:
		return secretPattern.ReplaceAllString(x, "[REDACTED]")
	case []any:
		for i := range x {
			x[i] = scrub(x[i])
		}
	case map[string]any:
		for k := range x {
			x[k] = scrub(x[k])
		}
	}
	return v
}
func Apply(v c.Snapshot, policy c.Policy) (c.Snapshot, []byte, error) {
	if v.SchemaVersion != 1 || v.SourceRef == "" || len(v.SourceRef) > 512 || v.Revision == "" || len(v.Revision) > 256 || v.Client == "" || v.Coverage == "" {
		return v, nil, c.ErrInvalid
	}
	seen := map[int]bool{}
	for _, m := range v.Messages {
		if m.Ordinal < 0 || seen[m.Ordinal] || m.Role == "" || !utf8.ValidString(m.Content) {
			return v, nil, c.ErrInvalid
		}
		seen[m.Ordinal] = true
	}
	// Copy through JSON; never mutate a caller's shared message slice.
	b, err := json.Marshal(v)
	if err != nil {
		return v, nil, c.ErrInvalid
	}
	var out c.Snapshot
	if json.Unmarshal(b, &out) != nil {
		return v, nil, c.ErrInvalid
	}
	if policy.Content == "metadata" {
		out.Raw = nil
		out.ToolCalls = nil
		out.ToolCoverage = "withheld_by_policy"
		for i := range out.Messages {
			out.Messages[i].Content = ""
			out.Messages[i].Raw = nil
		}
	}
	b, err = json.Marshal(out)
	if err != nil {
		return v, nil, c.ErrInvalid
	}
	var object any
	if json.Unmarshal(b, &object) != nil {
		return v, nil, c.ErrInvalid
	}
	if policy.Redaction == "secrets" {
		object = scrub(object)
	}
	b, err = json.Marshal(object)
	if err != nil {
		return v, nil, err
	}
	err = json.Unmarshal(b, &out)
	return out, b, err
}
