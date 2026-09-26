// Package contracts defines the versioned boundary between device collectors and the archive.
package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrForbidden = errors.New("access denied")
var ErrDeleted = fmt.Errorf("source deleted: %w", ErrForbidden)
var ErrConflict = errors.New("revision conflict")
var ErrInvalid = errors.New("invalid request")
var ErrNotFound = errors.New("not found")

type Principal struct {
	AgentLabel string    `json:"agent_label,omitempty"`
	Workspace  string    `json:"workspace"`
	Person     string    `json:"person"`
	Role       string    `json:"role"`
	TokenKind  string    `json:"token_kind,omitempty"`
	Device     string    `json:"device,omitempty"`
	EnrolledAt time.Time `json:"enrolled_at,omitempty"`
}

func (p Principal) Manager() bool {
	return p.Device == "" && (p.Role == "owner" || p.Role == "manager")
}

type Policy struct {
	Version       int    `json:"version"`
	Content       string `json:"content"`
	Redaction     string `json:"redaction"`
	Visibility    string `json:"visibility"`
	RetentionDays int    `json:"retention_days"`
}
type Message struct {
	Ordinal   int             `json:"ordinal"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Model     string          `json:"model,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
	Raw       json.RawMessage `json:"raw,omitempty"`
}
type Snapshot struct {
	ToolCalls     []json.RawMessage `json:"tool_calls,omitempty"`
	ToolCoverage  string            `json:"tool_coverage,omitempty"`
	SchemaVersion int               `json:"schema_version"`
	SourceRef     string            `json:"source_ref"`
	Revision      string            `json:"revision"`
	PolicyVersion int               `json:"policy_version"`
	Client        string            `json:"client"`
	Project       string            `json:"project"`
	Branch        string            `json:"branch,omitempty"`
	Account       string            `json:"account,omitempty"`
	AccountMethod string            `json:"account_method"`
	StartedAt     string            `json:"started_at,omitempty"`
	Coverage      string            `json:"coverage"`
	Messages      []Message         `json:"messages"`
	Raw           json.RawMessage   `json:"raw,omitempty"`
}
type Session struct {
	Snapshot
	ID          string    `json:"id"`
	Owner       string    `json:"owner"`
	ObservedBy  string    `json:"observed_by"`
	Attribution string    `json:"attribution"`
	ReceivedAt  time.Time `json:"received_at"`
}

// SessionSummary is deliberately bounded; full source content is available through Get/export.
type SessionSummary struct {
	ID           string    `json:"id"`
	SourceRef    string    `json:"source_ref"`
	Revision     string    `json:"revision"`
	Owner        string    `json:"owner"`
	ObservedBy   string    `json:"observed_by"`
	Attribution  string    `json:"attribution"`
	Client       string    `json:"client"`
	Project      string    `json:"project"`
	Branch       string    `json:"branch"`
	StartedAt    string    `json:"started_at"`
	ReceivedAt   time.Time `json:"received_at"`
	Preview      string    `json:"preview"`
	MessageCount int       `json:"message_count"`
}
type Filter struct {
	SourceRef string
	Query     string
	Person    string
	Project   string
	Client    string
	Before    time.Time
	Limit     int
}
