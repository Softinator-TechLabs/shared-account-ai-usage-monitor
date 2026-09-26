# People and project analytics

The workspace answers “which enrolled devices used which coding clients, in which recorded projects, on which days?” Full conversations remain a separate AgentsView deep dive. Employee token counts describe sources observed on enrolled devices; they do not verify historical prompt authors or quantify human productivity.

## Collection and meaning

`team-agent analytics-run --config /private/team-agent.json` is independent of the full transcript queue and quota worker. It enumerates official AgentsView sessions and reads each source's public `usage?breakdown=true` response. It projects timestamps, model and input/output/cache counters; prices, prompts and raw tool content are not part of this feed. The installed v0.44.0 API was probed before implementation; no native parsers were copied.

Daily buckets use event timestamps in Asia/Kolkata. Rolling 1/12/24/48-hour periods use the exact request-time interval, including its start and excluding its end. The last hour uses twelve five-minute buckets; the longer hour periods use one-hour buckets, anchored to the interval start. Chart labels use Asia/Kolkata. Missing timestamps are excluded and counted; missing counters stay null, known zero remains zero. Input, output, cache read and cache write are separate categories because provider conventions differ. Project names are recorded labels, not verified repository identities. A source on multiple devices counts once when copies agree. Conflicting content is excluded; different owners cannot each claim the same source. A newer capture replaces its old projection instead of adding its totals again. Unchanged reconciliation does not extend retention.

Checkpoints advance only after a durable server acknowledgement. The worker checks the current policy before reading/uploading and pauses on policy mismatch. It runs every five minutes and reconciles unchanged records hourly. A deleted source is suppressed locally and cannot be restored through replay. Aggregate coverage is visible; collected sources do not prove that all upstream history was available.

## Pages and accounts

- People names open `#person/<id>`: token categories, coding clients, models, projects, observed subscriptions and configured device viewers.
- Projects opens `#projects`; each label opens `#project/<label>` with period counters and observed device owners.
- Provider plus account identity separates Claude and Codex even when email matches. These account observations are not evidence that an individual historical prompt used that account.
- Native quota percentages remain account-wide. Employee/project quota percentages remain unknown. API-equivalent dollar prices are never treated as subscription spending.

The dashboard defaults to Last 24 hours, with Last hour, Last 12 hours and Last 48 hours ahead of the existing day periods. Token totals, chart axes and breakdown tables use K/M/B with up to two decimals; values below 1,000 remain plain. Hover and accessible labels retain exact counts, and the chart has an exact-count disclosure. This presentation never rounds the API data.

## AgentsView links

Managers can configure an origin and optional access key per enrolled device in Workspace. URL and key are separate controls; keys are never included in links, ordinary device JSON or MCP. The key is AES-256-GCM encrypted at rest, tied to device and origin. The API preserves a key when the `key` property is omitted; explicit `key: ""` clears it (the UI preserves a blank key on the same origin; clearing the URL removes both). Changing the origin clears its old key unless a new key is provided. Only human managers and the device's own person can reveal it; read/debug/device tokens cannot.

Set `VIEWER_ENCRYPTION_KEY` to a base64-encoded random 32-byte key before storing viewer credentials. Keep it in your deployment secret store and encrypted backup; losing it makes saved viewer keys unreadable. Do not rotate it without migrating existing ciphertext. Empty is allowed when viewer credential storage is unused.

A loopback URL opens only on that same computer. A LAN URL requires that device's AgentsView to listen on a reachable private interface and a separately configured firewall rule. Saving a URL does not enable sharing or prove reachability. The server never fetches these URLs. Do not expose an unauthenticated viewer publicly.

## Read-only MCP

`usage_analytics` accepts string `hours` (1, 12, 24, 48) or `days` (7, 14, 30, 90), plus `person`, `project`, `client`. Supplying both period types is rejected; omitting both retains the API default of 14 days. `/api/v1/analytics` uses the same query parameters. The response includes `granularity` (`5m`, `hour` or `day`) and `series`; hour-period points carry an ISO `timestamp`. Day responses retain the legacy `daily` array and date-only bounds. Hour responses use precise ISO bounds and leave `daily` empty. It returns the same permission-scoped period/category/group data as the dashboard, including missing-data and copy-conflict coverage. `list_quota_observations` reports timestamped account-wide readings. `list_device_viewers` returns visible origins and key availability, never credentials. Existing session/review tools remain available.

## Device rollout

Fresh macOS setup starts separate AgentsView, transcript, quota and analytics LaunchAgents. For an enrolled Mac, `team-setup upgrade --resources /absolute/App.app/Contents/Resources` preserves enrollment and queues while replacing the companion and adding analytics. Pause/resume covers all four services. Explicit uninstall removes LaunchAgents but retains private data; central revocation/deletion is separate.

Windows and Linux can run `analytics-run` under their existing per-user service manager, using the same acknowledged private config. Their native installer/startup behavior is not yet certified. Cross-compilation is not device testing.

The full transcript import size/backlog problem is independent and remains open. These analytics do not certify full centralized conversation availability, exact account/session binding, Claude/Antigravity quota collection, or central AgentsView device isolation.

- [Usage shares and subscription estimates](usage-estimates.md): Today, per-client model/effort/project pies, prompt and proposed-line counters, conditional quota allocation and explicit missing-data boundaries.
