// team-agent is a foreground service suitable for launchd, systemd or Task Scheduler.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/agentsview"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/companion"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/policy"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/spool"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type config struct {
	CodexProfiles     []companion.CodexProfile `json:"codex_profiles,omitempty"`
	Server            string                   `json:"server"`
	Token             string                   `json:"token"`
	Upstream          string                   `json:"upstream"`
	UpstreamTokenFile string                   `json:"upstream_token_file"`
	Policy            c.Policy                 `json:"policy"`
	Account           string                   `json:"declared_account,omitempty"`
}

func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: team-agent enroll|run|once|analytics-run|analytics-once|quota-run|quota-once|status|policy|ack|discard-queue|discard-quota-queue --config FILE")
	}
	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	path := flags.String("config", "team-agent.json", "private configuration path")
	server := flags.String("server", "", "central HTTPS origin")
	invitation := flags.String("invitation-file", "", "single-use invitation JSON file from owner")
	device := flags.String("device", "", "unique device label")
	upstream := flags.String("upstream", "http://127.0.0.1:8080", "AgentsView loopback origin")
	upstreamToken := flags.String("upstream-token-file", "", "AgentsView token file")
	version := flags.Int("ack-version", 0, "explicitly acknowledge policy version")
	confirm := flags.Bool("confirm", false, "confirm irreversible queue discard")
	if e := flags.Parse(os.Args[2:]); e != nil {
		return e
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg := config{}
	if command == "enroll" {
		if _, e := os.Stat(*path); !os.IsNotExist(e) {
			return errors.New("config already exists or cannot be inspected")
		}
		b, e := os.ReadFile(*invitation)
		if e != nil {
			return e
		}
		var invite struct {
			Invitation string   `json:"invitation"`
			Policy     c.Policy `json:"policy"`
		}
		if e = json.Unmarshal(b, &invite); e != nil {
			return e
		}
		if *version < 1 || *version != invite.Policy.Version {
			return errors.New("read invitation policy and supply its --ack-version")
		}
		cl, e := companion.New(*server, "")
		if e != nil {
			return e
		}
		var result struct {
			Token string `json:"token"`
		}
		e = cl.Request(ctx, "POST", "/api/v1/enroll", map[string]any{"invitation": invite.Invitation, "device": *device, "policy_version": *version}, &result)
		if e != nil {
			return e
		}
		cfg = config{Server: *server, Token: result.Token, Upstream: *upstream, UpstreamTokenFile: *upstreamToken, Policy: invite.Policy}
		return save(*path, cfg)
	}
	b, e := os.ReadFile(*path)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, &cfg); e != nil {
		return e
	}
	if cfg.Account != "" {
		return errors.New("declared_account cannot identify historical prompts; remove it and manage assignments centrally")
	}
	if command == "ack" || command == "discard-queue" || command == "discard-quota-queue" {
		for _, suffix := range []string{".lock", ".quota.lock", ".analytics.lock"} {
			lock := *path + suffix
			if e = os.Mkdir(lock, 0700); e != nil {
				return errors.New("pause all collectors before changing policy or discarding queues")
			}
			defer os.Remove(lock)
		}
	}
	cl, e := companion.New(cfg.Server, cfg.Token)
	if e != nil {
		return e
	}
	q := &spool.Queue{Dir: *path + ".queue", MaxBytes: 1 << 30}
	if command == "analytics-run" || command == "analytics-once" {
		var secret string
		if cfg.UpstreamTokenFile != "" {
			body, err := os.ReadFile(cfg.UpstreamTokenFile)
			if err != nil {
				return err
			}
			secret = strings.TrimSpace(string(body))
		}
		av, err := agentsview.New(cfg.Upstream, secret)
		if err != nil {
			return err
		}
		lock := *path + ".analytics.lock"
		if err = os.Mkdir(lock, 0700); err != nil {
			return errors.New("analytics collector already running or stale lock")
		}
		defer os.Remove(lock)
		status := companion.SyncStatusWriter{Path: *path + ".analytics-status.json"}
		for {
			reportStatusWrite(status.Begin())
			stats, err := cl.UsageCycleWithProgress(ctx, av, *path+".analytics.checkpoint.json", cfg.Policy.Version, func() {
				reportStatusWrite(status.Uploaded())
			})
			reportStatusWrite(status.Finish(err))
			log.Printf("Analytics sources: scanned=%d uploaded=%d unchanged=%d suppressed=%d", stats.Scanned, stats.Uploaded, stats.Skipped, stats.Suppressed)
			if command == "analytics-once" {
				return err
			}
			if err != nil {
				log.Print(err)
			}
			timer := time.NewTimer(5 * time.Minute)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
		}
	}
	if command == "quota-run" || command == "quota-once" {
		if len(cfg.CodexProfiles) == 0 {
			cfg.CodexProfiles = companion.DefaultCodexProfile()
		}
		if len(cfg.CodexProfiles) > 16 {
			return errors.New("configure 1-16 explicit codex_profiles first")
		}
		lock := *path + ".quota.lock"
		if err := os.Mkdir(lock, 0700); err != nil {
			return errors.New("quota collector already running or stale lock")
		}
		defer os.Remove(lock)
		queue := &spool.Queue{Dir: *path + ".quota.queue", MaxBytes: 16 << 20}
		status := companion.SyncStatusWriter{Path: *path + ".quota-status.json"}
		for {
			if len(cfg.CodexProfiles) == 0 {
				cfg.CodexProfiles = companion.DefaultCodexProfile()
			}
			reportStatusWrite(status.Begin())
			err := cl.QuotaCycle(ctx, queue, cfg.Policy.Version, cfg.CodexProfiles)
			reportStatusWrite(status.Finish(err))
			if command == "quota-once" {
				return err
			}
			if err != nil {
				log.Print(err)
			}
			timer := time.NewTimer(60 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
		}
	}

	switch command {
	case "status":
		rows, e := q.Pending()
		if e != nil {
			return e
		}
		quotaRows, err := (&spool.Queue{Dir: *path + ".quota.queue"}).Pending()
		if err != nil {
			return err
		}
		usage, suppressed, err := companion.UsageCheckpointCounts(*path + ".analytics.checkpoint.json")
		if err != nil {
			return err
		}
		fmt.Printf("Queued revisions: %d; quota observations: %d; analytics acknowledged sources: %d; analytics suppressed sources: %d; acknowledged policy: %d\n", len(rows), len(quotaRows), usage, suppressed, cfg.Policy.Version)
		return nil
	case "policy":
		p, e := cl.Policy(ctx)
		if e == nil {
			b, _ := json.MarshalIndent(p, "", "  ")
			fmt.Println(string(b))
		}
		return e
	case "discard-queue", "discard-quota-queue":
		if command == "discard-quota-queue" {
			q = &spool.Queue{Dir: *path + ".quota.queue"}
		}
		if !*confirm {
			return errors.New("--confirm required; discarded quota observations cannot be recreated; transcript snapshots can be recollected")
		}
		rows, e := q.Pending()
		if e != nil {
			return e
		}
		for _, r := range rows {
			if e = q.Ack(r.Key); e != nil {
				return e
			}
		}
		if command == "discard-quota-queue" {
			return nil
		}
		if err := os.Remove(*path + ".checkpoint.json"); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case "ack":
		p, e := cl.Policy(ctx)
		if e != nil {
			return e
		}
		if p.Version != *version {
			return errors.New("read current policy and supply exact --ack-version")
		}
		rows, e := q.Pending()
		if e != nil {
			return e
		}
		quotaRows, err := (&spool.Queue{Dir: *path + ".quota.queue"}).Pending()
		if err != nil {
			return err
		}
		if len(rows) > 0 || len(quotaRows) > 0 {
			return errors.New("deliver or explicitly discard both pending queues before acknowledging new policy")
		}
		if e = cl.Request(ctx, "POST", "/api/v1/device/ack", map[string]int{"version": *version}, nil); e != nil {
			return e
		}
		cfg.Policy = p
		return save(*path, cfg)
	case "run", "once":
	default:
		return errors.New("unknown command")
	}
	var upstreamSecret string
	if cfg.UpstreamTokenFile != "" {
		b, e = os.ReadFile(cfg.UpstreamTokenFile)
		if e != nil {
			return e
		}
		upstreamSecret = strings.TrimSpace(string(b))
	}
	av, e := agentsview.New(cfg.Upstream, upstreamSecret)
	if e != nil {
		return e
	}
	// An atomic directory prevents two writers using the same config/spool. A crashed process
	// leaves the lock; the operator removes it only after confirming no process is alive.
	lock := *path + ".lock"
	if e = os.Mkdir(lock, 0700); e != nil {
		return errors.New("collector lock exists; check another process or stale lock")
	}
	defer os.Remove(lock)
	// Drain durable records before touching upstream. A broken upstream must not strand them.
	checkpoint := map[string]string{}
	checkpointPath := *path + ".checkpoint.json"
	if data, err := os.ReadFile(checkpointPath); err == nil {
		if json.Unmarshal(data, &checkpoint) != nil {
			return errors.New("invalid collection checkpoint")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	cycle := func() error {
		online := true
		if err := cl.Deliver(ctx, q, cfg.Policy.Version); err != nil {
			if !companion.Offline(err) {
				return err
			}
			online = false
		}
		// Only a genuine outage permits cached-policy capture. Explicit denial stops reads.
		if current, err := cl.Policy(ctx); err != nil {
			if !companion.Offline(err) {
				return err
			}
			online = false
		} else if current.Version != cfg.Policy.Version {
			return errors.New("policy changed; collection paused")
		}
		// A first history import can take hours. Report the live device before reading it.
		if online {
			if err := cl.Request(ctx, "POST", "/api/v1/device/heartbeat", map[string]string{"os": runtime.GOOS}, nil); err != nil {
				if !companion.Offline(err) {
					return err
				}
				online = false
			}
		}
		pendingMeta := map[string]string{}
		err := av.CollectEach(ctx, cfg.Policy.Version, func(id, hash string) bool {
			pendingMeta[id] = hash
			return q.Suppressed(id) || checkpoint[id] == fmt.Sprintf("%d:%s", cfg.Policy.Version, hash)
		}, func(v c.Snapshot) error {
			_, body, err := policy.Apply(v, cfg.Policy)
			if err != nil {
				return err
			}
			if _, err = q.Append(body); err != nil {
				return err
			}
			checkpoint[v.SourceRef] = fmt.Sprintf("%d:%s", cfg.Policy.Version, pendingMeta[v.SourceRef])
			if err = saveCheckpoint(checkpointPath, checkpoint); err != nil {
				return err
			}
			if online {
				if err = cl.Deliver(ctx, q, cfg.Policy.Version); err != nil {
					if !companion.Offline(err) {
						return err
					}
					online = false
				} else if err = cl.Request(ctx, "POST", "/api/v1/device/heartbeat", map[string]string{"os": runtime.GOOS}, nil); err != nil {
					if !companion.Offline(err) {
						return err
					}
					online = false
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if err = cl.Deliver(ctx, q, cfg.Policy.Version); err != nil {
			return err
		}
		return cl.Request(ctx, "POST", "/api/v1/device/heartbeat", map[string]string{"os": runtime.GOOS}, nil)
	}
	initialErr := cycle()
	if err := initialErr; command == "once" {
		return err
	} else if err != nil {
		log.Print(err)
	}
	delivery := time.NewTicker(15 * time.Second)
	defer delivery.Stop()
	reconcile := time.NewTicker(15 * time.Minute)
	if initialErr != nil {
		reconcile.Reset(time.Minute)
	}
	defer reconcile.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-delivery.C:
			if e = cl.Deliver(ctx, q, cfg.Policy.Version); e != nil {
				log.Print(e)
			} else if e = cl.Request(ctx, "POST", "/api/v1/device/heartbeat", map[string]string{"os": runtime.GOOS}, nil); e != nil {
				log.Print(e)
			}
		case <-reconcile.C:
			if e = cycle(); e != nil {
				log.Print(e)
				reconcile.Reset(time.Minute)
			} else {
				reconcile.Reset(15 * time.Minute)
			}
		}
	}
}
func save(path string, v config) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".agent-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
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

func saveCheckpoint(path string, value map[string]string) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".checkpoint-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	closed := f.Close()
	if e != nil {
		return e
	}
	if closed != nil {
		return closed
	}
	return os.Rename(f.Name(), path)
}

// Do not leak config paths or source metadata through a local status failure.
func reportStatusWrite(err error) {
	if err != nil {
		log.Print("collector status could not be saved")
	}
}
