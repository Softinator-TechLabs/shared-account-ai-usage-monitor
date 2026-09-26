package agentsview

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Public Session API projection only: this is not a native transcript parser.
// Text is decoded for a single page and discarded; only counters survive.
type activityMessage struct {
	Ordinal       *int   `json:"ordinal"`
	Role          string `json:"role"`
	Timestamp     string `json:"timestamp"`
	Model         string `json:"model"`
	Effort        string `json:"reasoning_effort"`
	System        bool   `json:"is_system"`
	Automated     bool   `json:"is_automated"`
	Source        string `json:"source"`
	SourceType    string `json:"source_type"`
	SourceSubtype string `json:"source_subtype"`
	PromptSource  string `json:"prompt_source"`
	Tools         []struct {
		Name  string `json:"tool_name"`
		ID    string `json:"tool_use_id"`
		Input string `json:"input_json"`
	} `json:"tool_calls"`
}
type activityModel struct{ model, effort string }

func (client *Client) collectUsageActivity(ctx context.Context, s usageSession) ([]c.UsageActivity, map[int]activityModel, string) {
	unavailable := func() ([]c.UsageActivity, map[int]activityModel, string) { return nil, nil, "unavailable" }
	// A stable source revision and explicit count are required for multi-page
	// attribution. Old APIs without this contract keep tokens but no enrichment.
	if s.NativeRevision == "" || s.Messages < 0 || s.Messages > 100000 {
		return unavailable()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var out []c.UsageActivity
	models := map[int]activityModel{}
	seenTools := map[string]bool{}
	pending := -1
	from, total := 0, 0
	coverage := "reported"
	for pageIndex := 0; pageIndex <= 1000; pageIndex++ {
		var page struct {
			Messages []activityMessage `json:"messages"`
			Count    *int              `json:"count"`
			Revision string            `json:"transcript_revision"`
		}
		err := client.get(ctx, "/api/v1/sessions/"+url.PathEscape(s.ID)+"/messages", url.Values{"from": {strconv.Itoa(from)}, "limit": {"100"}, "direction": {"asc"}}, &page)
		if err != nil || page.Messages == nil || page.Count == nil || *page.Count != len(page.Messages) || len(page.Messages) > 100 || (page.Revision != "" && page.Revision != s.NativeRevision) {
			return unavailable()
		}
		for _, m := range page.Messages {
			if m.Ordinal == nil || *m.Ordinal < from || *m.Ordinal >= 1000000 {
				return unavailable()
			}
			from = *m.Ordinal + 1
			total++
			if total > s.Messages {
				return unavailable()
			}
			if m.Role == "assistant" && !m.System {
				models[*m.Ordinal] = activityModel{m.Model, m.Effort}
			}
			// Upstream marks delegated sessions independently of is_automated.
			// A parent alone is not enough: human fork sessions still count.
			human := m.Role == "user" && s.RelationshipType != "subagent" && !s.Automated && !m.System && !m.Automated && humanSource(m)
			// Associate a prompt lacking model metadata with the next actual assistant
			// response before the next human prompt. This is response association, not
			// proof of which model the human selected or an effort multiplier.
			if human {
				pending = -1
			}
			if m.Role == "assistant" && !m.System && m.Model != "" && pending >= 0 {
				out[pending].Model = m.Model
				if out[pending].Effort == "" {
					out[pending].Effort = m.Effort
				}
				pending = -1
			}
			if _, err := time.Parse(time.RFC3339Nano, m.Timestamp); err != nil {
				if human || len(m.Tools) > 0 {
					coverage = "partial"
				}
				continue
			}
			if human {
				n := int64(1)
				out = append(out, c.UsageActivity{Timestamp: m.Timestamp, Model: m.Model, Effort: m.Effort, Prompts: &n})
				if m.Model == "" {
					pending = len(out) - 1
				}
			}
			if m.Role != "assistant" || m.System {
				continue
			}
			var lines int64
			known := false
			for _, tool := range m.Tools {
				if tool.ID != "" {
					if seenTools[tool.ID] {
						continue
					}
					seenTools[tool.ID] = true
				}
				n, ok := proposedLines(tool.Name, tool.Input)
				if ok {
					known = true
					lines += n
				}
			}
			if known {
				out = append(out, c.UsageActivity{Timestamp: m.Timestamp, Model: m.Model, Effort: m.Effort, GeneratedLines: &lines})
			}
		}
		if len(page.Messages) < 100 {
			if total != s.Messages {
				return unavailable()
			}
			return out, models, coverage
		}
	}
	return unavailable()
}

func humanSource(m activityMessage) bool {
	for _, v := range []string{m.Source, m.SourceType, m.SourceSubtype, m.PromptSource} {
		switch strings.ToLower(v) {
		case "system", "automated", "automation", "tool", "tool_result", "sdk", "background", "compact_boundary", "compaction":
			return false
		}
	}
	return true
}

// proposedLines counts replacement/write text, not a net Git diff, accepted
// changes, stdout or prose. Unknown tool shapes are absent, never invented zero.
func proposedLines(name, input string) (int64, bool) {
	var fields map[string]json.RawMessage
	_ = json.Unmarshal([]byte(input), &fields)
	text := func(key string) (string, bool) {
		var v string
		raw, ok := fields[key]
		if !ok || json.Unmarshal(raw, &v) != nil {
			return "", false
		}
		return v, true
	}
	switch name {
	case "Write", "write_file", "functions.write_file":
		if v, ok := text("content"); ok {
			return textLines(v), true
		}
	case "Edit", "edit_file", "functions.edit_file":
		if v, ok := text("new_string"); ok {
			return textLines(v), true
		}
	case "MultiEdit":
		var edits []struct {
			New *string `json:"new_string"`
		}
		if json.Unmarshal(fields["edits"], &edits) != nil || edits == nil {
			return 0, false
		}
		var n int64
		for _, e := range edits {
			if e.New == nil {
				return 0, false
			}
			n += textLines(*e.New)
		}
		return n, true
	case "apply_patch", "functions.apply_patch":
		patch, ok := text("patch")
		if !ok {
			patch, ok = text("input")
		}
		if !ok {
			ok = json.Unmarshal([]byte(input), &patch) == nil
		}
		if !ok {
			return 0, false
		}
		return patchAddedLines(patch)
	}
	return 0, false
}
func textLines(s string) int64 {
	if s == "" {
		return 0
	}
	n := int64(strings.Count(s, "\n"))
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
func patchAddedLines(patch string) (int64, bool) {
	lines := strings.Split(strings.TrimSpace(patch), "\n")
	if len(lines) < 3 || lines[0] != "*** Begin Patch" || lines[len(lines)-1] != "*** End Patch" {
		return 0, false
	}
	var added int64
	inFile := false
	addFile := false
	for _, line := range lines[1 : len(lines)-1] {
		if strings.HasPrefix(line, "*** Add File: ") || strings.HasPrefix(line, "*** Update File: ") {
			inFile = true
			addFile = strings.HasPrefix(line, "*** Add File: ")
			continue
		}
		if strings.HasPrefix(line, "*** Delete File: ") {
			inFile = false
			continue
		}
		if strings.HasPrefix(line, "*** Move to: ") || line == "*** End of File" {
			if !inFile {
				return 0, false
			}
			continue
		}
		if !inFile {
			return 0, false
		}
		if strings.HasPrefix(line, "+") {
			added++
			continue
		}
		if !addFile && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "@@")) {
			continue
		}
		return 0, false
	}
	return added, true
}
