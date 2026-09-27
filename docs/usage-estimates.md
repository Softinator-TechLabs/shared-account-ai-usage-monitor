# Understanding usage shares and subscription estimates

Shared Account AI Usage Monitor helps people understand which models and projects consumed their **captured AI activity**. It does not turn a subscription into an API bill or score an employee from token counts.

This page describes the `2026-09-26-standard-v1` calculation. Every analytics response includes the rate version, coverage, time zone and attribution limitations. The dashboard and read-only `usage_analytics` MCP tool use the same permission-scoped response.

## Three different measurements

| Measurement | Meaning | Example |
| --- | --- | --- |
| Observed account quota | The provider's reported utilization for a particular account and window | Weekly utilization rose from 30% to 34%: **4 percentage points (pp)** |
| Captured weighted usage share | Model-weighted token activity divided by captured weighted activity for that client | A project represents 75% of the captured Codex weight today |
| Conditional quota allocation | An observed quota increase distributed proportionally among associated captured activity | That project's estimated contribution to the 4 pp increase is **3 pp**, under the assumptions below |

**75% of captured activity does not mean 75% of the subscription allowance.** A $20 Plus, $100 Claude Max 5x or $200 Codex Pro subscription price is a plan attribute, not a token quota denominator. Provider limits vary with models, tasks, context, other product surfaces and policy. Paid extra credits and subscription allowances are distinct.

Accounts are identified by provider and account identity. `person@example.invalid` on Codex and the same email on Claude are separate accounts. Device enrollment identifies a device's assigned person; it does not prove who authored a historical prompt. Assignment labels alone never bind a session to a shared account.

## Model, effort and project pies

For each timestamped usage point:

```
weight = (uncached_input × input_rate
        + output × output_rate
        + cache_read × cache_read_rate
        + cache_write × cache_write_rate) / 1,000,000
share = group_weight / captured_client_weight
```

AgentsView normalizes Codex input to the **uncached** portion. Cached input is counted once; reasoning tokens are not added again to output. Rates have different units for different clients, so we never add Codex credit weights to Claude API-equivalent weights.

- **Codex:** published native credit rates at standard speed. Speed information is not available in this projection, so this is an explicitly standard-speed comparison proxy. Fast-mode usage can differ. Recorded reasoning effort groups the chart; we do not invent an effort multiplier.
- **Claude:** relative published API rates are a comparison proxy, **not Claude subscription metering**. The five-minute cache-write rate is assumed because the available counter does not identify cache TTL. A one-hour cache write may have a different rate. This is not money spent.
- **Antigravity and unknown clients/models:** counts can still appear, but no weighted share is invented without a supported rate and counter contract. Antigravity-hosted Claude is not silently treated as a Claude Code account.
- All four token categories must be reported for a point to be weighted. Unknown rates, unsupported categories and missing counters stay unweighted; the UI shows the excluded point count. A pie covers only its known-weight portion.
- Model, effort, project and branch come from recorded upstream metadata. Missing values remain Unknown. Project labels are not verified repository URLs.

The rate table is intentionally explicit in [`internal/usageweights/rates.go`](../internal/usageweights/rates.go). A provider's renamed or new model remains unknown until reviewed, with tests. Changes to rates get a new version. Currently the selected rate version is applied to the requested historical period for comparison; this is **not a historical price ledger**.

## Prompt and generated-line counters

The collector reads the public AgentsView Session API, not another copy of native Codex/Claude parsers. It projects counters locally and discards message/tool input text from this analytics payload. Full conversation policy and archive access remain separate.

- Prompts count timestamped human user messages. System/tool messages, known automated sources and automated sessions are excluded.
- When a prompt lacks model metadata, its model/effort can be associated with the next actual assistant response before the next human prompt. That is response association, not proof of what a person selected. Unknown remains unknown.
- **Proposed lines** count recognized structured write/edit/patch inputs. Write counts supplied content, Edit counts replacement text, and patch counts added lines. Replacement text can include unchanged context. Repeated edits count again, but duplicate tool IDs within a session do not.
- Tool output, shell stdout, prose and unsupported tool shapes are not counted as generated code. A proposed patch can fail or be reverted. These counts are **not net Git additions, accepted changes, delivered features or productivity**.
- Enrichment requires a stable upstream revision, complete ascending pagination and matching message count. Failures leave token counters usable but mark activity unavailable. Missing counts are not zero. Partial timestamp coverage is disclosed.

## Conditional quota allocation

The first adapter supports **observed Codex quota windows**. Claude and Antigravity quota adapters are not yet implemented; their subscription percentage cannot be derived by dividing API-equivalent cost by the monthly fee.

