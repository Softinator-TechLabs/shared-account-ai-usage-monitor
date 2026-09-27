package contracts

import "time"

type UsagePoint struct {
	Timestamp        string `json:"timestamp"`
	Model            string `json:"model"`
	Effort           string `json:"effort,omitempty"`
	InputTokens      *int64 `json:"input_tokens"`
	OutputTokens     *int64 `json:"output_tokens"`
	CacheReadTokens  *int64 `json:"cache_read_tokens"`
	CacheWriteTokens *int64 `json:"cache_write_tokens"`
}

// UsageActivity contains projected counters, never prompt text or tool input.
type UsageActivity struct {
	Timestamp      string `json:"timestamp"`
	Model          string `json:"model"`
	Effort         string `json:"effort,omitempty"`
	Prompts        *int64 `json:"prompts"`
	GeneratedLines *int64 `json:"generated_lines"`
}
type UsageCapture struct {
	SourceRef        string          `json:"source_ref"`
	Revision         string          `json:"revision"`
	PolicyVersion    int             `json:"policy_version"`
	Client           string          `json:"client"`
	Project          string          `json:"project"`
	Branch           string          `json:"branch"`
	StartedAt        string          `json:"started_at"`
	EndedAt          string          `json:"ended_at,omitempty"`
	ObservedAt       time.Time       `json:"observed_at"`
	Messages         int             `json:"messages"`
	Prompts          int             `json:"prompts"`
	Coverage         string          `json:"coverage"`
	Points           []UsagePoint    `json:"points"`
	Activity         []UsageActivity `json:"activity,omitempty"`
	ActivityCoverage string          `json:"activity_coverage,omitempty"`
}
