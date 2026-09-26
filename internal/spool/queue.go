// Package spool persists outbound records before delivery and never evicts unacknowledged content.
package spool

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
)

type Queue struct {
	Dir      string
	MaxBytes int64
	mu       sync.Mutex
}
type Item struct {
	Key  string
	Body []byte
}

var keyPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (q *Queue) Append(body []byte) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := os.MkdirAll(q.Dir, 0700); err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	key := hex.EncodeToString(sum[:])
	path := filepath.Join(q.Dir, key+".json")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return "", errors.New("unsafe spool entry")
		}
		return key, nil
	}
	rows, err := q.pending()
	if err != nil {
		return "", err
	}
	size := int64(len(body))
	for _, r := range rows {
		size += int64(len(r.Body))
	}
	if q.MaxBytes > 0 && size > q.MaxBytes {
		return "", errors.New("queue capacity exhausted; collection paused")
	}
	file, err := os.CreateTemp(q.Dir, ".pending-")
	if err != nil {
		return "", err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(body)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err = os.Rename(temp, path); err != nil {
		return "", err
	}
	return key, nil
}
func (q *Queue) Pending() ([]Item, error) { q.mu.Lock(); defer q.mu.Unlock(); return q.pending() }
func (q *Queue) pending() ([]Item, error) {
	entries, err := os.ReadDir(q.Dir)
	if os.IsNotExist(err) {
		return []Item{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Item{}
	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		key := name[:len(name)-5]
		if !keyPattern.MatchString(key) {
			return nil, errors.New("unexpected spool entry")
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, errors.New("unsafe spool entry")
		}
		b, err := os.ReadFile(filepath.Join(q.Dir, name))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != key {
			return nil, errors.New("spool checksum mismatch")
		}
		out = append(out, Item{Key: key, Body: b})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}
func (q *Queue) Ack(key string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !keyPattern.MatchString(key) {
		return errors.New("invalid spool key")
	}
	err := os.Remove(filepath.Join(q.Dir, key+".json"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Suppression contains only a source hash, never transcript text. It survives restarts.
func (q *Queue) Suppress(source string) error {
	dir := q.Dir + ".suppressed"
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	sum := sha256.Sum256([]byte(source))
	return os.WriteFile(filepath.Join(dir, hex.EncodeToString(sum[:])), []byte("source_deleted\n"), 0600)
}
func (q *Queue) Suppressed(source string) bool {
	sum := sha256.Sum256([]byte(source))
	_, e := os.Stat(filepath.Join(q.Dir+".suppressed", hex.EncodeToString(sum[:])))
	return e == nil
}
