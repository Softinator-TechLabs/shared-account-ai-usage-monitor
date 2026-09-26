package web

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net"
	"net/http"
	"sync"
	"time"
)

type attemptWindow struct {
	count   int
	expires time.Time
}
type authLimiter struct {
	mu      sync.Mutex
	clients map[string]attemptWindow
	total   attemptWindow
}

func (l *authLimiter) allow(address string) bool {
	host, _, _ := net.SplitHostPort(address) // Never trust arbitrary forwarded headers.
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.clients == nil {
		l.clients = map[string]attemptWindow{}
	}
	for k, v := range l.clients {
		if now.After(v.expires) {
			delete(l.clients, k)
		}
	}
	if now.After(l.total.expires) {
		l.total = attemptWindow{expires: now.Add(time.Minute)}
	}
	v := l.clients[host]
	if now.After(v.expires) {
		v = attemptWindow{expires: now.Add(time.Minute)}
	}
	if v.count >= 30 || l.total.count >= 120 || len(l.clients) >= 10000 {
		return false
	}
	v.count++
	l.total.count++
	l.clients[host] = v
	return true
}
func (s *Server) passwordRoutes() {
	s.mux.HandleFunc("GET /auth/options", func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]bool{"password": s.config.AuthMode != "oidc", "oidc": s.auth != nil, "demo": s.config.Demo})
	})
	public := func(path string, f func(http.ResponseWriter, *http.Request)) {
		s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if s.config.AuthMode == "oidc" || r.Header.Get("Origin") != s.config.Origin {
				fail(w, c.ErrForbidden)
				return
			}
			if !s.limiter.allow(r.RemoteAddr) {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 8192)
			f(w, r)
		})
	}
	public("POST /auth/password", func(w http.ResponseWriter, r *http.Request) {
		var v struct{ Email, Password string }
		if !read(w, r, &v) {
			return
		}
		tok, e := s.store.PasswordLogin(r.Context(), s.config.Workspace, v.Email, v.Password)
		if e != nil {
			fail(w, e)
			return
		}
		s.cookie(w, tok, 28800)
		write(w, map[string]bool{"ok": true})
	})
	public("POST /auth/setup", func(w http.ResponseWriter, r *http.Request) {
		var v struct{ Token, Password string }
		if !read(w, r, &v) {
			return
		}
		if e := s.store.AcceptUserInvite(r.Context(), v.Token, v.Password); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.human("POST /api/v1/user-invitations", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v struct{ Email, Name, Role string }
		if !read(w, r, &v) {
			return
		}
		tok, e := s.store.InviteUser(r.Context(), p, v.Email, v.Name, v.Role)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]string{"url": s.config.Origin + "/#setup/" + tok, "expires_in": "24 hours"})
	})
	s.human("GET /api/v1/people", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		rows, e := s.store.People(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, rows)
	})
	s.human("DELETE /api/v1/devices/{id}", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		if e := s.store.RevokeDevice(r.Context(), p, r.PathValue("id")); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.mux.HandleFunc("POST /api/v1/device/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.principal(w, r, true); !ok {
			return
		}
		var v struct{ OS string }
		if !read(w, r, &v) {
			return
		}
		if e := s.store.Heartbeat(r.Context(), token(r), v.OS); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
}
