package contracts

import "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"

// QuotaObservation describes a profile at read time, never a historical session binding.
type QuotaObservation struct {
	quota.Snapshot
	EventID       string `json:"event_id"`
	PolicyVersion int    `json:"policy_version"`
	Provider      string `json:"provider"`
	Profile       string `json:"profile"`
	Error         string `json:"error,omitempty"`
}