For each account, bucket and window:

1. Read timestamped provider quota observations and canonicalize duplicate simultaneous readings. Never sum the same account's readings from multiple PCs.
2. Require two readings within 15 minutes, a known unchanged plan, matching window/reset identity and non-decreasing utilization. Keep weekly and short windows separate. Resets, drops, conflicts, failed reads, gaps and partial period boundaries are not allocated.
3. Associate each captured Codex usage point only when its own device has matching account/profile observations before and after it, within 15 minutes. Multiple nearby profiles/accounts make that association ambiguous. This is still an **inferred association**, not per-request authentication evidence.
4. Use the entire authorized workspace denominator before applying a person/project filter. A personal filter never turns that person's partial share into 100% of account consumption. Restricted self-only visibility cannot establish a team denominator and gets no allocation.
5. Allocate the observed percentage-point increase in proportion to those associated weights. Unknown weights, undated points, conflicting copied sources or unbound points prevent an affected estimate. Truncated history never produces an allocation.

An estimated interval assumes **the captured, associated activity represents the account's consumption during that interval**. Other machines, cloud tasks, ChatGPT/Claude application use, delayed provider accounting or provider-specific weights may be missing. Therefore even a complete local capture cannot prove actual per-employee quota debits. This version is a proportional conditional estimate, not an empirically calibrated model; it does not claim an accuracy percentage.

Charts and MCP retain the observed increase separately from the allocated estimate and its reason. Account summaries total known same-cycle observed increases even when allocation is withheld; estimated totals include only eligible allocated intervals. Separate observed and allocated interval counts make this partial coverage explicit. Missing estimates remain Unknown. The UI groups only the same account/window/reset and discloses unavailable intervals. Never combine weekly and short-window pp into one total, and never extrapolate a partial set of intervals into an entire week's consumption.

## Periods, permissions and maintenance

**Today** means midnight Asia/Kolkata through the request time. Last 1/12/24/48 hours are rolling windows; timestamps are filtered before counting prompts, proposed lines or token shares. Source coverage counts describe available sources, including sources with no in-period point. Existing seven-to-ninety-day views remain available.

Analytics remain subject to workspace visibility, device/person permissions, retention, deletion and policy redaction. Device upload credentials cannot read analytics. No raw prompts, real team identities or credentials belong in this public repository; examples here are synthetic. Discussion/coaching should use the full session context and delivered work evidence with human review, not language fluency or raw activity rank.

The collector and AgentsView are independently versioned. We reuse AgentsView's public projection; upgrades are tested against synthetic contract fixtures and a local canary rather than auto-merging parser changes. An existing collector needs the enrichment-capable release before old sources gain prompt/effort/line metadata. Its `usage-v3` projection replaces each source snapshot instead of adding revision totals. Failed enrichment can retry during hourly reconciliation.

## Sources checked on 26 September 2026

- [OpenAI Codex pricing and native credit rates](https://learn.chatgpt.com/docs/pricing)
- [Claude model pricing](https://platform.claude.com/docs/en/about-claude/pricing)
- [Claude Max plan](https://support.claude.com/en/articles/11049741-what-is-the-max-plan)
- [Claude usage and length limits](https://support.claude.com/en/articles/11647753-how-do-usage-and-length-limits-work)
- [Antigravity plan changes](https://antigravity.google/blog/changes-to-antigravity-plans) and [plans](https://antigravity.google/docs/plans)
- [AgentsView](https://www.agentsview.io/) — upstream session reader and normalization. See [upstream credits](../THIRD_PARTY_NOTICES.md).

## Reading a newly reset account

If the provider shows 97% remaining, it reports 3% used for that account window. A project pie showing 100% means all **captured weighted activity in the selected period** belongs to that project; it does not by itself assign all 3 percentage points. The reset window and selected period must match, quota observations must bracket the activity, and missing intervals must remain unallocated.

Account summaries show the latest endpoint reading, the observed increase over captured same-cycle intervals and a project table of conditional allocated percentage points. For example, the first recorded reading might already be 2% used and a later one 3%. If the intervening activity is attributable to one project, the estimate can assign the **1 pp observed increase**; the earlier 2 pp stays unexplained. Choose a period containing the reset for context, and inspect interval coverage. Even sole use by one employee does not prove which project caused a provider debit.

Missing usage is scoped to the source's reported start/end range, widened to include all dated token/activity evidence. An old incomplete session outside a new interval does not block that interval. Unknown/inverted bounds still block conservatively, and conflicting source copies contribute every copy's possible range. This requires collector v0.6.0 or later to recapture optional `ended_at` metadata. Upload time is never substituted for session end time.
