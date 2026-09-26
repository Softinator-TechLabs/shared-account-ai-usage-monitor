package web

import (
	"context"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPasswordHTTPOriginCookieAndPeople(t *testing.T) {
	s := testutil.Database(t)
	ctx := context.Background()
	p := c.Principal{Workspace: "team", Person: "owner", Role: "owner"}
	s.SetPolicy(ctx, p, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	inv, err := s.InviteUser(ctx, p, "owner@example.com", "Owner", "owner")
	if err != nil {
		t.Fatal(err)
	}
	app, err := New(ctx, s, Config{Origin: "https://usage.example.com", Workspace: "team"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, origin, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		return w
	}
	body := `{"token":"` + inv + `","password":"synthetic password only"}`
	if w := request("/auth/setup", "https://evil.example", body); w.Code != 403 {
		t.Fatal("setup CSRF", w.Code)
	}
	if w := request("/auth/setup", "https://usage.example.com", body); w.Code != 200 {
		t.Fatal("setup", w.Code, w.Body.String())
	}
	w := request("/auth/password", "https://usage.example.com", `{"email":"owner@example.com","password":"synthetic password only"}`)
	if w.Code != 200 {
		t.Fatal("login", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatal("unsafe cookie", cookies)
	}
	r := httptest.NewRequest("GET", "/api/v1/people", nil)
	r.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "owner@example.com") {
		t.Fatal("people", w.Code, w.Body.String())
	}
}
