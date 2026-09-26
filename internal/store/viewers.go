package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net"
	"net/url"
	"os"
	"strings"
)

type DeviceViewer struct {
	ID     string `json:"id"`
	Person string `json:"person"`
	Device string `json:"device"`
	URL    string `json:"url"`
	HasKey bool   `json:"has_key"`
}

func viewerOrigin(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if len(raw) > 2048 {
		return "", c.ErrInvalid
	}
	u, e := url.Parse(raw)
	if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", c.ErrInvalid
	}
	if u.Scheme == "http" {
		ip := net.ParseIP(u.Hostname())
		if u.Hostname() != "localhost" && (ip == nil || (!ip.IsLoopback() && !ip.IsPrivate())) {
			return "", c.ErrInvalid
		}
	}
	u.Path = ""
	return u.String(), nil
}
func viewerCipher(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("VIEWER_ENCRYPTION_KEY must be configured to store access keys")
	}
	b, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(b)
}
func viewerEnvKey() ([]byte, error) {
	v := os.Getenv("VIEWER_ENCRYPTION_KEY")
	if v == "" {
		return nil, nil
	}
	b, e := base64.StdEncoding.DecodeString(v)
	if e != nil || len(b) != 32 {
		return nil, errors.New("VIEWER_ENCRYPTION_KEY must be base64 encoded 32 bytes")
	}
	return b, nil
}
func (s *Store) DeviceViewers(ctx context.Context, p c.Principal) ([]DeviceViewer, error) {
	if p.Device != "" || p.Person == "" {
		return nil, c.ErrForbidden
	}
	policy, e := s.Policy(ctx, p.Workspace)
	if e != nil {
		return nil, e
	}
	rows, e := s.DB.Query(ctx, `SELECT t.digest,t.person,t.device,coalesce(v.url,''),coalesce(octet_length(v.encrypted_key)>0,false) FROM tm_tokens t LEFT JOIN tm_device_viewers v ON v.device_digest=t.digest JOIN tm_members m ON m.workspace=t.workspace AND m.person=t.person WHERE t.workspace=$1 AND t.kind='device' AND t.expires_at>now() AND m.active AND ($2 OR t.person=$3) ORDER BY t.person,t.device,t.digest`, p.Workspace, p.Manager() || policy.Visibility == "team", p.Person)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []DeviceViewer{}
	for rows.Next() {
		var v DeviceViewer
		if e = rows.Scan(&v.ID, &v.Person, &v.Device, &v.URL, &v.HasKey); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SetDeviceViewer(ctx context.Context, p c.Principal, id, raw string, key *string) error {
	if !p.Manager() || p.TokenKind != "human" {
		return c.ErrForbidden
	}
	origin, e := viewerOrigin(raw)
	if e != nil {
		return e
	}
	if key != nil && (len(*key) > 8192 || strings.ContainsAny(*key, "\r\n")) {
		return c.ErrInvalid
	}
	visible, e := s.DeviceViewers(ctx, p)
	if e != nil {
		return e
	}
	found := false
	oldURL := ""
	for _, v := range visible {
		if v.ID == id {
			found = true
			oldURL = v.URL
		}
	}
	if !found {
		return c.ErrNotFound
	}
	var sealed []byte
	if origin != "" && key != nil && *key != "" {
		a, e := viewerCipher(s.ViewerKey)
		if e != nil {
			return e
		}
		nonce := make([]byte, a.NonceSize())
		if _, e = rand.Read(nonce); e != nil {
			return e
		}
		sealed = a.Seal(nonce, nonce, []byte(*key), []byte(id+"\x00"+origin))
	}
	if origin == "" {
		_, e = s.DB.Exec(ctx, "DELETE FROM tm_device_viewers WHERE device_digest=$1", id)
	} else {
		_, e = s.DB.Exec(ctx, `INSERT INTO tm_device_viewers(device_digest,url,encrypted_key) VALUES($1,$2,$3) ON CONFLICT(device_digest) DO UPDATE SET url=excluded.url,encrypted_key=CASE WHEN $4 THEN excluded.encrypted_key ELSE tm_device_viewers.encrypted_key END`, id, origin, sealed, key != nil || oldURL != origin)
	}
	if e != nil {
		return e
	}
	return s.Audit(ctx, p, "device.viewer.configure", id)
}
func (s *Store) DeviceViewerKey(ctx context.Context, p c.Principal, id string) (string, error) {
	if p.TokenKind != "human" || p.Device != "" {
		return "", c.ErrForbidden
	}
	visible, e := s.DeviceViewers(ctx, p)
	if e != nil {
		return "", e
	}
	var target *DeviceViewer
	for i := range visible {
		if visible[i].ID == id {
			target = &visible[i]
			break
		}
	}
	if target == nil || (!p.Manager() && target.Person != p.Person) {
		return "", c.ErrForbidden
	}
	if !target.HasKey {
		return "", nil
	}
	var b []byte
	if e = s.DB.QueryRow(ctx, "SELECT encrypted_key FROM tm_device_viewers WHERE device_digest=$1", id).Scan(&b); e != nil {
		return "", e
	}
	a, e := viewerCipher(s.ViewerKey)
	if e != nil || len(b) < a.NonceSize() {
		return "", c.ErrInvalid
	}
	plain, e := a.Open(nil, b[:a.NonceSize()], b[a.NonceSize():], []byte(id+"\x00"+target.URL))
	if e != nil {
		return "", e
	}
	if e = s.Audit(ctx, p, "device.viewer.key.read", id); e != nil {
		return "", e
	}
	return string(plain), nil
}
