package macsetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Status contains only local service state and explicitly recorded sync receipts.
// It never includes enrollment, process arguments, log contents or credentials.
type Status struct {
	Configured bool            `json:"configured"`
	State      string          `json:"state"`
	Action     string          `json:"action"`
	CheckedAt  string          `json:"checked_at"`
	Services   []ServiceStatus `json:"services"`
	Analytics  SyncStatus      `json:"analytics"`
	Quota      SyncStatus      `json:"quota"`
}
type ServiceStatus struct {
	Name         string `json:"name"`
	Installed    bool   `json:"installed"`
	Loaded       bool   `json:"loaded"`
	Disabled     bool   `json:"disabled"`
	State        string `json:"state"`
	PID          int    `json:"pid,omitempty"`
	LastExitCode *int   `json:"last_exit_code,omitempty"`
}
type SyncStatus struct {
	State         string `json:"state"`
	LastAttemptAt string `json:"last_attempt_at,omitempty"`
	LastSuccessAt string `json:"last_success_at,omitempty"`
	LastErrorAt   string `json:"last_error_at,omitempty"`
	LastUploadAt  string `json:"last_upload_at,omitempty"`
}

var errServiceNotLoaded = errors.New("service is not loaded")

type statusProbe func(context.Context, ...string) ([]byte, error)

func ReadStatus(ctx context.Context, home string) Status {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	probe := func(ctx context.Context, args ...string) ([]byte, error) {
		b, err := exec.CommandContext(ctx, "/bin/launchctl", args...).CombinedOutput()
		// launchctl distinguishes a missing service from a failed inspection.
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 113 {
			err = errServiceNotLoaded
		}
		return b, err
	}
	return readStatus(ctx, home, probe, time.Now().UTC())
}
func readStatus(ctx context.Context, home string, probe statusProbe, now time.Time) Status {
	cfg := filepath.Join(Root(home), "team-agent.json")
	info, err := os.Stat(cfg)
	s := Status{Configured: err == nil && info.Mode().IsRegular(), State: "not_configured", CheckedAt: now.UTC().Format(time.RFC3339), Services: []ServiceStatus{}, Analytics: readSyncStatus(cfg + ".analytics-status.json"), Quota: readSyncStatus(cfg + ".quota-status.json")}
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	disabled, disabledErr := probe(ctx, "print-disabled", domain)
	for _, name := range serviceOrder(true) {
		info, err := os.Stat(serviceFile(home, name))
		installed := err == nil && info.Mode().IsRegular()
		label := "com.softinator.ai-usage." + name
		re := regexp.MustCompile(`(?m)^\s*"` + regexp.QuoteMeta(label) + `"\s*=>\s*true\s*$`)
		b, err := probe(ctx, "print", domain+"/"+label)
		v := launchStatus(name, installed, re.Match(disabled), b, err)
		if disabledErr != nil && !v.Loaded && installed {
			v.State = "unknown"
		}
		s.Services = append(s.Services, v)
	}
	if !s.Configured {
		return s
	}
	running, paused, off, unknown := 0, 0, 0, false
	for _, v := range s.Services {
		if v.Loaded {
			s.Action = "pause"
		}
		switch v.State {
		case "running":
			running++
		case "paused":
			paused++
		case "off":
			off++
		case "unknown":
			unknown = true
		}
	}
	if s.Action == "" && !unknown {
		s.Action = "resume"
	}
	switch {
	case unknown:
		s.State = "unknown"
	case paused == len(s.Services):
		s.State = "paused"
	case off == len(s.Services):
		s.State = "off"
	case running == len(s.Services):
		s.State = "running"
	default:
		s.State = "needs_attention"
	}
	return s
}

var launchField = regexp.MustCompile(`(?m)^\t(state|pid|last exit code) = ([^\n]+)$`)

func launchStatus(name string, installed, disabled bool, b []byte, err error) ServiceStatus {
	s := ServiceStatus{Name: name, Installed: installed, Disabled: disabled, State: "unknown"}
	if err != nil {
		if errors.Is(err, errServiceNotLoaded) {
			switch {
			case !installed:
				s.State = "not_installed"
			case disabled:
				s.State = "paused"
			default:
				s.State = "off"
			}
		}
		return s
	}
	s.Loaded = true
	s.State = "loaded"
	state := ""
	for _, v := range launchField.FindAllStringSubmatch(string(b), -1) {
		value := strings.TrimSpace(v[2])
		switch v[1] {
		case "state":
			state = value
		case "pid":
			s.PID, _ = strconv.Atoi(value)
		case "last exit code":
			if code, err := strconv.Atoi(value); err == nil {
				s.LastExitCode = &code
			}
		}
	}
	if state == "running" && s.PID > 0 {
		s.State = "running"
	} else if s.LastExitCode != nil && *s.LastExitCode != 0 {
		s.State = "error"
	}
	return s
}
func readSyncStatus(path string) SyncStatus {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SyncStatus{State: "never_synced"}
	}
	var s SyncStatus
	if err != nil || len(b) > 16384 || json.Unmarshal(b, &s) != nil {
		return SyncStatus{State: "unknown"}
	}
	for _, stamp := range []string{s.LastAttemptAt, s.LastSuccessAt, s.LastErrorAt, s.LastUploadAt} {
		if stamp != "" {
			if _, err := time.Parse(time.RFC3339, stamp); err != nil {
				return SyncStatus{State: "unknown"}
			}
		}
	}
	if s.LastAttemptAt == "" {
		return SyncStatus{State: "unknown"}
	}
	switch s.State {
	case "ok":
		if s.LastSuccessAt == "" {
			return SyncStatus{State: "unknown"}
		}
	case "error":
		if s.LastErrorAt == "" {
			return SyncStatus{State: "unknown"}
		}
	case "syncing":
	default:
		return SyncStatus{State: "unknown"}
	}
	return s
}
