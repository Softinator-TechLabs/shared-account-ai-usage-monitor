package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/macsetup"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Choose install, upgrade, status, pause, resume or uninstall")
		os.Exit(1)
	}
	f := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	file := f.String("invitation-file", "", "private connection file")
	server := f.String("server", "", "fallback HTTPS workspace for legacy JSON")
	resources := f.String("resources", "", "app resources")
	ack := f.Int("ack-version", 0, "acknowledged policy")
	jsonOutput := f.Bool("json", false, "machine-readable status")
	f.Parse(os.Args[2:])
	home, e := os.UserHomeDir()
	if e != nil {
		panic(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	switch os.Args[1] {
	case "status":
		snapshot := macsetup.ReadStatus(ctx, home)
		if *jsonOutput {
			if err := json.NewEncoder(os.Stdout).Encode(snapshot); err != nil {
				os.Exit(1)
			}
		} else {
			fmt.Printf("Background services: %s\n", snapshot.State)
			for _, service := range snapshot.Services {
				fmt.Printf("%s: %s\n", service.Name, service.State)
			}
			fmt.Printf("Analytics sync: %s; last success: %s\nQuota sync: %s; last success: %s\n", snapshot.Analytics.State, snapshot.Analytics.LastSuccessAt, snapshot.Quota.State, snapshot.Quota.LastSuccessAt)
		}
		return
	case "install":
		e = macsetup.Install(ctx, home, *resources, *file, *server, *ack, func(s string) { fmt.Println(s) })
	case "pause", "resume":
		e = macsetup.SetServices(ctx, home, os.Args[1] == "resume")
	case "upgrade":
		e = macsetup.Upgrade(ctx, home, *resources)
	case "uninstall":
		e = macsetup.Uninstall(ctx, home)
	default:
		e = fmt.Errorf("unknown setup action")
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("Done")
}
