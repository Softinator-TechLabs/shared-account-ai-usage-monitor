package web

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPIdentityAndCSRF(t *testing.T) {
	s := testutil.Database(t)
	ctx := context.Background()
	owner := c.Principal{Workspace: "team", Person: "owner", Role: "owner"}
	s.SetPolicy(ctx, owner, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	s.SetMember(ctx, owner, "owner", "owner", "")
	token, _ := s.IssueHumanToken(ctx, "team", "owner")
	inv, _ := s.Invite(ctx, owner, "owner", "device", time.Now().Add(time.Hour))
	dev, _ := s.Enroll(ctx, inv, "demo-mac", 1)
	app, err := New(ctx, s, Config{Origin: "http://127.0.0.1:9999", Workspace: "team", Demo: true})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, token, origin, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		return res
	}
	res := request("/api/v1/policy", dev, "", `{"version":2,"content":"full","redaction":"none","visibility":"team"}`)
	if res.Code != 403 {
		t.Fatal("device can administer", res.Code)
	}
	req := httptest.NewRequest("POST", "/api/v1/invitations", strings.NewReader(`{"person":"bob"}`))
	req.AddCookie(&http.Cookie{Name: "team_session", Value: token})
	req.Header.Set("Origin", "https://evil.example")
	res = httptest.NewRecorder()
	app.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatal("CSRF accepted", res.Code)
	}
	snap := c.Snapshot{SchemaVersion: 1, SourceRef: "fixture-1", Revision: "r1", PolicyVersion: 1, Client: "codex", Project: "fixture", Coverage: "synthetic", AccountMethod: "unknown", Messages: []c.Message{{Ordinal: 0, Role: "user", Content: "<script>alert(1)</script> Hinglish"}}}
	body, _ := json.Marshal(snap)
	res = request("/api/v1/ingest", dev, "", string(body))
	if res.Code != 200 {
		t.Fatal("ingest", res.Code, res.Body.String())
	}
	req = httptest.NewRequest("GET", "/api/v1/activity?q=Hinglish", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res = httptest.NewRecorder()
	app.ServeHTTP(res, req)
	if res.Code != 200 || !strings.Contains(res.Body.String(), "fixture-1") {
		t.Fatal("search", res.Code, res.Body.String())
	}
	req = httptest.NewRequest("GET", "/api/v1/activity", nil)
	req.Header.Set("Authorization", "Bearer "+dev)
	res = httptest.NewRecorder()
	app.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatal("device archive access", res.Code)
	}
}
func TestDemoOriginFailsClosed(t *testing.T) {
	if _, err := New(context.Background(), nil, Config{Origin: "https://public.example", Workspace: "team", Demo: true}); err == nil {
		t.Fatal("public demo allowed")
	}
}

func TestReadTokenCannotWriteAndMalformedAuthCannotBypassCSRF(t *testing.T) {
	s := testutil.Database(t)
	ctx := context.Background()
	p := c.Principal{Workspace: "team", Person: "owner", Role: "owner"}
	s.SetPolicy(ctx, p, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	s.SetMember(ctx, p, "owner", "owner", "")
	readToken, _ := s.ReadToken(ctx, p)
	humanToken, _ := s.IssueHumanToken(ctx, "team", "owner")
	app, _ := New(ctx, s, Config{Origin: "http://127.0.0.1:9999", Workspace: "team", Demo: true})
	for _, auth := range []string{"Bearer " + readToken, "NotBearer anything"} {
		r := httptest.NewRequest("POST", "/api/v1/invitations", strings.NewReader(`{"person":"bob"}`))
		r.AddCookie(&http.Cookie{Name: "team_session", Value: humanToken})
		r.Header.Set("Authorization", auth)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("write boundary bypassed", auth, w.Code)
		}
	}
}

func TestInvalidUTF8RejectedBeforeJSONDecoding(t *testing.T) {
	raw := append([]byte(`{"content":"bad`), append([]byte{0xff}, []byte(`text"}`)...)...)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", strings.NewReader(string(raw)))
	var out map[string]string
	if read(w, r, &out) || w.Code != 400 {
		t.Fatal("invalid bytes were repaired and accepted", w.Code)
	}
}
