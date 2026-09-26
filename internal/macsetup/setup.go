// Package macsetup installs only this application's per-user services.
package macsetup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Policy struct {
	Version    int    `json:"version"`
	Content    string `json:"content"`
	Redaction  string `json:"redaction"`
	Visibility string `json:"visibility"`
	Retention  int    `json:"retention_days"`
}
type Invitation struct {
	Server     string `json:"server"`
	Person     string `json:"person"`
	Invitation string `json:"invitation"`
	Policy     Policy `json:"policy"`
}

func (v Invitation) Validate() error {
	u, e := url.Parse(v.Server)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return errors.New("enter the HTTPS workspace address, without a path")
	}
	p := v.Policy
	if v.Invitation == "" || p.Version < 1 || (p.Content != "full" && p.Content != "metadata") || (p.Redaction != "none" && p.Redaction != "secrets") || (p.Visibility != "team" && p.Visibility != "self_managers") || p.Retention < 0 {
		return errors.New("connection file is incomplete; download a new one from Connect device")
	}
	return nil
}
func ReadInvitation(path, server string) (Invitation, error) {
	var v Invitation
	b, e := os.ReadFile(path)
	if e != nil {
		return v, errors.New("cannot read the connection file")
	}
	if len(b) > 65536 {
		return v, errors.New("connection file is too large")
	}
	if e = json.Unmarshal(b, &v); e != nil {
		return v, errors.New("choose an .aiusage or device-invitation.json connection file")
	}
	if v.Server == "" {
		v.Server = strings.TrimRight(server, "/")
	}
	return v, v.Validate()
}

var checksums = map[string]string{"arm64": "b21fdf093e631237e39d33ab80fe50fbc799e0bac11ac8ede8e0dc988ae12303", "amd64": "95df0f56c039c394601d755301bbf523e56e0a8d2a648287b80cc203897f3fc2"}

