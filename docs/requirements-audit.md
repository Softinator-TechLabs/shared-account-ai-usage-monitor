# Original requirements audit — 26 September 2026

The product is an experimental owner pilot, not a complete employee telemetry system. This audit compares the original full-history/shared-subscription/coaching mission with the approved [AgentsView-first migration](agentsview-first.md). Organization identities, prompts, live counts and credentials remain in the private management repository.

## Gaps that affect the owner's daily workflow

| Requirement | Implemented evidence | Remaining work / acceptance gate | Priority |
| --- | --- | --- | --- |
| Every acknowledged Codex/Claude session centrally searchable | Session API archive, revisions, durable queue, selected full-fidelity proof | Oversized history fails the existing whole-session JSONB store; delivery stops behind the failed capture. Reconcile every upstream source/revision and test >256 MiB UTF-8 history before claiming completeness. Never clear the queue to make status green. | P0 |
| Independent device and conversation-sync health | Enrollment, session heartbeat, independent Codex quota receipts | People now distinguishes fresh quota receipts from stale session sync. Need per-device queue size, last successful source, error code and coverage ledger. A quota receipt is not proof of successful conversation sync. | P0 |
| Shared central AgentsView | Released upstream PostgreSQL pilot and failure receipt | Direct shared writer failed device isolation; permissioned transport and authenticated reader gateway remain unimplemented. No broad employee database credentials. | P0 |
| Per-device viewer URL and separate access key | Mac app opens local viewer and copies local key separately | Web deep links still target the browsing machine's loopback. Need explicit per-device reachable address, authorized key delivery/rotation, LAN permission UX and offline/revocation tests. Never imply localhost opens another person's computer. | P1 |
| Person, device, project and prompt navigation | Directory, source-owner session filter, archived project/branch/model details | Directory now shows top five recorded projects for sessions started in the last 14 IST calendar days. This is partial historical evidence, not live current assignment. Full device coverage and project/repo aliases remain needed. | P1 |
| Shared account current usage and reset | Native default/profile Codex adapter and account observations | Current default-profile owner pilot only; Cockpit multi-instance/profile coverage, Claude and Antigravity quota adapters need device proof. Account-wide readings are not summed across devices. | P1 |
| Which employee consumed what percentage | Declared assignments and observed quota-device context | Event-time session/profile/account binding, concurrent-device coverage, reset/gap handling and a documented allocation method are missing. Native account totals do not expose exact prompt debits. Exact share stays unknown; any future estimate must label assumptions and unallocated usage. | P1 |
| Full history, all operating systems/clients | Mac preview and six cross-build targets | Windows/Ubuntu install, login, startup/sleep/wake/uninstall, Antigravity fidelity and attachments remain unverified. Cross-compilation is not runtime proof. | P1 |
| Discussion, prompt and work ratings, agent coaching | Immutable review anchors, comments, separate ratings, scoped draft submission, read-only MCP | Native analysis execution and model provenance remain unproved; upstream-reader review-anchor migration and central agent context need validation. No English-fluency or token-count productivity ranking. | P2 |
| Delivered work and daily/weekly project progress | Private GitHub evidence/report harness | Central PR/check/diff links, feature acceptance criteria and session→outcome joins are missing. File edit tool inputs are not proof that code shipped. Human review remains required. | P2 |
| Low maintenance open source foundation | Separate clean public tree, upstream credit, pinned interfaces, CI | Central migration, compatibility canaries and backup/restore/cutover rehearsal are still open. Do not grow another general-purpose transcript reader while upstream reuse is being validated. | P1 |

## Current directory increment

An enrolled person gets **Add another device**, not a repeated first-connection call to action. People appears before workspace charts. The account card explicitly says **account total** and **Personal quota share: unknown**. Declared assignment and observed device evidence remain separate. No recent connection, expired enrollment and quota-connected/session-stale states are distinct. Refresh reloads people, quota observations and activity.

Recorded project counts deduplicate immutable captures by source, use compact metrics and inherit person/workspace visibility. They are not a ranking or a statement that the employee is currently working on that project. Historical author identity remains unknown where collection cannot prove it.

## Next implementation sequence

1. Pass the existing central-transport authorization gate and oversized-history fidelity/recovery test. Preserve the existing archive, spool and review anchors until reconciliation succeeds.
2. Surface a delivery ledger with upstream indexed, queued, accepted and failed source counts. Prove Codex and Claude history parity on the owner pilot.
3. Add per-device viewer registration and separate URL/key access, with local-network permissions and authenticated central access kept distinct.
4. Capture timed profile/account bindings and quota windows across concurrent devices; expose measurable account totals, unknown allocation and only defensible estimates.
5. Validate employee OS/client matrix, review-anchor migration, native analysis and Git outcome links before a team-wide release claim.

These gates are still open. UI refinements in this change do not resolve the history backlog or implement exact employee percentages.
