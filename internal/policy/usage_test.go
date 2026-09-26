package policy

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"strings"
	"testing"
)

func TestUsagePolicyPreservesUnknownCountersAndCaller(t *testing.T) {
	secret := "sk-syntheticcredential0123456789"
	n := int64(9007199254740993)
	v := c.UsageCapture{SourceRef: secret, Revision: "r1", Client: "codex", Project: secret, Branch: secret, Coverage: "reported", Points: []c.UsagePoint{{Model: secret, InputTokens: &n}}}
	for _, content := range []string{"metadata", "full"} {
		for _, redaction := range []string{"none", "secrets"} {
			out, b, err := ApplyUsage(v, c.Policy{Content: content, Redaction: redaction})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), secret) != (redaction == "none") {
				t.Fatal("incorrect metadata redaction", content, redaction)
			}
			if len(out.Points) != 1 || out.Points[0].InputTokens == nil || *out.Points[0].InputTokens != n || out.Points[0].OutputTokens != nil {
				t.Fatal("usage counter fidelity lost")
			}
			if v.SourceRef != secret || v.Points[0].Model != secret || *v.Points[0].InputTokens != n {
				t.Fatal("caller mutated")
			}
		}
	}
}
