// Package agentsview is the only component coupled to the upstream Session API.
package agentsview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"
)

type Client struct {
	base  string
	token string
	http  *http.Client
}

func New(base, token string) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || !net.ParseIP(u.Hostname()).IsLoopback() || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return nil, errors.New("upstream must be an explicit loopback IP origin")
	}
	return &Client{base: base, token: token, http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("upstream redirect refused") }}}, nil
}
func (client *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", client.base+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	if client.token != "" {
		req.Header.Set("Authorization", "Bearer "+client.token)
	}
	res, err := client.http.Do(req)
	if err != nil {
		return errors.New("upstream unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("upstream HTTP %d", res.StatusCode)
	}
	// A bound is an explicit collection error, never successful truncation.
	b, err := io.ReadAll(io.LimitReader(res.Body, 128<<20+1))
	if err != nil {
		return err
	}
	if len(b) > 128<<20 {
		return errors.New("upstream page exceeds collection limit")
	}
	if !utf8.Valid(b) {
		return errors.New("upstream contains invalid UTF-8; fidelity cannot be preserved")
	}
	return json.Unmarshal(b, out)
}
func (client *Client) Collect(ctx context.Context, policyVersion int) ([]c.Snapshot, error) {
	out := []c.Snapshot{}
	err := client.CollectEach(ctx, policyVersion, nil, func(v c.Snapshot) error { out = append(out, v); return nil })
	return out, err
}

// CollectEach bounds memory to one source. skip receives immutable metadata plus its raw hash;
// callers may skip a source only after its previous snapshot was durably queued.
func (client *Client) CollectEach(ctx context.Context, policyVersion int, skip func(string, string) bool, emit func(c.Snapshot) error) error {
	cursor := ""
	seen := map[string]bool{}
	for {
		q := url.Values{"limit": {"100"}, "include_one_shot": {"true"}, "include_automated": {"true"}, "include_children": {"true"}, "include_source": {"true"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Sessions []json.RawMessage `json:"sessions"`
			Next     string            `json:"next_cursor"`
		}
		if err := client.get(ctx, "/api/v1/sessions", q, &page); err != nil {
			return err
		}
		if page.Sessions == nil {
			return errors.New("session enumeration capability missing")
		}
		for _, raw := range page.Sessions {
			var session struct {
				ID             string `json:"id"`
				Project        string `json:"project"`
				Agent          string `json:"agent"`
				Branch         string `json:"git_branch"`
				Started        string `json:"started_at"`
				NativeRevision string `json:"transcript_revision"`
			}
			if err := json.Unmarshal(raw, &session); err != nil {
				return err
			}
			if session.ID == "" || session.Agent == "" {
				return errors.New("upstream session identity missing")
			}
			metaHash := sha256.Sum256(raw)
			if skip != nil && skip(session.ID, hex.EncodeToString(metaHash[:])) && session.NativeRevision != "" {
				continue
			}
			snapshot := cSnapshot(session.ID, session.Project, session.Agent, session.Branch, session.Started, policyVersion, raw)
			start := 0
			for {
				var page struct {
					Messages []json.RawMessage `json:"messages"`
					More     bool              `json:"has_more"`
					Total    int               `json:"total"`
				}
				if err := client.get(ctx, "/api/v1/sessions/"+url.PathEscape(session.ID)+"/messages", url.Values{"from": {strconv.Itoa(start)}, "limit": {"100"}, "direction": {"asc"}}, &page); err != nil {
					return err
				}
				if page.Messages == nil {
					return errors.New("message capability missing")
				}
				for _, raw := range page.Messages {
					var fields map[string]json.RawMessage
					if json.Unmarshal(raw, &fields) != nil {
						return errors.New("invalid message")
					}
					if fields["content"] == nil || fields["ordinal"] == nil {
						return errors.New("message content/ordinal missing")
					}
					var msg c.Message
					if err := json.Unmarshal(raw, &msg); err != nil {
						return err
					}
					if msg.Ordinal < start {
						return errors.New("message cursor did not advance")
					}
					msg.Raw = raw
					snapshot.Messages = append(snapshot.Messages, msg)
				}
				if len(page.Messages) < 100 && !page.More && len(snapshot.Messages) >= page.Total {
					break
				}
				if len(page.Messages) == 0 {
					return errors.New("message pagination stalled")
				}
				start = snapshot.Messages[len(snapshot.Messages)-1].Ordinal + 1
			}
			// Released API lacks a pagination revision guard. Re-read the source revision
			// before acknowledging a multi-page capture; a moving source is retried later.
			if session.NativeRevision != "" {
				var after struct {
					Revision string `json:"transcript_revision"`
				}
				if err := client.get(ctx, "/api/v1/sessions/"+url.PathEscape(session.ID), nil, &after); err != nil {
					return err
				}
				if after.Revision != session.NativeRevision {
					return errors.New("upstream transcript changed during collection; retry required")
				}
			}
			// Revision depends on archived content, not collection time. Raw API IDs are retained, so a rebuilt upstream index may produce a new revision.
			rawBody, _ := json.Marshal(snapshot)
			sum := sha256.Sum256(rawBody)
			snapshot.Revision = hex.EncodeToString(sum[:])
			if err := emit(snapshot); err != nil {
				return err
			}
		}
		cursor = page.Next
		if cursor == "" {
			break
		}
		if seen[cursor] {
			return errors.New("session pagination stalled")
		}
		seen[cursor] = true
	}
	return nil
}
func cSnapshot(id, project, agent, branch, started string, policy int, raw json.RawMessage) c.Snapshot {
	return c.Snapshot{SchemaVersion: 1, SourceRef: id, PolicyVersion: policy, Client: agent, Project: project, Branch: branch, StartedAt: started, AccountMethod: "unknown", Coverage: "normalized_text; native identity and attachments not verified", Messages: []c.Message{}, Raw: raw}
}
