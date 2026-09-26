package web

import (
	"context"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/quota"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/testutil"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQuotaHTTPPermissionsAndReplay(t *testing.T) {
	s := testutil.Database(t)
	ctx := context.Background()
	owner := c.Principal{Workspace: "team", Person: "owner", Role: "owner"}
	s.SetPolicy(ctx, owner, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	s.SetMember(ctx, owner, "owner", "owner", "")
	human, _ := s.IssueHumanToken(ctx, "team", "owner")
	inv, _ := s.Invite(ctx, owner, "alice", "device", time.Now().Add(time.Hour))
	dev, _ := s.Enroll(ctx, inv, "synthetic-device", 1)
	app, _ := New(ctx, s, Config{Origin: "http://127.0.0.1:9999", Workspace: "team", Demo: true})
	v := c.QuotaObservation{EventID: "synthetic-event", Provider: "codex", Profile: "default", PolicyVersion: 1, Snapshot: quota.Snapshot{Email: "synthetic@example.test", ObservedAt: time.Now().UTC()}}
	body, _ := json.Marshal(v)
	call := func(method, path, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(string(body)))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		return w
	}
	if w := call("POST", "/api/v1/device/quota-observations", human); w.Code != 403 {
		t.Fatal(w.Code)
	}
	for i := 0; i < 2; i++ {
		if w := call("POST", "/api/v1/device/quota-observations", dev); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w := call("GET", "/api/v1/quota-observations", dev); w.Code != 403 {
		t.Fatal("device can read", w.Code)
	}
	if w := call("GET", "/api/v1/quota-observations", human); w.Code != 200 || !strings.Contains(w.Body.String(), "synthetic@example.test") {
		t.Fatal(w.Code, w.Body.String())
	}
	s.Revoke(ctx, owner, "alice")
	if w := call("POST", "/api/v1/device/quota-observations", dev); w.Code != 403 {
		t.Fatal("revoked device", w.Code)
	}
}
