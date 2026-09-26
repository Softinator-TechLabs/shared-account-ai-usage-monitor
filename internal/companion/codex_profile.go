package companion

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DefaultCodexProfile locates an installed executable, never opens credential files.
func DefaultCodexProfile() []CodexProfile {
	home, e := os.UserHomeDir()
	if e != nil {
		return nil
	}
	profile := os.Getenv("CODEX_HOME")
	if profile == "" {
		profile = filepath.Join(home, ".codex")
	}
	profile, e = filepath.Abs(profile)
	if e != nil {
		return nil
	}
	candidates := []string{}
	if path, e := exec.LookPath("codex"); e == nil {
		candidates = append(candidates, path)
	}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/Applications/ChatGPT.app/Contents/Resources/codex-cli/CodexCLI.app/Contents/MacOS/codex", "/Applications/Codex.app/Contents/Resources/codex", "/opt/homebrew/bin/codex", "/usr/local/bin/codex")
	}
	for _, path := range candidates {
		if st, e := os.Stat(path); e == nil && !st.IsDir() {
			path, e = filepath.Abs(path)
			if e == nil {
				return []CodexProfile{{Label: "codex-default", Executable: path, Home: profile}}
			}
		}
	}
	return nil
}
