package companion

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/spool"
	"time"
)

type CodexProfile struct {
	Label      string `json:"label"`
	Executable string `json:"executable"`
	Home       string `json:"home"`
}

// QuotaCycle has its own bounded durable queue; transcript backfill cannot starve it.
func (cl *Client) QuotaCycle(ctx context.Context, q *spool.Queue, version int, profiles []CodexProfile) error {
	policy, e := cl.Policy(ctx)
	online := e == nil
	if e != nil && !Offline(e) {
		return e
	}
	if online && policy.Version != version {
		return errors.New("policy changed; quota collection paused")
	}
	deliver := func() error {
		rows, e := q.Pending()
		if e != nil {
			return e
		}
		for _, row := range rows {
			var v c.QuotaObservation
			if json.Unmarshal(row.Body, &v) != nil || v.PolicyVersion != version {
				return errors.New("quota queue policy mismatch")
			}
		}
		for _, row := range rows {
			var ack struct {
				ID string `json:"id"`
			}
			if e = cl.Request(ctx, "POST", "/api/v1/device/quota-observations", json.RawMessage(row.Body), &ack); e != nil {
				return e
			}
			if ack.ID == "" {
				return errors.New("missing quota acknowledgement")
			}
			if e = q.Ack(row.Key); e != nil {
				return e
			}
		}
		return nil
	}
	if online {
		if e = deliver(); e != nil {
			if !Offline(e) {
				return e
			}
			online = false
		}
	}
	readFailed := false
	for _, profile := range profiles {
		v := c.QuotaObservation{EventID: quotaEventID(), PolicyVersion: version, Provider: "codex", Profile: profile.Label}
		v.Snapshot, e = quota.ReadCodexSnapshot(ctx, profile.Executable, profile.Home)
		if e != nil {
			readFailed = true
			v.Snapshot = quota.Snapshot{ObservedAt: time.Now().UTC()}
			v.Error = "unavailable"
		}
		body, e := json.Marshal(v)
		if e != nil {
			return e
		}
		if _, e = q.Append(body); e != nil {
			return e
		}
	}
	if !online {
		return ErrUnavailable
	}
	if e = deliver(); e != nil {
		return e
	}
	if readFailed {
		return errors.New("quota read unavailable; error observations delivered")
	}
	return nil
}

func quotaEventID() string {
	var b [24]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
