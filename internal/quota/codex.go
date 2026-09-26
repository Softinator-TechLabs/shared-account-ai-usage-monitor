// Package quota observes native subscription state without starting inference.
package quota

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("Codex account observation unavailable")

type Window struct {
	Bucket             string   `json:"bucket"`
	Name               string   `json:"name"`
	UsedPercent        *float64 `json:"used_percent"`
	WindowDurationMins *int64   `json:"window_duration_mins"`
	ResetsAt           *int64   `json:"resets_at"`
}
type Snapshot struct {
	Email         string    `json:"email"`
	Plan          string    `json:"plan"`
	SourceVersion string    `json:"source_version"`
	ObservedAt    time.Time `json:"observed_at"`
	Windows       []Window  `json:"windows"`
}
type Command struct {
	Executable  string
	Args        []string
	ProfileHome string
}

func ReadCodexSnapshot(ctx context.Context, executable, profileHome string) (Snapshot, error) {
	if !filepath.IsAbs(executable) || !filepath.IsAbs(profileHome) {
		return Snapshot{}, ErrUnavailable
	}
	return readCommand(ctx, Command{Executable: executable, Args: []string{"app-server", "--stdio"}, ProfileHome: profileHome})
}
func readCommand(ctx context.Context, cfg Command) (Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cfg.Executable, cfg.Args...)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "CODEX_HOME=") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "CODEX_HOME="+cfg.ProfileHome)
	cmd.Stderr = io.Discard
	in, e := cmd.StdinPipe()
	if e != nil {
		return Snapshot{}, ErrUnavailable
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		return Snapshot{}, ErrUnavailable
	}
	if cmd.Start() != nil {
		return Snapshot{}, ErrUnavailable
	}
	defer func() { in.Close(); cmd.Process.Kill(); cmd.Wait() }()
	enc := json.NewEncoder(in)
	scan := bufio.NewScanner(out)
	scan.Buffer(make([]byte, 4096), 2<<20)
	id := 0
	changed := false
	rpc := func(method string, params any, dest any) error {
		id++
		if enc.Encode(map[string]any{"id": id, "method": method, "params": params}) != nil {
			return ErrUnavailable
		}
		for scan.Scan() {
			var msg struct {
				ID     int             `json:"id"`
				Method string          `json:"method"`
				Result json.RawMessage `json:"result"`
				Error  json.RawMessage `json:"error"`
			}
			if json.Unmarshal(scan.Bytes(), &msg) != nil {
				return ErrUnavailable
			}
			if msg.Method == "account/updated" {
				changed = true
			}
			if msg.ID != id {
				continue
			}
			if len(msg.Error) > 0 && string(msg.Error) != "null" {
				return ErrUnavailable
			}
			if json.Unmarshal(msg.Result, dest) != nil {
				return ErrUnavailable
			}
			return nil
		}
		return ErrUnavailable
	}
	var init struct {
		UserAgent string `json:"userAgent"`
	}
	if rpc("initialize", map[string]any{"clientInfo": map[string]string{"name": "shared_account_usage_monitor", "version": "0.3.0"}}, &init) != nil {
		return Snapshot{}, ErrUnavailable
	}
	if enc.Encode(map[string]any{"method": "initialized", "params": map[string]any{}}) != nil {
		return Snapshot{}, ErrUnavailable
	}
	type account struct {
		Account *struct {
			Type  string `json:"type"`
			Email string `json:"email"`
			Plan  string `json:"planType"`
		} `json:"account"`
	}
	var before, after account
	if rpc("account/read", map[string]bool{"refreshToken": false}, &before) != nil || before.Account == nil || before.Account.Type != "chatgpt" || before.Account.Email == "" {
		return Snapshot{}, ErrUnavailable
	}
	changed = false
	var raw json.RawMessage
	if rpc("account/rateLimits/read", map[string]any{}, &raw) != nil {
		return Snapshot{}, ErrUnavailable
	}
	if rpc("account/read", map[string]bool{"refreshToken": false}, &after) != nil || changed || after.Account == nil || *before.Account != *after.Account {
		return Snapshot{}, ErrUnavailable
	}
	windows, e := parseWindows(raw)
	if e != nil {
		return Snapshot{}, e
	}
	return Snapshot{Email: strings.ToLower(strings.TrimSpace(before.Account.Email)), Plan: before.Account.Plan, SourceVersion: init.UserAgent, ObservedAt: time.Now().UTC(), Windows: windows}, nil
}
func parseWindows(raw []byte) ([]Window, error) {
	type bucket struct {
		ID        string  `json:"limitId"`
		Primary   *Window `json:"primary"`
		Secondary *Window `json:"secondary"`
	}
	// Provider keys are camelCase; parse explicitly instead of coercing missing numbers to zero.
	type wireWindow struct {
		Used     *float64 `json:"usedPercent"`
		Duration *int64   `json:"windowDurationMins"`
		Reset    *int64   `json:"resetsAt"`
	}
	type wireBucket struct {
		ID        string      `json:"limitId"`
		Primary   *wireWindow `json:"primary"`
		Secondary *wireWindow `json:"secondary"`
	}
	var v struct {
		Map    json.RawMessage `json:"rateLimitsByLimitId"`
		Legacy *wireBucket     `json:"rateLimits"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return nil, ErrUnavailable
	}
	buckets := map[string]wireBucket{}
	if len(v.Map) > 0 && string(v.Map) != "null" {
		if json.Unmarshal(v.Map, &buckets) != nil {
			return nil, ErrUnavailable
		}
	} else if v.Legacy != nil {
		key := v.Legacy.ID
		if key == "" {
			key = "codex"
		}
		buckets[key] = *v.Legacy
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := []Window{}
	for _, k := range keys {
		b := buckets[k]
		for _, pair := range []struct {
			name string
			v    *wireWindow
		}{{"primary", b.Primary}, {"secondary", b.Secondary}} {
			w := pair.v
			if w == nil {
				continue
			}
			if w.Used != nil && (*w.Used < 0 || *w.Used > 100) || w.Duration != nil && *w.Duration <= 0 || w.Reset != nil && *w.Reset <= 0 {
				return nil, ErrUnavailable
			}
			result = append(result, Window{Bucket: k, Name: pair.name, UsedPercent: w.Used, WindowDurationMins: w.Duration, ResetsAt: w.Reset})
		}
	}
	return result, nil
}
