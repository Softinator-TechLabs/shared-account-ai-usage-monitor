package usageweights

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"testing"
)

func n(v int64) *int64 { return &v }
func TestWeightsDoNotMixProvidersOrInventUnknownRates(t *testing.T) {
	p := c.UsagePoint{Model: "gpt-6-sol", InputTokens: n(1000000), OutputTokens: n(1000000), CacheReadTokens: n(1000000), CacheWriteTokens: n(0)}
	if w := Weight("codex", p); w == nil || *w != 305 {
		t.Fatal(w)
	}
	p.Effort = "high"
	if w := Weight("codex", p); w == nil || *w != 305 {
		t.Fatal("invented effort multiplier", w)
	}
	for _, client := range []string{"claude", "antigravity", "unknown"} {
		if Weight(client, p) != nil {
			t.Fatal("provider aliasing", client)
		}
	}
	p.Model = "gpt-6-sol-new-unknown"
	if Weight("codex", p) != nil {
		t.Fatal("unknown model priced")
	}
	p.Model = "gpt-6-sol"
	p.InputTokens = nil
	if Weight("codex", p) != nil {
		t.Fatal("missing count treated as zero")
	}
	p.InputTokens = n(1)
	p.CacheWriteTokens = n(1)
	if Weight("codex", p) != nil {
		t.Fatal("unsupported category priced")
	}
}
func TestClaudeSeparateCategoryRates(t *testing.T) {
	p := c.UsagePoint{Model: "claude-sonnet-4-5-20250929", InputTokens: n(1000000), OutputTokens: n(1000000), CacheReadTokens: n(1000000), CacheWriteTokens: n(1000000)}
	if w := Weight("claude", p); w == nil || *w != 22.05 {
		t.Fatal(w)
	}
	p.Model = "claude-imaginary-4-5"
	if Weight("claude", p) != nil {
		t.Fatal("unknown model priced")
	}
}
