package store

import (
	"context"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"strings"
	"testing"
	"time"
)

func TestViewerOriginSecretIsolationAndVisibility(t *testing.T) {
	s := testStore(t)
	s.ViewerKey = []byte(strings.Repeat("x", 32))
	ctx := context.Background()
	own := owner()
	own.TokenKind = "human"
	inv, e := s.Invite(ctx, own, "alice", "device", time.Now().Add(time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	token, e := s.Enroll(ctx, inv, "mac", 1)
	if e != nil {
		t.Fatal(e)
	}
	id := Hash([]byte(token))
	key := "synthetic-viewer-secret"
	if e = s.SetDeviceViewer(ctx, own, id, "http://192.168.1.4:18087", &key); e != nil {
		t.Fatal(e)
	}
	list, e := s.DeviceViewers(ctx, own)
	if e != nil || len(list) != 1 || !list[0].HasKey {
		t.Fatal(list, e)
	}
	var raw []byte
	s.DB.QueryRow(ctx, "SELECT encrypted_key FROM tm_device_viewers WHERE device_digest=$1", id).Scan(&raw)
	if strings.Contains(string(raw), key) {
		t.Fatal("stored plaintext")
	}
	got, e := s.DeviceViewerKey(ctx, own, id)
	if e != nil || got != key {
		t.Fatal("decrypt", e)
	}
	for _, kind := range []string{"read", "debug", "device"} {
		p := own
		p.TokenKind = kind
		if _, e = s.DeviceViewerKey(ctx, p, id); !errors.Is(e, c.ErrForbidden) {
			t.Fatal("secret leaked", kind, e)
		}
	}
	if e = s.SetDeviceViewer(ctx, own, id, "https://example.test/?token=secret", nil); !errors.Is(e, c.ErrInvalid) {
		t.Fatal(e)
	}
	if e = s.SetDeviceViewer(ctx, own, id, "http://8.8.8.8:80", nil); !errors.Is(e, c.ErrInvalid) {
		t.Fatal(e)
	}
	if e = s.SetDeviceViewer(ctx, own, id, "http://192.168.1.5:18087", nil); e != nil {
		t.Fatal(e)
	}
	if got, e = s.DeviceViewerKey(ctx, own, id); e != nil || got != "" {
		t.Fatal("origin change retained secret", e)
	}
	s.SetPolicy(ctx, own, c.Policy{Version: 2, Content: "full", Redaction: "none", Visibility: "self_managers"})
	p := c.Principal{Workspace: "team", Person: "bob", Role: "member", TokenKind: "human"}
	if _, e = s.DeviceViewerKey(ctx, p, id); !errors.Is(e, c.ErrForbidden) {
		t.Fatal("crossperson key", e)
	}
}
