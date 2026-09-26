# People and project analytics implementation plan

> **For agentic workers:** use the executing-plans workflow for each isolated task and request a bounded review before merge.

**Goal:** employee detail and project overview with daily recorded token use, shared-account context and device viewer access, also available through permissioned MCP.

**Architecture:** Extend the approved team layer with compact per-source usage facts from the official AgentsView Session Usage API. An independent analytics collector must not wait behind transcript delivery. Upstream owns native parsing and token normalization; this service joins enrolled device/person context and aggregates the same facts for web and MCP. No API-equivalent costs are retained.

**User direction:** 26 September: analytics is primary; full conversations belong in AgentsView. Same email at Claude and Codex identifies two accounts. Exact individual quota shares remain unknown. Existing implementation/deployment authorization continues.

**Tech stack:** Go/PostgreSQL, existing embedded JS/CSS; no new runtime dependencies.

## Contracts

`contracts.UsageCapture`: source_ref, revision (metadata hash), policy_version, client, project, branch, started_at, observed_at (UTC), messages, prompts, coverage (reported/unavailable), points[]. Each point: timestamp, model, input_tokens/output_tokens/cache_read_tokens/cache_write_tokens pointers (null unknown). Per-source API uses breakdown=true, no subagent rollup. No prompt, cost, bearer key or raw field survives projection.

POST `/api/v1/device/usage` upload-only; latest capture per workspace/device/person/source. New observation replaces that source's counters, never sums revisions. Source deletion blocks recollection and removes analytics. Expiry/policy visibility apply. Cross-device source copies deduplicate globally; conflicting owners count as unknown attribution, not two employees. Unknown timestamps excluded from dated charts and counted in coverage. Four token counters remain separate; no context/output double count or percentage inference.

GET `/api/v1/analytics?days=7|14|30|90&person=&project=&client=` returns `start,end,timezone,coverage{sources,unavailable_sources,undated_points,attribution_conflicts,last_observed_at},totals{input_tokens,output_tokens,cache_read_tokens,cache_write_tokens,points},daily[] {day,+counters},people[] {person,+counters,sources},projects[] {project,+counters,people:[ids],sources},clients[] {client,+counters},models[] {model,+counters}`. Aggregates have nullable counters when no reported value; missing imports are not zero use. Project grouping uses recorded names, explicitly not a verified repository identity. The same endpoint powers MCP `usage_analytics`.

Device viewer settings: GET `/api/v1/device-viewers` returns visible `{id,person,device,url}` (no key). POST `/api/v1/devices/{id}/viewer` manager only `{url,key?}`; omitted key preserves existing, explicit clear separately. URL allows loopback or RFC1918 IP HTTP or HTTPS host (no credentials/query/fragment). No server fetching; browser opens explicit user-configured address. GET `/api/v1/devices/{id}/viewer-key` same person visibility, human-only, no read-token access, audited. Keys are secrets, omitted from normal data/MCP/logs. Existing server secrets-at-rest policy applies; launch does not embed key in URL. Loopback labelled this computer; LAN access needs a reachable authenticated listener and appropriate firewall permissions, not automatic firewall changes.

## Tasks and checks

- [ ] Collector: compact upstream adapter, independent analytics-once/run path, acknowledged policy, source checkpoints only after successful upload, unavailable records on unsupported endpoint, no transcript queue changes. Fixture tests and native owner pilot.
- [ ] Store/API: schema, replay/replacement, dates, nulls, overlapping sources, visibility/device/read-token gates, deletion/expiry, viewer config and key access. Disposable DB and HTTP tests.
- [ ] Web: `#person/<encoded id>` and `#project/<encoded name>`, Projects nav, daily chart with category switch, provider/client/model split, project contributors, account cards keyed by provider+email, device URL and key controls. Loading/error/empty/missing coverage; desktop/mobile/reload browser tests.
- [ ] MCP: `usage_analytics` filters and native quota observations, no viewer keys. Test actual JSON-RPC→permissioned HTTP result.
- [ ] Release: full CI and independent review, server deployment, owner collector installation/run, live data and UI/MCP proof, update status/compatibility/private receipt. Full transcript migration remains a separate open gate.
