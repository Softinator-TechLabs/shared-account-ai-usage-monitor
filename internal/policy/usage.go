package policy

import (
	"bytes"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
)

// ApplyUsage applies the acknowledged metadata policy without mutating the
// caller's points. Usage captures have no transcript content to strip, but their
// identifiers and recorded project/model labels can still contain secrets.
func ApplyUsage(v c.UsageCapture, p c.Policy) (c.UsageCapture, []byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return v, nil, c.ErrInvalid
	}
	if p.Redaction == "secrets" {
		var object any
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.UseNumber()
		if err = decoder.Decode(&object); err != nil {
			return v, nil, c.ErrInvalid
		}
		b, err = json.Marshal(scrub(object))
		if err != nil {
			return v, nil, err
		}
	}
	var out c.UsageCapture
	err = json.Unmarshal(b, &out)
	return out, b, err
}
