// Package companion coordinates acknowledged collection and replayable delivery.
package companion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/spool"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

var ErrUnavailable = errors.New("central server unavailable; queued data retained")

type HTTPError struct {
	Status int
	Code   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("central HTTP %d; queued data retained", e.Status)
}
func Offline(err error) bool {
	var h *HTTPError
	return errors.Is(err, ErrUnavailable) || (errors.As(err, &h) && h.Status >= 500)
}

type Client struct {
	base, token string
	http        *http.Client
}

func New(base, token string) (*Client, error) {
	u, e := url.Parse(base)
	if e != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("central URL must be an origin")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && net.ParseIP(u.Hostname()).IsLoopback()) {
		return nil, errors.New("central URL requires HTTPS; loopback is allowed for demos")
	}
	return &Client{base: base, token: token, http: &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }}}, nil
}
func (cl *Client) Request(ctx context.Context, method, path string, in, out any) error {
	var body []byte
	var e error
	if in != nil {
		body, e = json.Marshal(in)
		if e != nil {
			return e
		}
	}
	req, e := http.NewRequestWithContext(ctx, method, cl.base+path, bytes.NewReader(body))
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+cl.token)
	req.Header.Set("Content-Type", "application/json")
	res, e := cl.http.Do(req)
	if e != nil {
		return ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		var v struct {
			Error string `json:"error"`
		}
		json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&v)
		return &HTTPError{Status: res.StatusCode, Code: v.Error}
	}
	if out == nil {
		io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
		return nil
	}
	return json.NewDecoder(io.LimitReader(res.Body, 128<<20)).Decode(out)
}
func (cl *Client) Policy(ctx context.Context) (c.Policy, error) {
	var p c.Policy
	e := cl.Request(ctx, "GET", "/api/v1/device/policy", nil, &p)
	return p, e
}
func (cl *Client) Deliver(ctx context.Context, q *spool.Queue, version int) error {
	p, e := cl.Policy(ctx)
	if e != nil {
		return e
	}
	if p.Version != version {
		return errors.New("policy changed; collection and delivery paused until explicit acknowledgement")
	}
	rows, e := q.Pending()
	if e != nil {
		return e
	}
	for _, row := range rows {
		var v c.Snapshot
		if json.Unmarshal(row.Body, &v) != nil {
			return errors.New("invalid queued snapshot")
		}
		if v.PolicyVersion != version {
			return errors.New("old-policy queue requires explicit discard before resuming; nothing uploaded")
		}
	}
	for _, row := range rows {
		var ack struct {
			ID string `json:"id"`
		}
		if len(row.Body) > 4<<20 {
			digest := sha256.Sum256(row.Body)
			hash := hex.EncodeToString(digest[:])
			total := (len(row.Body) + (1 << 20) - 1) / (1 << 20)
			for i := 0; i < total; i++ {
				end := (i + 1) * (1 << 20)
				if end > len(row.Body) {
					end = len(row.Body)
				}
				if e = cl.Request(ctx, "POST", "/api/v1/ingest/chunk", map[string]any{"hash": hash, "index": i, "total": total, "body": row.Body[i*(1<<20) : end]}, nil); e != nil {
					return e
				}
			}
			e = cl.Request(ctx, "POST", "/api/v1/ingest/commit", map[string]string{"hash": hash}, &ack)
		} else {
			e = cl.Request(ctx, "POST", "/api/v1/ingest", json.RawMessage(row.Body), &ack)
		}
		if e != nil {
			var h *HTTPError
			if errors.As(e, &h) && h.Status == 410 && h.Code == "source_deleted" {
				var v c.Snapshot
				if json.Unmarshal(row.Body, &v) != nil {
					return e
				}
				if err := q.Suppress(v.SourceRef); err != nil {
					return err
				}
				if err := q.Ack(row.Key); err != nil {
					return err
				}
				continue
			}
			return e
		}
		if ack.ID == "" {
			return errors.New("missing durable acknowledgement")
		}
		if e = q.Ack(row.Key); e != nil {
			return e
		}
	}
	return nil
}
