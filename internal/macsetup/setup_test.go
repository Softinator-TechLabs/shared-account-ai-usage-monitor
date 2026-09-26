package macsetup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
