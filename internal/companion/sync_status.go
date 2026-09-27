package companion

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// SyncStatusWriter records only collector health, never source data or errors.
// The collector's existing process lock serializes writes to its own status file.
// Callers must report write failures without interrupting collection.
type SyncStatusWriter struct{ Path string }

type syncStatus struct {
	State         string `json:"state"`
	LastAttemptAt string `json:"last_attempt_at,omitempty"`
	LastSuccessAt string `json:"last_success_at,omitempty"`
	LastErrorAt   string `json:"last_error_at,omitempty"`
	LastUploadAt  string `json:"last_upload_at,omitempty"`
}

func (w SyncStatusWriter) Begin() error {
	return w.update(func(s *syncStatus, now string) { s.State = "syncing"; s.LastAttemptAt = now })
}

// Uploaded records source acknowledgement progress without claiming cycle success.
func (w SyncStatusWriter) Uploaded() error {
	return w.update(func(s *syncStatus, now string) { s.LastUploadAt = now })
}

// Finish receives the whole cycle result, including failed provider reads and
// offline capture. Successfully delivering an error observation is not success.
func (w SyncStatusWriter) Finish(cycleErr error) error {
	return w.update(func(s *syncStatus, now string) {
		if cycleErr != nil {
			s.State = "error"
			s.LastErrorAt = now
		} else {
			s.State = "ok"
			s.LastSuccessAt = now
		}
	})
}

func (w SyncStatusWriter) update(change func(*syncStatus, string)) error {
	status := syncStatus{}
	body, err := os.ReadFile(w.Path)
	if err == nil {
		if json.Unmarshal(body, &status) != nil {
			return errors.New("invalid collector status")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	change(&status, time.Now().UTC().Format(time.RFC3339Nano))
	body, err = json.Marshal(status)
	if err != nil {
		return err
	}
	dir := filepath.Dir(w.Path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".sync-status-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(body)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), w.Path)
}
