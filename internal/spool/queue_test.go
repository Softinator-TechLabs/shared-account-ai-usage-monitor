package spool

import (
	"os"
	"testing"
)

func TestDurableReplayAndAck(t *testing.T) {
	dir := t.TempDir()
	q := Queue{Dir: dir, MaxBytes: 1024}
	key, err := q.Append([]byte(`{"content":"Hinglish"}`))
	if err != nil {
		t.Fatal(err)
	}
	restarted := Queue{Dir: dir, MaxBytes: 1024}
	rows, err := restarted.Pending()
	if err != nil || len(rows) != 1 {
		t.Fatal("lost backlog", err)
	}
	if err = q.Ack(key); err != nil {
		t.Fatal(err)
	}
	rows, _ = restarted.Pending()
	if len(rows) != 0 {
		t.Fatal("ack not durable")
	}
}
func TestFullQueueAndPathTraversal(t *testing.T) {
	q := Queue{Dir: t.TempDir(), MaxBytes: 2}
	if _, err := q.Append([]byte("longer")); err == nil {
		t.Fatal("capacity ignored")
	}
	if err := q.Ack("../../victim"); err == nil {
		t.Fatal("path escape")
	}
}
func TestSymlinkNotRead(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir() + "/private"
	os.WriteFile(target, []byte("must not read"), 0600)
	os.Symlink(target, dir+"/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.json")
	rows, err := (&Queue{Dir: dir}).Pending()
	if err == nil || len(rows) != 0 {
		t.Fatal("symlink followed")
	}
}
