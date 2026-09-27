package agentsview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"net/url"
	"time"
)

// Usage enumeration is independent of transcript collection. Only typed metadata
// and token counters survive projection; upstream labels, prices and raw data do not.
type usageSession struct {
	ID               string `json:"id"`
	Agent            string `json:"agent"`
	Project          string `json:"project"`
	Branch           string `json:"git_branch"`
	Started          string `json:"started_at"`
	Ended            string `json:"ended_at"`
	NativeRevision   string `json:"transcript_revision"`
	Messages         int    `json:"message_count"`
	Prompts          int    `json:"user_message_count"`
	Automated        bool   `json:"is_automated"`
	RelationshipType string `json:"relationship_type"`
}

// CollectUsageEach skips only caller-acknowledged captures. The skip callback can
// fail closed on a policy change before any per-source usage read. Individual bad
// sources do not strand the remaining history; their count and first error return.
func (client *Client) CollectUsageEach(ctx context.Context, version int, skip func(string, string) (bool, error), emit func(c.UsageCapture) error) error {
	cursor := ""
	seen := map[string]bool{}
	failures := 0
	var first error
	for {
		q := url.Values{"limit": {"100"}, "include_one_shot": {"true"}, "include_automated": {"true"}, "include_children": {"true"}, "include_source": {"true"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Sessions []usageSession `json:"sessions"`
			Next     string         `json:"next_cursor"`
		}
		if err := client.get(ctx, "/api/v1/sessions", q, &page); err != nil {
			return err
		}
		if page.Sessions == nil {
			return errors.New("usage session enumeration capability missing")
		}
		for _, session := range page.Sessions {
			if err := ctx.Err(); err != nil {
				return err
			}
			if session.ID == "" || session.Agent == "" {
				return errors.New("usage source identity missing")
			}
			metadata, _ := json.Marshal(session)
			sum := sha256.Sum256(append([]byte("usage-v3:"), metadata...))
			revision := hex.EncodeToString(sum[:])
			if skip != nil {
				yes, err := skip(session.ID, revision)
				if err != nil {
					return err
				}
				if yes {
					continue
				}
			}
			capture, err := client.collectUsage(ctx, session, revision, version)
			if err != nil {
				failures++
				if first == nil {
					first = err
				}
				continue
			}
			if err = emit(capture); err != nil {
				return err
			}
		}
		cursor = page.Next
		if cursor == "" {
			break
		}
		if seen[cursor] {
			return errors.New("usage session pagination stalled")
		}
		seen[cursor] = true
	}
	if failures > 0 {
		return fmt.Errorf("usage collection failed for %d sources: %w", failures, first)
	}
	return nil
}

func (client *Client) collectUsage(ctx context.Context, s usageSession, revision string, version int) (c.UsageCapture, error) {
	out := c.UsageCapture{SourceRef: s.ID, Revision: revision, PolicyVersion: version, Client: s.Agent, Project: s.Project, Branch: s.Branch, StartedAt: s.Started, EndedAt: s.Ended, ObservedAt: time.Now().UTC(), Messages: s.Messages, Prompts: s.Prompts, Coverage: "reported", Points: []c.UsagePoint{}}
	var usage struct {
		Count        *int  `json:"breakdown_count"`
		HasTokenData *bool `json:"has_token_data"`
		Breakdown    []struct {
			MessageOrdinal *int   `json:"message_ordinal"`
			Timestamp      string `json:"timestamp"`
			Model          string `json:"model"`
			Input          *int64 `json:"input_tokens"`
			Output         *int64 `json:"output_tokens"`
			CacheWrite     *int64 `json:"cache_creation_input_tokens"`
			CacheRead      *int64 `json:"cache_read_input_tokens"`
		} `json:"breakdown"`
	}
	err := client.get(ctx, "/api/v1/sessions/"+url.PathEscape(s.ID)+"/usage", url.Values{"breakdown": {"true"}}, &usage)
	if errors.Is(err, errUnsupported) {
		out.Coverage = "unavailable"
	} else if err != nil {
		return out, err
	} else {
		if usage.Count == nil {
			out.Coverage = "unavailable"
		} else if *usage.Count != len(usage.Breakdown) {
			return out, errors.New("incomplete usage breakdown")
		} else {
			if usage.HasTokenData != nil && !*usage.HasTokenData {
				out.Coverage = "unavailable"
			}
			for _, row := range usage.Breakdown {
				for _, n := range []*int64{row.Input, row.Output, row.CacheRead, row.CacheWrite} {
					if n != nil && *n < 0 {
						return out, errors.New("negative upstream token counter")
					}
				}
				out.Points = append(out.Points, c.UsagePoint{Timestamp: row.Timestamp, Model: row.Model, InputTokens: row.Input, OutputTokens: row.Output, CacheReadTokens: row.CacheRead, CacheWriteTokens: row.CacheWrite})
			}
		}
	}
	// Enrichment failures leave authoritative token rows usable. No raw message
	// content or tool inputs become part of the capture.
	activity, efforts, coverage := client.collectUsageActivity(ctx, s)
	out.Activity, out.ActivityCoverage = activity, coverage
	if coverage != "unavailable" {
		for i, row := range usage.Breakdown {
			if row.MessageOrdinal != nil && i < len(out.Points) {
				meta := efforts[*row.MessageOrdinal]
				if meta.model == row.Model {
					out.Points[i].Effort = meta.effort
				}
			}
		}
	}
	if s.NativeRevision != "" {
		var after struct {
			Revision string `json:"transcript_revision"`
		}
		if err := client.get(ctx, "/api/v1/sessions/"+url.PathEscape(s.ID), nil, &after); err != nil {
			return out, err
		}
		if after.Revision != s.NativeRevision {
			return out, errors.New("upstream usage revision changed during collection; retry required")
		}
	}
	return out, nil
}
