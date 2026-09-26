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

func TestUsageHTTPPermissionsAndDeletion(t *testing.T) {
	s := testutil.Database(t)
	ctx := context.Background()
	owner := c.Principal{Workspace: "team", Person: "owner", Role: "owner"}
	s.SetPolicy(ctx, owner, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	s.SetMember(ctx, owner, "owner", "owner", "")
	human, _ := s.IssueHumanToken(ctx, "team", "owner")
	inv, _ := s.Invite(ctx, owner, "alice", "device", time.Now().Add(time.Hour))
	dev, _ := s.Enroll(ctx, inv, "synthetic-device", 1)
	app, _ := New(ctx, s, Config{Origin: "http://127.0.0.1:9999", Workspace: "team", Demo: true})
	n := int64(42)
	v := c.UsageCapture{SourceRef: "synthetic-usage", Revision: "1", PolicyVersion: 1, Client: "claude", Project: "synthetic-project", ObservedAt: time.Now().UTC(), Coverage: "reported", Points: []c.UsagePoint{{Timestamp: time.Now().UTC().Format(time.RFC3339), OutputTokens: &n}}}
	b, _ := json.Marshal(v)
	call := func(method, path, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(string(b)))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		return w
	}
	if w := call("POST", "/api/v1/device/usage", human); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/v1/device/usage", dev); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("GET", "/api/v1/analytics", dev); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call("GET", "/api/v1/analytics?days=14&person=alice", human); w.Code != 200 || !strings.Contains(w.Body.String(), `"output_tokens":42`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("GET", "/api/v1/analytics?days=oops", human); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if e := s.DeleteSource(ctx, owner, v.SourceRef); e != nil {
		t.Fatal(e)
	}
	if w := call("POST", "/api/v1/device/usage", dev); w.Code != 410 {
		t.Fatal(w.Code, w.Body.String())
	}
}
