package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/macsetup"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Choose install, pause or resume")
		os.Exit(1)
	}
	f := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	file := f.String("invitation-file", "", "private connection file")
	server := f.String("server", "", "fallback HTTPS workspace for legacy JSON")
	resources := f.String("resources", "", "app resources")
	ack := f.Int("ack-version", 0, "acknowledged policy")
	f.Parse(os.Args[2:])
	home, e := os.UserHomeDir()
	if e != nil {
		panic(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	switch os.Args[1] {
	case "install":
		e = macsetup.Install(ctx, home, *resources, *file, *server, *ack, func(s string) { fmt.Println(s) })
	case "pause", "resume":
		for _, name := range []string{"quota", "companion", "agentsview"} {
			if e = macsetup.Service(ctx, home, name, os.Args[1] == "resume"); e != nil {
				break
			}
		}
	default:
		e = fmt.Errorf("unknown setup action")
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("Done")
}
