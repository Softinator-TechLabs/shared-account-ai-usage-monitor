# AgentsView-first central telemetry design

Approved direction: 2026-09-26. Implementation of this migration is pending. This decision supersedes the original plan to expand our own transcript browser and full-history store. The deployed prototype remains available during validation.

## Ownership and maintenance

| Capability | Owner after migration | Current evidence |
| --- | --- | --- |
| Native parsing, session reader, search, session analytics | Independently versioned official AgentsView | Local viewer works; installed v0.44.0 exposes PostgreSQL push/serve commands |
| Central transcript sync and storage | Upstream PostgreSQL sync, subject to pilot gates | Documented upstream; not yet tested in our central deployment |
| People, devices, permissions, consent and account inventory | Shared Account AI Usage Monitor | Experimental implementation exists |
| Observed account/profile bindings, quotas and reset history | Small provider adapters plus our timeline | Manual observations exist; one Codex read-only protocol probe succeeded |
| Prompt discussions, separate prompt/work ratings, agent drafts, Git evidence | Our team layer with immutable source references | Existing review anchors must survive migration |
| Account login/session switching | Native clients and existing account manager | No new credential pooling or automatic rotation |

Stop adding a second general-purpose transcript reader, parser or analytics engine. Repairs needed to preserve existing data and access remain in scope. Keep AgentsView credit and license notices. Do not fork upstream or import its internal Go packages/database schema. Use public interfaces with a pinned compatibility contract.

```text
Native clients → local official AgentsView → upstream PostgreSQL sync
                                                ↓ private database
Team login → authenticated viewer gateway → central official AgentsView UI
    └──────→ our people/accounts/quota/coaching UI and permissioned MCP
Device companion → observed profile/account/quota events → our private database
```

The diagram is the target, not a deployed topology. Separate upstream and team-layer schemas/roles and backup ownership. Use an explicit viewer launch/link first; iframe support is not assumed. Cross-view review links resolve through a small tested source adapter.

## Central pilot and access boundary

Start with the documented `agentsview pg push --watch` and `agentsview pg serve` pathway. Pin the released binary and checksums. Installed v0.44.0 is the candidate, not a compatibility promise; newer checkout functionality must not be treated as released. Hosted raw sync is an alternative only if the released capability and maintenance trade-off are separately verified.

Pilot on an isolated synthetic database and private network. Never publish PostgreSQL to the Internet or distribute an organization-wide administrator DSN. Verify whether the pinned upstream can operate with acceptable per-device database roles, revocation and integrity boundaries. If write privileges allow reading/overwriting other devices or cannot meet the access contract, stop this transport gate and evaluate a scoped upload transport; do not weaken authorization to ship.

The viewer must sit behind our authenticated gateway. Keep its upstream bearer credential on the server, strip incoming authorization/cookie forwarding as appropriate, and make direct upstream access unreachable externally. Gate HTML, API, assets, exports, deep links and MCP consistently. Enforce read-only upstream operations in the pilot. Test sign-out, member revocation and debug-link expiry.

An upstream shared viewer does not implement our per-person visibility. Initial central pilot is only for explicitly acknowledged `full/none/team` workspaces. Metadata, secret-filtered or `self_managers` workspaces remain on the existing supported path until equivalent upstream isolation/filtering is verified. Do not route their full local history through PG push. Enrollment and policy acknowledgement must gate the upstream background service as well as our companion; revocation/policy changes must stop future upload and retain the existing retention/deletion contract.

## Account and quota evidence

Codex adapter uses the documented local App Server protocol: `account/read` with `refreshToken:false`, `account/rateLimits/read`, and account/rate-limit notifications where supported. It starts no coding turn, changes no account, redeems no reset credits and uploads no OAuth tokens. Executable/profile paths are explicit; do not depend on a shell alias resolving in an OS background service.

Each observation records provider, confirmed account reference, device/profile/instance, UTC observation time, source version, bucket ID, used percentage, window duration and scheduled reset. Prefer `rateLimitsByLimitId`; use legacy `rateLimits` only when the newer map is absent. Missing values stay null; remaining percentage is computed only from a present used percentage. Failed reads create stale/error coverage, never a zero-usage observation.

Account identity is distinct from employee identity. Where only an email is exposed, record that evidence without inventing a provider account ID. A present CLI profile proves only its current observed identity. Session/turn bindings require runtime-specific evidence; profile reuse, account changes and concurrent instances cannot silently reuse a global current account. Historical account and author remain unknown when unobserved.

Store account-window observations once for aggregation, while retaining device provenance. Preserve out-of-order evidence, last-good freshness and contradictions. Scheduled reset time is distinct from an observed window transition. Never sum the same shared account's percentage once per employee/device. Display raw per-session token semantics separately from provider quota.

Account change from 40% to 48% means 8 percentage points observed for that account window. It does not establish a project's or employee's debit. Any future modeled allocation must carry method, coverage, estimated label and an unallocated remainder. No API-equivalent dollar amount is a subscription bill. Claude and Antigravity need independent adapters and receipts; Codex support does not imply their support.

## Coaching and migration integrity

Retain existing workspace/source/revision/message-ordinal anchors and comments. New upstream references need instance/source identifiers and a content/revision fingerprint; an upstream database-local ID alone is insufficient. When an index changes or content cannot be matched, mark the reference unresolved and preserve the original reviewed capture. Never move a rating to a merely similar prompt.

Human comments, agent drafts and independent prompt/work ratings continue through our authorization layer. Read-only MCP can join accessible source context and quota evidence; imported prompts cannot instruct it to take actions. Work ratings need delivery evidence. Neither token volume, prompt length, English/Hinglish fluency nor an inferred mental-effort score measures employee productivity.

Test synthetic sessions above the previously observed 256 MiB JSONB-element failure boundary, long UTF-8 text, tool data, copied/forked sessions, overlapping machines, interrupted uploads, upgrades and restore. Verify canonical exported content hashes, coverage and comment anchors before switching the default reader. Keep the existing archive/queue intact, with explicit ownership of dual ingestion and rollback. Do not silently recollect deleted records. Local viewer access is confirmed by the owner; full historical import is not.

## Optional office LAN access

Central outbound sync needs no inbound laptop firewall exemption. Peer LAN viewing is optional, disabled by default, and separate from central deployment. If implemented, request OS-admin permission for a scoped port/application rule on trusted networks, require authentication plus HTTPS or encrypted private overlay, and expose revoke/disable controls. Current loopback-only links stay unchanged until an allowlisted LAN destination feature is tested. Do not disable firewalls or open unauthenticated viewers.

## Upgrade and release contract

Pin upstream releases; test recorded API fixtures and synthetic end-to-end fidelity, then canary on one device and isolated server. Upgrade the upstream binary independently when compatible. Change our adapter only when its contract changes. No promise of maintenance-free updates and no blind automatic parser upgrades. macOS, Windows and Ubuntu startup, sleep/wake, offline replay, uninstall and credential revocation need separate receipts; CLI cross-builds are insufficient.

## Sources and acceptance

- [AgentsView PostgreSQL sync](https://www.agentsview.io/docs/pg-sync/)
- [AgentsView remote access](https://www.agentsview.io/docs/remote-access/)
- [AgentsView hosted raw sync](https://www.agentsview.io/docs/hosted-raw-sync/)
- [Codex App Server protocol](https://learn.chatgpt.com/docs/app-server)

Research checked 2026-09-26; local command availability and one read-only Codex probe are recorded separately from unperformed central/device gates. Follow the [implementation plan](superpowers/plans/2026-09-26-agentsview-first.md). Do not mark migration complete until those gates have receipts.
