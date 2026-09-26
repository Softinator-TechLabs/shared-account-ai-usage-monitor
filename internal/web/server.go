// Package web exposes one permission boundary for the browser and read-only agent clients.
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/store"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed assets/*
var assets embed.FS

type Config struct {
	Origin, Workspace, Issuer, ClientID, ClientSecret, AuthMode string
	Demo                                                        bool
}
type Server struct {
	store   *store.Store
	config  Config
	mux     *http.ServeMux
	auth    *oidcFlow
	limiter authLimiter
}

func New(ctx context.Context, s *store.Store, cfg Config) (*Server, error) {
	u, e := url.Parse(cfg.Origin)
	if e != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.User != nil || cfg.Workspace == "" {
		return nil, errors.New("valid origin and workspace required")
	}
	if cfg.Demo {
		if u.Scheme != "http" || !net.ParseIP(u.Hostname()).IsLoopback() {
			return nil, errors.New("synthetic demo requires literal loopback HTTP origin")
		}
	} else if u.Scheme != "https" {
		return nil, errors.New("production origin must use HTTPS")
	}
	app := &Server{store: s, config: cfg, mux: http.NewServeMux()}
	if cfg.AuthMode != "" && cfg.AuthMode != "password" && cfg.AuthMode != "oidc" && cfg.AuthMode != "both" {
		return nil, errors.New("invalid auth mode")
	}
	if !cfg.Demo && ((cfg.AuthMode == "" && cfg.Issuer != "") || cfg.AuthMode == "oidc" || cfg.AuthMode == "both") {
		app.auth, e = newOIDC(ctx, cfg)
		if e != nil {
			return nil, e
		}
	}
	app.routes()
	return app, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	s.mux.ServeHTTP(w, r)
}
func token(r *http.Request) string {
	if v := r.Header.Get("Authorization"); v != "" {
		if strings.HasPrefix(v, "Bearer ") {
			return strings.TrimPrefix(v, "Bearer ")
		}
		return ""
	}
	if k, e := r.Cookie("team_session"); e == nil {
		return k.Value
	}
	return ""
}
func (s *Server) principal(w http.ResponseWriter, r *http.Request, device bool) (c.Principal, bool) {
	p, e := s.store.Authenticate(r.Context(), token(r))
	if e != nil {
		fail(w, c.ErrForbidden)
		return p, false
	}
	if (p.Device != "") != device || ((p.TokenKind == "read" || p.TokenKind == "debug") && r.Method != "GET" && r.Method != "HEAD" && r.URL.Path != "/auth/logout") {
		fail(w, c.ErrForbidden)
		return p, false
	}
	if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("Authorization") == "" && r.Header.Get("Origin") != s.config.Origin {
		fail(w, c.ErrForbidden)
		return p, false
	}
	return p, true
}
func read(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
	body, e := io.ReadAll(r.Body)
	if e != nil || !utf8.Valid(body) {
		fail(w, c.ErrInvalid)
		return false
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		fail(w, c.ErrInvalid)
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, c.ErrInvalid)
		return false
	}
	return true
}
func write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, e error) {
	code := 500
	switch {
	case errors.Is(e, c.ErrDeleted):
		code = 410
	case errors.Is(e, c.ErrForbidden):
		code = 403
	case errors.Is(e, c.ErrNotFound):
		code = 404
	case errors.Is(e, c.ErrConflict):
		code = 409
	case errors.Is(e, c.ErrInvalid):
		code = 400
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	label := http.StatusText(code)
	if errors.Is(e, c.ErrDeleted) {
		label = "source_deleted"
	}
	json.NewEncoder(w).Encode(map[string]string{"error": label})
}
func (s *Server) human(pattern string, f func(http.ResponseWriter, *http.Request, c.Principal)) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		p, ok := s.principal(w, r, false)
		if ok {
			f(w, r, p)
		}
	})
}
func (s *Server) cookie(w http.ResponseWriter, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: "team_session", Value: value, Path: "/", HttpOnly: true, Secure: !s.config.Demo, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (s *Server) routes() {
	s.quotaRoutes()
	s.passwordRoutes()
	s.debugRoutes()
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if e := s.store.DB.Ping(r.Context()); e != nil {
			w.WriteHeader(503)
			return
		}
		write(w, map[string]any{"status": "ok", "schema": 1, "demo": s.config.Demo})
	})
	s.mux.HandleFunc("GET /auth/login", s.login)
	s.mux.HandleFunc("GET /auth/callback", s.callback)
	s.mux.HandleFunc("POST /auth/demo", func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Demo || r.Header.Get("Origin") != s.config.Origin {
			fail(w, c.ErrForbidden)
			return
		}
		var v struct {
			Person string `json:"person"`
		}
		if !read(w, r, &v) {
			return
		}
		if v.Person != "owner" && v.Person != "alice" && v.Person != "bob" {
			fail(w, c.ErrForbidden)
			return
		}
		t, e := s.store.IssueHumanToken(r.Context(), s.config.Workspace, v.Person)
		if e != nil {
			fail(w, e)
			return
		}
		s.cookie(w, t, 28800)
		write(w, map[string]bool{"ok": true})
	})
	s.human("POST /auth/logout", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		if e := s.store.Logout(r.Context(), token(r)); e != nil {
			fail(w, e)
			return
		}
		s.cookie(w, "", -1)
		write(w, map[string]bool{"ok": true})
	})
	s.human("GET /api/v1/dashboard", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		days := 14
		if raw := r.URL.Query().Get("days"); raw != "" {
			var e error
			days, e = strconv.Atoi(raw)
			if e != nil {
				fail(w, c.ErrInvalid)
				return
			}
		}
		d, e := s.store.Dashboard(r.Context(), p, days, time.Now())
		if e != nil {
			fail(w, e)
			return
		}
		write(w, d)
	})
	s.human("GET /api/v1/me", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		policy, e := s.store.Policy(r.Context(), p.Workspace)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]any{"principal": p, "policy": policy, "demo": s.config.Demo})
	})
	s.human("POST /api/v1/access-token", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		v, e := s.store.ReadToken(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]any{"token": v, "expires_in": 28800})
	})
	s.human("GET /api/v1/activity", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		q := r.URL.Query()
		f := c.Filter{SourceRef: q.Get("source"), Query: q.Get("q"), Person: q.Get("person"), Project: q.Get("project"), Client: q.Get("client")}
		if q.Get("before") != "" {
			var e error
			f.Before, e = time.Parse(time.RFC3339Nano, q.Get("before"))
			if e != nil {
				fail(w, c.ErrInvalid)
				return
			}
		}
		rows, e := s.store.List(r.Context(), p, f)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, rows)
	})
	for _, route := range []string{"GET /api/v1/sessions/{id}", "GET /api/v1/sessions/{id}/export"} {
		s.human(route, func(w http.ResponseWriter, r *http.Request, p c.Principal) {
			row, e := s.store.Get(r.Context(), p, r.PathValue("id"))
			if e != nil {
				fail(w, e)
				return
			}
			write(w, row)
		})
	}
	s.human("GET /api/v1/sessions/{id}/reviews", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		rows, e := s.store.Reviews(r.Context(), p, r.PathValue("id"))
		if e != nil {
			fail(w, e)
			return
		}
		write(w, rows)
	})
	s.human("POST /api/v1/sessions/{id}/reviews", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v store.Review
		if !read(w, r, &v) {
			return
		}
		v.SessionID = r.PathValue("id")
		out, e := s.store.AddReview(r.Context(), p, v)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, out)
	})
	s.human("POST /api/v1/policy", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v c.Policy
		if !read(w, r, &v) {
			return
		}
		if e := s.store.SetPolicy(r.Context(), p, v); e != nil {
			fail(w, e)
			return
		}
		write(w, v)
	})
	s.human("GET /api/v1/members", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		v, e := s.store.Members(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, v)
	})
	s.human("POST /api/v1/members", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v struct {
			Person  string `json:"person"`
			Role    string `json:"role"`
			Subject string `json:"subject"`
		}
		if !read(w, r, &v) {
			return
		}
		if e := s.store.SetMember(r.Context(), p, v.Person, v.Role, v.Subject); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.human("DELETE /api/v1/members/{person}", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		if e := s.store.Revoke(r.Context(), p, r.PathValue("person")); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.human("POST /api/v1/invitations", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v struct {
			Person string `json:"person"`
		}
		if !read(w, r, &v) {
			return
		}
		inv, e := s.store.Invite(r.Context(), p, v.Person, "device", time.Now().Add(24*time.Hour))
		if e != nil {
			fail(w, e)
			return
		}
		policy, e := s.store.Policy(r.Context(), p.Workspace)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]any{"invitation": inv, "policy": policy, "server": s.config.Origin, "person": v.Person})
	})
	s.mux.HandleFunc("POST /api/v1/enroll", func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			Invitation    string `json:"invitation"`
			Device        string `json:"device"`
			PolicyVersion int    `json:"policy_version"`
		}
		if !read(w, r, &v) {
			return
		}
		t, e := s.store.Enroll(r.Context(), v.Invitation, v.Device, v.PolicyVersion)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]string{"token": t})
	})
	s.mux.HandleFunc("GET /api/v1/device/policy", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.store.DevicePolicy(r.Context(), token(r))
		if e != nil {
			fail(w, e)
			return
		}
		write(w, v)
	})
	s.mux.HandleFunc("POST /api/v1/device/ack", func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			Version int `json:"version"`
		}
		if !read(w, r, &v) {
			return
		}
		if e := s.store.Acknowledge(r.Context(), token(r), v.Version); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.mux.HandleFunc("POST /api/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		p, ok := s.principal(w, r, true)
		if !ok {
			return
		}
		var v c.Snapshot
		if !read(w, r, &v) {
			return
		}
		row, e := s.store.Accept(r.Context(), p, v)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]string{"id": row.ID, "revision": row.Revision})
	})
	s.human("DELETE /api/v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		v, e := s.store.Get(r.Context(), p, r.PathValue("id"))
		if e == nil {
			e = s.store.DeleteSource(r.Context(), p, v.SourceRef)
		}
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.human("GET /api/v1/accounts", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		v, e := s.store.Accounts(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, v)
	})
	s.human("POST /api/v1/accounts", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v store.Account
		if !read(w, r, &v) {
			return
		}
		if e := s.store.PutAccount(r.Context(), p, v); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.human("POST /api/v1/quotas", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v store.Quota
		if !read(w, r, &v) {
			return
		}
		if e := s.store.AddQuota(r.Context(), p, v); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.human("POST /api/v1/sessions/{id}/analysis", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v struct {
			Ordinal int `json:"ordinal"`
		}
		if !read(w, r, &v) {
			return
		}
		out, e := s.store.BeginAnalysis(r.Context(), p, r.PathValue("id"), v.Ordinal, "external-native")
		if e != nil {
			fail(w, e)
			return
		}
		write(w, out)
	})
	s.mux.HandleFunc("POST /api/v1/analysis/result", func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			Body string `json:"body"`
		}
		if !read(w, r, &v) {
			return
		}
		if e := s.store.FinishAnalysis(r.Context(), token(r), v.Body); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.mux.HandleFunc("POST /api/v1/ingest/chunk", func(w http.ResponseWriter, r *http.Request) {
		p, ok := s.principal(w, r, true)
		if !ok {
			return
		}
		var v struct {
			Hash  string `json:"hash"`
			Index int    `json:"index"`
			Total int    `json:"total"`
			Body  []byte `json:"body"`
		}
		if !read(w, r, &v) {
			return
		}
		if e := s.store.SaveChunk(r.Context(), p, v.Hash, v.Index, v.Total, v.Body); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.mux.HandleFunc("POST /api/v1/ingest/commit", func(w http.ResponseWriter, r *http.Request) {
		p, ok := s.principal(w, r, true)
		if !ok {
			return
		}
		var v struct {
			Hash string `json:"hash"`
		}
		if !read(w, r, &v) {
			return
		}
		row, e := s.store.CommitChunks(r.Context(), p, v.Hash)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]string{"id": row.ID, "revision": row.Revision})
	})
	s.human("GET /api/v1/deletion-ledger", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		v, e := s.store.DeletionLedger(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, v)
	})
	sub, _ := fs.Sub(assets, "assets")
	s.mux.Handle("GET /", http.FileServer(http.FS(sub)))
}
