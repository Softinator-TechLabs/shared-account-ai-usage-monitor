// team-server serves the self-hosted archive; demo data is strictly loopback only.
package main

import (
	"context"
	"encoding/json"
	"flag"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/store"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/web"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	addr := flag.String("listen", "127.0.0.1:8090", "listen address")
	demo := flag.Bool("demo", false, "synthetic loopback-only login/data")
	flag.Parse()
	if *demo {
		host, _, e := net.SplitHostPort(*addr)
		if e != nil || !net.ParseIP(host).IsLoopback() {
			return c.ErrForbidden
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, e := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		return e
	}
	defer db.DB.Close()
	workspace := os.Getenv("TEAM_WORKSPACE")
	if workspace == "" {
		workspace = "team"
		if *demo {
			workspace = "synthetic-demo"
		}
	}
	if *demo && !strings.HasPrefix(workspace, "synthetic-") {
		return c.ErrInvalid
	}
	origin := os.Getenv("PUBLIC_ORIGIN")
	if *demo && origin == "" {
		origin = "http://" + *addr
	}
	owner := c.Principal{Workspace: workspace, Person: "owner", Role: "owner"}
	// Bootstrap is explicit and idempotent; existing policy/membership never overwritten.
	if _, e = db.Policy(ctx, workspace); e == c.ErrNotFound {
		var p c.Policy
		if *demo {
			p = c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team", RetentionDays: 90}
		} else if e = json.Unmarshal([]byte(os.Getenv("INITIAL_POLICY")), &p); e != nil {
			return e
		}
		if e = db.SetPolicy(ctx, owner, p); e != nil {
			return e
		}
	}
	var count int
	if e = db.DB.QueryRow(ctx, "SELECT count(*) FROM tm_members WHERE workspace=$1", workspace).Scan(&count); e != nil {
		return e
	}
	if !*demo && os.Getenv("OWNER_EMAIL") != "" {
		file := os.Getenv("OWNER_SETUP_FILE")
		if file == "" {
			return c.ErrInvalid
		}
		tok, err := db.PrepareOwner(ctx, workspace, os.Getenv("OWNER_EMAIL"), os.Getenv("OWNER_SETUP_TOKEN"))
		if err != nil {
			return err
		}
		if tok != "" {
			f, err := os.CreateTemp(filepath.Dir(file), ".owner-setup-")
			if err != nil {
				return err
			}
			defer os.Remove(f.Name())
			if err = f.Chmod(0600); err == nil {
				_, err = f.WriteString(origin + "/#setup/" + tok + "\n")
			}
			if err == nil {
				err = f.Sync()
			}
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
			if err = os.Rename(f.Name(), file); err != nil {
				return err
			}
			log.Print("Owner setup link saved to configured private file; expires in 24 hours")
		} else {
			_ = os.Remove(file)
		}
	} else if count == 0 {
		subject := os.Getenv("OWNER_OIDC_SUBJECT")
		if !*demo && subject == "" {
			return c.ErrInvalid
		}
		if e = db.SetMember(ctx, owner, "owner", "owner", subject); e != nil {
			return e
		}
	}

	if *demo {
		for _, name := range []string{"alice", "bob"} {
			if e = db.SetMember(ctx, owner, name, "member", ""); e != nil {
				return e
			}
		}
		if e = seed(ctx, db, workspace); e != nil {
			return e
		}
	}
	if file := os.Getenv("RESTORE_DELETION_LEDGER"); file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		var refs []string
		if err = json.Unmarshal(b, &refs); err != nil {
			return err
		}
		if err = db.RestoreDeletions(ctx, owner, refs); err != nil {
			return err
		}
	}
	if e = db.Expire(ctx); e != nil {
		return e
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := db.Expire(ctx); err != nil {
					log.Print("retention maintenance failed; inspect database health")
				}
			}
		}
	}()
	app, e := web.New(ctx, db, web.Config{Origin: origin, Workspace: workspace, Demo: *demo, AuthMode: os.Getenv("AUTH_MODE"), Issuer: os.Getenv("OIDC_ISSUER"), ClientID: os.Getenv("OIDC_CLIENT_ID"), ClientSecret: os.Getenv("OIDC_CLIENT_SECRET")})
	if e != nil {
		return e
	}
	server := &http.Server{Addr: *addr, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 90 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	go func() {
		<-ctx.Done()
		shutdown, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		server.Shutdown(shutdown)
	}()
	log.Printf("team archive listening on %s (synthetic=%t)", *addr, *demo)
	e = server.ListenAndServe()
	if e == http.ErrServerClosed {
		return nil
	}
	return e
}
func seed(ctx context.Context, s *store.Store, workspace string) error {
	p, e := s.Policy(ctx, workspace)
	if e != nil {
		return e
	}
	examples := []struct{ person, project, prompt, reply string }{{"alice", "sample-checkout", "Checkout ki duplicate charge bug fix karo. Pehle failing test reproduce karo; idempotency key same ho to payment sirf ek baar hona chahiye. Retry aur timeout cases bhi cover karna.", "Synthetic response: identified the retry boundary and proposed an idempotency test."}, {"bob", "sample-publishing", "Is page ko better bana do.", "Synthetic response: which reader task and acceptance criteria should improve?"}, {"alice", "sample-publishing", "PDF export mein Hindi headings toot rahi hain. Font fallback ko isolate karo, original manuscript mat badalna. Before/after fixture aur visual proof chahiye.", "Synthetic response: prepared a font fallback fixture. No real work or PR is implied."}}
	for i, x := range examples {
		v := c.Snapshot{SchemaVersion: 1, SourceRef: "synthetic:" + x.person + ":" + x.project, Revision: "demo-v1", PolicyVersion: p.Version, Client: []string{"codex", "claude", "codex"}[i], Project: x.project, Branch: "synthetic/example", Account: "shared-demo", AccountMethod: "synthetic_declaration", StartedAt: "2026-09-26T09:30:00Z", Coverage: "synthetic example; no employee activity", Messages: []c.Message{{Ordinal: 0, Role: "user", Content: x.prompt, Model: "synthetic-model", Timestamp: "2026-09-26T09:30:00Z"}, {Ordinal: 1, Role: "assistant", Content: x.reply, Model: "synthetic-model", Timestamp: "2026-09-26T09:30:02Z"}}}
		_, e = s.Accept(ctx, c.Principal{Workspace: workspace, Person: x.person, Role: "member", Device: "synthetic-device", EnrolledAt: time.Now()}, v)
		if e != nil && e != c.ErrForbidden {
			return e
		}
	}
	return nil
}
