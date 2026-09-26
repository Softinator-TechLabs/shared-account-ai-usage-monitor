package quota

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// The subprocess emulates the documented wire protocol, not collector internals.
func TestCodexWireHelper(t *testing.T) {
	if os.Getenv("QUOTA_HELPER") != "1" {
		return
	}
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	reads := 0
	for {
		var req struct {
			ID     int            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if dec.Decode(&req) != nil {
			os.Exit(0)
		}
		var result any
		switch req.Method {
		case "initialize":
			result = map[string]any{"userAgent": "codex-test/1"}
		case "initialized":
			continue
		case "account/read":
			if req.Params["refreshToken"] != false {
				os.Exit(3)
			}
			reads++
			email := "synthetic@example.test"
			if os.Getenv("QUOTA_SWITCH") == "1" && reads > 1 {
				email = "changed@example.test"
			}
			result = map[string]any{"account": map[string]any{"type": "chatgpt", "email": email, "planType": "pro"}}
		case "account/rateLimits/read":
			if os.Getenv("QUOTA_HANG") == "1" {
				time.Sleep(time.Minute)
			}
			if os.Getenv("QUOTA_ERROR") == "1" {
				enc.Encode(map[string]any{"id": req.ID, "error": map[string]any{"message": "PRIVATE TOKEN"}})
				continue
			}
			json.Unmarshal([]byte(`{"rateLimits":{"primary":{"usedPercent":99}},"rateLimitsByLimitId":{"codex":{"primary":{"usedPercent":42,"windowDurationMins":10080,"resetsAt":1790759058},"secondary":null},"other":{"primary":{"usedPercent":null}}}}`), &result)
		default:
			os.Exit(4)
		}
		enc.Encode(map[string]any{"id": req.ID, "result": result})
	}
}
func TestReadCodexWire(t *testing.T) {
	t.Setenv("QUOTA_HELPER", "1")
	cfg := Command{Executable: os.Args[0], Args: []string{"-test.run=TestCodexWireHelper"}, ProfileHome: t.TempDir()}
	got, err := readCommand(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "synthetic@example.test" || got.Plan != "pro" || len(got.Windows) != 2 {
		t.Fatalf("unexpected snapshot %#v", got)
	}
	if *got.Windows[0].UsedPercent != 42 || got.Windows[1].UsedPercent != nil {
		t.Fatal("map precedence/null lost")
	}
	t.Setenv("QUOTA_SWITCH", "1")
	if _, err = readCommand(context.Background(), cfg); err == nil {
		t.Fatal("account changed during read accepted")
	}
}
func TestReadCodexTimeoutAndSafeError(t *testing.T) {
	t.Setenv("QUOTA_HELPER", "1")
	t.Setenv("QUOTA_HANG", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := readCommand(ctx, Command{Executable: os.Args[0], Args: []string{"-test.run=TestCodexWireHelper"}, ProfileHome: t.TempDir()}); err == nil {
		t.Fatal("timeout accepted")
	}
	t.Setenv("QUOTA_HANG", "0")
	t.Setenv("QUOTA_ERROR", "1")
	if _, err := readCommand(context.Background(), Command{Executable: os.Args[0], Args: []string{"-test.run=TestCodexWireHelper"}, ProfileHome: t.TempDir()}); err == nil || err.Error() != "Codex account observation unavailable" {
		t.Fatal("unsafe error", err)
	}
}
func TestParseWindows(t *testing.T) {
	for _, raw := range []string{`{"rateLimits":{"limitId":"codex","primary":{"usedPercent":101}}}`, `{"rateLimits":{"primary":{"usedPercent":-1}}}`} {
		if _, e := parseWindows([]byte(raw)); e == nil {
			t.Fatal("invalid percent accepted")
		}
	}
	got, e := parseWindows([]byte(`{"rateLimits":{"primary":{"usedPercent":1}},"rateLimitsByLimitId":{}}`))
	if e != nil || len(got) != 0 {
		t.Fatal("empty newer map must win")
	}
	got, e = parseWindows([]byte(`{"rateLimits":{"limitId":"codex","primary":{"usedPercent":0}}}`))
	if e != nil || len(got) != 1 || got[0].WindowDurationMins != nil {
		t.Fatal("legacy or missing duration lost")
	}
}
