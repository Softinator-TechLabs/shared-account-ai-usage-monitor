package web

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/http"
)

func (s *Server) quotaRoutes() {
	s.human("GET /api/v1/quota-observations", func(w http.ResponseWriter, r *http.Request, p c.Principal) {
		rows, e := s.store.NativeQuotas(r.Context(), p)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, rows)
	})
	s.mux.HandleFunc("POST /api/v1/device/quota-observations", func(w http.ResponseWriter, r *http.Request) {
		p, ok := s.principal(w, r, true)
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		var v c.QuotaObservation
		if !read(w, r, &v) {
			return
		}
		if e := s.store.ObserveQuota(r.Context(), p, v); e != nil {
			fail(w, e)
			return
		}
		write(w, map[string]string{"id": v.EventID})
	})
}
