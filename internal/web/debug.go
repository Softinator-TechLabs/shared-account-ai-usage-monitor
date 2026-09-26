package web

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/http"
)

func (s *Server) debugRoutes() {
	s.human("POST /api/v1/debug-links", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v struct {
			Label string `json:"label"`
		}
		if !read(w, r, &v) {
			return
		}
		grant, tok, e := s.store.CreateDebugLink(r.Context(), p, v.Label)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]any{"id": grant.ID, "label": grant.Label, "expires_at": grant.ExpiresAt, "url": s.config.Origin + "/#agent/" + tok})
	})
	s.human("GET /api/v1/debug-links", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		rows, e := s.store.DebugLinks(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, rows)
	})
	s.human("DELETE /api/v1/debug-links/{id}", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		if e := s.store.RevokeDebugLink(r.Context(), p, r.PathValue("id")); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.mux.HandleFunc("POST /auth/agent", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != s.config.Origin {
			fail(w, c.ErrForbidden)
			return
		}
		if !s.limiter.allow(r.RemoteAddr) {
			w.WriteHeader(429)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		var v struct {
			Token string `json:"token"`
		}
		if !read(w, r, &v) {
			return
		}
		age, e := s.store.DebugLogin(r.Context(), s.config.Workspace, v.Token)
		if e != nil {
			fail(w, e)
			return
		}
		s.cookie(w, v.Token, age)
		write(w, map[string]bool{"ok": true})
	})
}
