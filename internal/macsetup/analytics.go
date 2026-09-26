package macsetup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

type serviceFunc func(context.Context, string, string, bool) error

func writeAnalyticsService(home string) error {
	root := Root(home)
	for _, suffix := range []string{".stdout.log", ".stderr.log"} {
		path := filepath.Join(root, "analytics"+suffix)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err = write(path, nil, 0600); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := os.Chmod(path, 0600); err != nil {
			return err
		}
	}
	args := []string{"/usr/bin/env", "-i", "HOME=" + home, "PATH=/usr/bin:/bin:/usr/sbin:/sbin", filepath.Join(root, "bin", "team-agent"), "analytics-run", "--config", filepath.Join(root, "team-agent.json")}
	return write(serviceFile(home, "analytics"), plist(home, root, "analytics", args), 0600)
}

func serviceOrder(start bool) []string {
	if start {
		return []string{"agentsview", "analytics", "quota", "companion"}
	}
	return []string{"analytics", "quota", "companion", "agentsview"}
}

// SetServices includes every independently running collector in pause/resume.
func SetServices(ctx context.Context, home string, start bool) error {
	for _, name := range serviceOrder(start) {
		if err := Service(ctx, home, name, start); err != nil {
			return err
		}
	}
	return nil
}

// Upgrade replaces only the companion binary and prepares its new analytics
// service. Existing enrollment, logs, queues, checkpoints and AgentsView persist.
func Upgrade(ctx context.Context, home, resources string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("this upgrade is for macOS")
	}
	return upgrade(ctx, home, resources, Service)
}

func upgrade(ctx context.Context, home, resources string, service serviceFunc) error {
	root := Root(home)
	if resources == "" {
		return errors.New("provide the app Resources directory with --resources")
	}
	info, err := os.Stat(filepath.Join(root, "team-agent.json"))
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("this Mac is not enrolled; connect it before upgrading")
	}
	binary, err := os.ReadFile(filepath.Join(resources, "team-agent"))
	if err != nil || len(binary) == 0 {
		return errors.New("app bundle is incomplete; existing collectors were not changed")
	}
	// Inspect every required artifact before stopping the running collectors.
	for _, name := range []string{"analytics", "quota", "companion"} {
		if err = service(ctx, home, name, false); err != nil {
			return err
		}
	}
	if err = write(filepath.Join(root, "bin", "team-agent"), binary, 0700); err != nil {
		return err
	}
	if err = writeAnalyticsService(home); err != nil {
		return err
	}
	for _, name := range serviceOrder(true) {
		if err = service(ctx, home, name, true); err != nil {
			return err
		}
	}
	return nil
}

// Uninstall stops and removes this application's LaunchAgents. Private data and
// binaries remain available for recovery; central revocation/deletion is separate.
func Uninstall(ctx context.Context, home string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("this uninstall is for macOS")
	}
	return uninstall(ctx, home, Service)
}

func uninstall(ctx context.Context, home string, service serviceFunc) error {
	for _, name := range serviceOrder(false) {
		if err := service(ctx, home, name, false); err != nil {
			return err
		}
	}
	for _, name := range serviceOrder(false) {
		if err := os.Remove(serviceFile(home, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
