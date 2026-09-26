package companion

import (
	"context"
	"encoding/json"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/policy"
	"os"
	"path/filepath"
	"time"
)

type UsageSource interface {
	CollectUsageEach(context.Context, int, func(string, string) (bool, error), func(c.UsageCapture) error) error
}

type UsageStats struct{ Scanned, Uploaded, Skipped, Suppressed int }
type usageCheckpointEntry struct {
	Revision       string    `json:"revision"`
	PolicyVersion  int       `json:"policy_version"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
	Suppressed     bool      `json:"suppressed,omitempty"`
}
type usageCheckpoint map[string]usageCheckpointEntry

// UsageCycle deliberately has no transcript queue dependency. Without central
// policy verification it reads no upstream usage. Captures remain reconstructible
// at the source; only durable server acknowledgements advance this checkpoint.
func (cl *Client) UsageCycle(ctx context.Context, source UsageSource, path string, version int) (UsageStats, error) {
	stats := UsageStats{}
	var acknowledgedPolicy c.Policy
	checkPolicy := func() error {
		p, err := cl.Policy(ctx)
		if err != nil {
			return err
		}
		if version < 1 || p.Version != version {
			return errors.New("policy changed; usage collection paused until explicit acknowledgement")
		}
		acknowledgedPolicy = p
		return nil
	}
	if err := checkPolicy(); err != nil {
		return stats, err
	}
	checkpoint, err := loadUsageCheckpoint(path)
	if err != nil {
		return stats, err
	}
	err = source.CollectUsageEach(ctx, version, func(id, revision string) (bool, error) {
		stats.Scanned++
		entry, ok := checkpoint[id]
		if entry.Suppressed {
			stats.Suppressed++
			return true, nil
		}
		// Reconcile even stable/missing upstream revisions: upstream reparsing and
		// retention may change coverage independently of transcript metadata.
		if ok && entry.Revision == revision && entry.PolicyVersion == version && time.Since(entry.AcknowledgedAt) >= 0 && time.Since(entry.AcknowledgedAt) < time.Hour {
			stats.Skipped++
			return true, nil
		}
		return false, checkPolicy()
	}, func(v c.UsageCapture) error {
		if err := checkPolicy(); err != nil {
			return err
		}
		originalSource := v.SourceRef
		v, _, err := policy.ApplyUsage(v, acknowledgedPolicy)
		if err != nil {
			return err
		}
		var ack struct {
			ID string `json:"id"`
		}
		err = cl.Request(ctx, "POST", "/api/v1/device/usage", v, &ack)
		suppressed := false
		if err != nil {
			var h *HTTPError
			if errors.As(err, &h) && h.Status == 410 && h.Code == "source_deleted" {
				suppressed = true
			} else {
				return err
			}
		} else if ack.ID == "" {
			return errors.New("missing durable usage acknowledgement")
		}
		checkpoint[originalSource] = usageCheckpointEntry{Revision: v.Revision, PolicyVersion: version, AcknowledgedAt: time.Now().UTC(), Suppressed: suppressed}
		if err := saveUsageCheckpoint(path, checkpoint); err != nil {
			return err
		}
		if suppressed {
			stats.Suppressed++
		} else {
			stats.Uploaded++
		}
		return nil
	})
	return stats, err
}

// UsageCheckpointCounts returns aggregate status without exposing source metadata.
func UsageCheckpointCounts(path string) (acknowledged, suppressed int, err error) {
	cp, err := loadUsageCheckpoint(path)
	if err != nil {
		return 0, 0, err
	}
	for _, entry := range cp {
		if entry.Suppressed {
			suppressed++
		} else {
			acknowledged++
		}
	}
	return acknowledged, suppressed, nil
}

func loadUsageCheckpoint(path string) (usageCheckpoint, error) {
	cp := usageCheckpoint{}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cp, nil
	}
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(data, &cp) != nil || cp == nil {
		return nil, errors.New("invalid usage checkpoint; restore it or explicitly remove it to reconcile")
	}
	for id, entry := range cp {
		if id == "" || entry.Revision == "" || entry.PolicyVersion < 1 || entry.AcknowledgedAt.IsZero() {
			return nil, errors.New("invalid usage checkpoint entry")
		}
	}
	return cp, nil
}

func saveUsageCheckpoint(path string, cp usageCheckpoint) error {
	body, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".usage-checkpoint-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(body)
	}
	if err == nil {
		err = f.Sync()
	}
	closed := f.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	// Persist the rename as well as the file contents across a crash.
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
