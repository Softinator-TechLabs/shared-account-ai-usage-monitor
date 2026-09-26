package web

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/testutil"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAgentDebugLinkScopeExpiryAndRevocation(t *testing.T) {
	s := testutil.Database(t)
	ctx := context.Background()
	p := c.Principal{Workspace: "team", Person: "owner", Role: "owner"}
	s.SetPolicy(ctx, p, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	s.SetMember(ctx, p, "owner", "owner", "")
	human, _ := s.IssueHumanToken(ctx, "team", "owner")
	app, _ := New(ctx, s, Config{Origin: "https://usage.example.com", Workspace: "team"})
	call := func(method, path, bearer, body, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		return w
	}
	w := call("POST", "/api/v1/debug-links", human, `{"label":"Codex debugging"}`, "")
	if w.Code != 200 {
		t.Fatal("create debug link", w.Code)
	}
	var grant struct {
		ID      string    `json:"id"`
		URL     string    `json:"url"`
		Expires time.Time `json:"expires_at"`
	}
	json.Unmarshal(w.Body.Bytes(), &grant)
	if delta := time.Until(grant.Expires); delta < 167*time.Hour || delta > 168*time.Hour {
		t.Fatal("not seven days", delta)
	}
	tok := strings.Split(grant.URL, "#agent/")[1]
	body := `{"token":"` + tok + `"}`
	if w := call("POST", "/auth/agent", "", body, "https://evil.example"); w.Code != 403 {
		t.Fatal("CSRF")
	}
	w = call("POST", "/auth/agent", "", body, "https://usage.example.com")
	if w.Code != 200 {
		t.Fatal("agent login", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].MaxAge > 604800 {
		t.Fatal("cookie scope")
	}
	if w := call("GET", "/api/v1/me", tok, "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"token_kind":"debug"`) {
		t.Fatal("agent identity", w.Code)
	}
	for _, path := range []string{"/api/v1/invitations", "/api/v1/debug-links", "/api/v1/access-token"} {
		if w := call("POST", path, tok, `{"person":"bob"}`, ""); w.Code != 403 {
			t.Fatal("agent mutation allowed", path, w.Code)
		}
	}
	// Cookie-based requests retain the same read-only boundary.
	r := httptest.NewRequest("POST", "/api/v1/invitations", strings.NewReader(`{"person":"bob"}`))
	r.AddCookie(cookies[0])
	r.Header.Set("Origin", "https://usage.example.com")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cookie gained writes")
	}
	s.DB.Exec(ctx, "UPDATE tm_tokens SET expires_at=now()-interval '1 second' WHERE digest=$1", grant.ID)
	if w := call("GET", "/api/v1/me", tok, "", ""); w.Code != 403 {
		t.Fatal("expired debug link")
	}
	s.DB.Exec(ctx, "UPDATE tm_tokens SET expires_at=now()+interval '1 hour' WHERE digest=$1", grant.ID)
	if w := call("DELETE", "/api/v1/debug-links/"+grant.ID, human, "", ""); w.Code != 200 {
		t.Fatal("revoke", w.Code)
	}
	if w := call("POST", "/auth/agent", "", body, "https://usage.example.com"); w.Code != 403 {
		t.Fatal("revoked link login")
	}
}
