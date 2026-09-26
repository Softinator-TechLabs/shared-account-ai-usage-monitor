package web

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/store"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) usageRoutes() {
	s.mux.HandleFunc("POST /api/v1/device/usage", func(w http.ResponseWriter, r *http.Request) {
		p, ok := s.principal(w, r, true)
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
		var v c.UsageCapture
		if !read(w, r, &v) {
			return
		}
		if e := s.store.ObserveUsage(r.Context(), p, v); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]string{"id": v.SourceRef})
	})
	s.human("GET /api/v1/analytics", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		q := r.URL.Query()
		days := 14
		if q.Get("days") != "" {
			var e error
			days, e = strconv.Atoi(q.Get("days"))
			if e != nil {
				fail(w, c.ErrInvalid)
				return
			}
		}
		v, e := s.store.Analytics(r.Context(), p, store.UsageFilter{Days: days, Person: q.Get("person"), Project: q.Get("project"), Client: q.Get("client")}, time.Now())
		if e != nil {
			fail(w, e)
			return
		}
		write(w, v)
	})
	s.human("GET /api/v1/device-viewers", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		v, e := s.store.DeviceViewers(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, v)
	})
	s.human("POST /api/v1/devices/{id}/viewer", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		var v struct {
			URL string  `json:"url"`
			Key *string `json:"key"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if !read(w, r, &v) {
			return
		}
		if e := s.store.SetDeviceViewer(r.Context(), p, r.PathValue("id"), v.URL, v.Key); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]bool{"ok": true})
	})
	s.human("GET /api/v1/devices/{id}/viewer-key", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		v, e := s.store.DeviceViewerKey(r.Context(), p, r.PathValue("id"))
		if e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]string{"key": v})
	})
}
