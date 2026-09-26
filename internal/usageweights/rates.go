// Package usageweights supplies dated comparison weights, never subscription bills.
package usageweights

import (
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"regexp"
	"strings"
)

const Version = "2026-09-26-standard-v1"

// Input is upstream-normalized uncached input. See docs/usage-estimates.md.
type rate struct{ input, output, read, write float64 }

var codex = map[string]rate{
	"gpt-6-astra": {250, 1250, 25, 0}, "gpt-6-sol": {50, 250, 5, 0}, "gpt-6-luna": {2.5, 12.5, .25, 0},
	"gpt-5.6-sol": {100, 500, 10, 0}, "gpt-5.6-terra": {50, 300, 5, 0}, "gpt-5.6-luna": {5, 30, .5, 0},
	"gpt-5.5": {125, 750, 12.5, 0}, "gpt-5.4": {62.5, 375, 6.25, 0}, "gpt-5.4-mini": {18.75, 113, 1.875, 0},
}
var claude = map[string]rate{
	"claude-opus-5-5": {4, 20, .2, 5}, "claude-sonnet-5": {2, 10, .2, 2.5}, "claude-haiku-4-5": {1, 5, .1, 1.25},
	"claude-opus-5": {5, 25, .5, 6.25}, "claude-opus-4-8": {5, 25, .5, 6.25}, "claude-opus-4-7": {5, 25, .5, 6.25},
	"claude-opus-4-6": {5, 25, .5, 6.25}, "claude-opus-4-5": {5, 25, .5, 6.25},
	"claude-opus-4": {15, 75, 1.5, 18.75}, "claude-opus-4-1": {15, 75, 1.5, 18.75},
	"claude-sonnet-4-6": {3, 15, .3, 3.75}, "claude-sonnet-4-5": {3, 15, .3, 3.75}, "claude-sonnet-4": {3, 15, .3, 3.75},
}
var dated = regexp.MustCompile(`-20[0-9]{6}$`)

// Weight uses standard speed and, for Claude, a five-minute cache-write proxy.
// Unknown categories/rates return nil. Effort has no invented rate multiplier.
func Weight(client string, p c.UsagePoint) *float64 {
	var r rate
	var ok bool
	switch client {
	case "codex":
		r, ok = codex[p.Model]
	case "claude":
		r, ok = claude[dated.ReplaceAllString(strings.ReplaceAll(p.Model, ".", "-"), "")]
	default:
		return nil
	}
	if !ok || p.InputTokens == nil || p.OutputTokens == nil || p.CacheReadTokens == nil || p.CacheWriteTokens == nil {
		return nil
	}
	if client == "codex" && *p.CacheWriteTokens != 0 {
		return nil
	}
	for _, v := range []*int64{p.InputTokens, p.OutputTokens, p.CacheReadTokens, p.CacheWriteTokens} {
		if *v < 0 {
			return nil
		}
	}
	value := (float64(*p.InputTokens)*r.input + float64(*p.OutputTokens)*r.output + float64(*p.CacheReadTokens)*r.read + float64(*p.CacheWriteTokens)*r.write) / 1e6
	return &value
}
