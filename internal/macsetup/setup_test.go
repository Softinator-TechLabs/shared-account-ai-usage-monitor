package macsetup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestInvitationValidation(t *testing.T) {
	for _, server := range []string{"https://workspace.example.com", "http://127.0.0.1:8090"} {
		v := Invitation{Server: server, Invitation: "synthetic", Policy: Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team", Retention: 90}}
		e := v.Validate()
		if server[:5] == "https" && e != nil {
			t.Fatal(e)
		}
		if server[:5] != "https" && e == nil {
			t.Fatal("accepted plaintext invitation")
		}
	}
	for _, server := range []string{"https://user:pass@example.com", "https://example.com/path", "https://example.com#fragment"} {
		v := Invitation{Server: server, Invitation: "x", Policy: Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"}}
		if v.Validate() == nil {
			t.Fatal("accepted", server)
		}
	}
}
func TestArchiveRejectsLinksAndRequiresPinnedDigest(t *testing.T) {
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	w := tar.NewWriter(z)
	w.WriteHeader(&tar.Header{Name: "agentsview", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
	w.Close()
	z.Close()
	if _, e := extract(b.Bytes()); e == nil {
		t.Fatal("accepted symlink")
	}
	if verify([]byte("tampered"), "0000") == nil {
		t.Fatal("checksum mismatch accepted")
	}
}

func TestPausePersistsEvenWhenServiceIsNotLoaded(t *testing.T) {
	for _, loaded := range []bool{false, true} {
		var calls []string
		run := func(_ context.Context, args ...string) error { calls = append(calls, args[1]); return nil }
		if e := serviceAction(context.Background(), "gui/1", "service.plist", "gui/1/service", false, func() bool { return loaded }, run); e != nil {
			t.Fatal(e)
		}
		want := []string{"disable"}
		if loaded {
			want = append(want, "bootout")
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatal(calls)
		}
	}
}
func TestResumeCanRetryFailedStartupWithoutReenrollment(t *testing.T) {
	var calls []string
	run := func(_ context.Context, args ...string) error {
		calls = append(calls, args[1])
		if args[1] == "bootstrap" && len(calls) == 2 {
			return errors.New("temporary startup failure")
		}
		return nil
	}
	action := func() error {
		return serviceAction(context.Background(), "gui/1", "service.plist", "gui/1/service", true, func() bool { return false }, run)
	}
	if action() == nil {
		t.Fatal("expected first bootstrap failure")
	}
	if e := action(); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(calls, []string{"enable", "bootstrap", "enable", "bootstrap"}) {
		t.Fatal(calls)
	}
}