func verify(b []byte, want string) error {
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != want {
		return errors.New("AgentsView checksum did not match; installation stopped")
	}
	return nil
}
func extract(b []byte) ([]byte, error) {
	z, e := gzip.NewReader(bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	defer z.Close()
	r := tar.NewReader(z)
	for {
		h, e := r.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if filepath.Base(h.Name) != "agentsview" {
			continue
		}
		if h.Typeflag != tar.TypeReg || h.Size > 256<<20 {
			return nil, errors.New("unsupported AgentsView archive entry")
		}
		data, e := io.ReadAll(io.LimitReader(r, 256<<20+1))
		if e != nil {
			return nil, e
		}
		if int64(len(data)) != h.Size {
			return nil, errors.New("incomplete AgentsView archive")
		}
		return data, nil
	}
	return nil, errors.New("AgentsView executable missing")
}
func Root(home string) string {
	return filepath.Join(home, "Library", "Application Support", "Shared Account AI Usage Monitor")
}
func write(path string, b []byte, mode os.FileMode) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".setup-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
func command(ctx context.Context, args ...string) error {
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Env = []string{"HOME=" + os.Getenv("HOME"), "PATH=/usr/bin:/bin:/usr/sbin:/sbin"}
	if e := c.Run(); e != nil {
		return fmt.Errorf("%s did not complete; retry or inspect the private setup log", filepath.Base(args[0]))
	}
	return nil
}
func serviceFile(home, name string) string {
	return filepath.Join(home, "Library", "LaunchAgents", "com.softinator.ai-usage."+name+".plist")
}
func plist(home, root, name string, args []string) []byte {
	esc := func(s string) string { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }
	var a strings.Builder
	for _, v := range args {
		a.WriteString("<string>" + esc(v) + "</string>")
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>com.softinator.ai-usage.` + name + `</string><key>ProgramArguments</key><array>` + a.String() + `</array><key>RunAtLoad</key><true/><key>KeepAlive</key><true/><key>ThrottleInterval</key><integer>20</integer><key>StandardOutPath</key><string>` + esc(filepath.Join(root, name+".stdout.log")) + `</string><key>StandardErrorPath</key><string>` + esc(filepath.Join(root, name+".stderr.log")) + `</string></dict></plist>`)
}

// serviceAction keeps launchd's disabled state in sync with explicit user intent.
// A bootout alone would restart collection at the next login.
func serviceAction(ctx context.Context, domain, file, target string, start bool, exists func() bool, run func(context.Context, ...string) error) error {
	if start {
		if e := run(ctx, "/bin/launchctl", "enable", target); e != nil {
			return e
		}
		if exists() {
			return nil
		}
		return run(ctx, "/bin/launchctl", "bootstrap", domain, file)
	}
	if e := run(ctx, "/bin/launchctl", "disable", target); e != nil {
		return e
	}
	if !exists() {
		return nil
	}
	return run(ctx, "/bin/launchctl", "bootout", target)
}
func Service(ctx context.Context, home, name string, start bool) error {
	if name != "agentsview" && name != "companion" && name != "quota" && name != "analytics" {
		return errors.New("unknown background service")
	}
	if start && (name == "quota" || name == "analytics") {
		if _, e := os.Stat(serviceFile(home, name)); os.IsNotExist(e) {
			return nil
		}
	}
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	target := domain + "/com.softinator.ai-usage." + name
	exists := func() bool { return exec.CommandContext(ctx, "/bin/launchctl", "print", target).Run() == nil }
	return serviceAction(ctx, domain, serviceFile(home, name), target, start, exists, command)
}
func Install(ctx context.Context, home, resources, file, server string, ack int, progress func(string)) error {
	if runtime.GOOS != "darwin" {
		return errors.New("this installer is for macOS")
	}
	v, e := ReadInvitation(file, server)
	if e != nil {
		return e
	}
	if ack != v.Policy.Version {
		return errors.New("review and acknowledge the collection policy first")
	}
	root := Root(home)
	cfg := filepath.Join(root, "team-agent.json")
	if _, e = os.Stat(cfg); !os.IsNotExist(e) {
		return errors.New("this Mac is already configured; use Resume sync or manage the device in your workspace")
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return e
	}
	if e = os.Chmod(root, 0700); e != nil {
		return e
	}
	lock := filepath.Join(root, "setup.lock")
	if e = os.Mkdir(lock, 0700); e != nil {
		return errors.New("another setup is running; close it before retrying")
	}
	defer os.Remove(lock)
	progress("Downloading verified AgentsView 0.44.0…")
	req, e := http.NewRequestWithContext(ctx, "GET", "https://github.com/kenn-io/agentsview/releases/download/v0.44.0/agentsview_0.44.0_darwin_"+runtime.GOARCH+".tar.gz", nil)
	if e != nil {
		return e
	}
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 8 {
			return errors.New("unsafe download redirect")
		}
		return nil
	}}
	res, e := client.Do(req)
	if e != nil {
		return errors.New("AgentsView download failed; check your connection and retry")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errors.New("AgentsView release download is unavailable")
	}
	archive, e := io.ReadAll(io.LimitReader(res.Body, 256<<20+1))
	if e != nil {
		return e
	}
	if e = verify(archive, checksums[runtime.GOARCH]); e != nil {
		return e
	}
	binary, e := extract(archive)
	if e != nil {
		return e
	}
	bin := filepath.Join(root, "bin")
	av := filepath.Join(bin, "agentsview-0.44.0")
	if e = write(av, binary, 0700); e != nil {
		return e
	}
	agent := filepath.Join(bin, "team-agent")
	b, e := os.ReadFile(filepath.Join(resources, "team-agent"))
	if e != nil {
		return errors.New("app bundle is incomplete; download it again")
	}
	if e = write(agent, b, 0700); e != nil {
		return e
	}
	progress("Preparing private local storage…")
	secret := make([]byte, 32)
	if _, e = rand.Read(secret); e != nil {
		return e
	}
	token := hex.EncodeToString(secret)
	avdir := filepath.Join(root, "agentsview")
	tokenfile := filepath.Join(avdir, "api-token")
	if e = write(tokenfile, []byte(token), 0600); e != nil {
		return e
	}
	avconfig := fmt.Sprintf("archive_content = \"full\"\nrequire_auth = true\nauth_token = %q\ndisable_update_check = true\nresult_content_blocked_categories = []\n", token) + disabledAgents
	if e = write(filepath.Join(avdir, "config.toml"), []byte(avconfig), 0600); e != nil {
		return e
	}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return e
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	privateInvite := filepath.Join(root, "setup-invitation.json")
	b, _ = json.Marshal(v)
	if e = write(privateInvite, b, 0600); e != nil {
		return e
	}
	defer os.Remove(privateInvite)
	for _, name := range []string{"agentsview", "companion", "quota"} {
		for _, suffix := range []string{".stdout.log", ".stderr.log"} {
			p := filepath.Join(root, name+suffix)
			if _, e = os.Stat(p); os.IsNotExist(e) {
				if e = write(p, nil, 0600); e != nil {
					return e
				}
			}
		}
	}
	avargs := []string{"/usr/bin/env", "-i", "HOME=" + home, "PATH=/usr/bin:/bin:/usr/sbin:/sbin", "AGENTSVIEW_DATA_DIR=" + avdir, av, "serve", "--host", "127.0.0.1", "--port", fmt.Sprint(port), "--no-browser", "--require-auth", "--no-update-check"}
	agentargs := []string{"/usr/bin/env", "-i", "HOME=" + home, "PATH=/usr/bin:/bin:/usr/sbin:/sbin", agent, "run", "--config", cfg}
	if e = write(serviceFile(home, "agentsview"), plist(home, root, "agentsview", avargs), 0600); e != nil {
		return e
	}
	if e = write(serviceFile(home, "companion"), plist(home, root, "companion", agentargs), 0600); e != nil {
		return e
	}
	quotaargs := []string{"/usr/bin/env", "-i", "HOME=" + home, "PATH=/usr/bin:/bin:/usr/sbin:/sbin", agent, "quota-run", "--config", cfg}
	if e = write(serviceFile(home, "quota"), plist(home, root, "quota", quotaargs), 0600); e != nil {
		return e
	}
	if e = writeAnalyticsService(home); e != nil {
		return e
	}
	// Prepare every local artifact before consuming the one-use invitation.
	// Once enrollment saves the config, Resume can complete service startup.
	progress("Connecting to your workspace…")
	if e = command(ctx, agent, "enroll", "--server", v.Server, "--config", cfg, "--invitation-file", privateInvite, "--device", hostname(), "--ack-version", fmt.Sprint(ack), "--upstream", fmt.Sprintf("http://127.0.0.1:%d", port), "--upstream-token-file", tokenfile); e != nil {
		return errors.New("workspace could not accept this invitation; it may be expired or already used. Download a new connection file")
	}
	progress("Starting background sync…")
	return SetServices(ctx, home, true)
}
func hostname() string {
	h, e := os.Hostname()
	if e != nil || h == "" {
		return "Mac"
	}
	return h
}

const disabledAgents = `disabled_agents = ["aider", "amp", "antigravity-cli", "augure-code", "augure-desktop", "cline", "codebuddy", "codebuff", "commandcode", "copilot", "cortex", "cowork", "crush", "cursor", "cursor-ide", "deepseek-harness", "deepseek-tui", "devin", "evener", "forge", "gemini", "goose", "gptme", "grok", "hermes", "icodemate", "iflow", "kilo", "kilo-legacy", "kimi", "kimi-work", "kiro", "kiro-ide", "mimocode", "omnigent", "omp", "openclaude", "openclaw", "opencode", "opencodereview", "openhands", "pi", "piebald", "poolside", "posit-assistant", "positron", "prime-agent", "qclaw", "qoder", "qwen", "qwenpaw", "reasonix", "roocode", "shelley", "tau", "trae", "traex", "vibe", "visualstudio-copilot", "vscode-copilot", "warp", "windsurf", "workbuddy", "zcode", "zed", "zencoder"]
`
