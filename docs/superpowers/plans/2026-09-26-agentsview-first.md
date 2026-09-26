# AgentsView-first Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reuse upstream central sessions/search/analytics while delivering reliable shared-account quota evidence and preserving team coaching.

**Architecture:** Official AgentsView remains independently pinned and unmodified. Its PostgreSQL transport/viewer must pass a synthetic private-network pilot and our authorization/policy gates before taking over the reader. Our companion/server retain people/device identity, observed provider bindings, quotas, review anchors and permissioned agent context.

**Tech Stack:** Existing Go/PostgreSQL/embedded UI; official AgentsView CLI; local Codex App Server JSON-RPC; existing synthetic Go/Python/browser checks.

**Spec:** [Approved design and evidence constraints](../../agentsview-first.md). Read [current status](../../status.md) before execution. This is a migration plan, not a completed release receipt.

## Global constraints

- Runtime content and credentials never enter Git or public assets.
- Initial central pilot is only for explicitly acknowledged `full/none/team` workspaces.
- Existing archive/queue and immutable review anchors remain intact until verified cutover.
- No upstream internal package/table dependency, parser fork or public PostgreSQL port.
- Keep native subscription clients; no model turn, account switch or reset-credit redemption for monitoring.
- No historical account/author inference, exact project quota claims or employee ranking from usage.
- macOS, Windows and Ubuntu runtime proof are separate; cross-build is insufficient.
- Use synthetic fixtures and an explicit disposable test database; never use production for regression tests.

## Review focus

1. Direct upstream/export/deep-link access bypassing workspace login — task 2 denial/revocation cases.
2. Policy changes or an overly broad PG writer exposing another device's history — tasks 1/2 role and policy gates.
3. Concurrent profiles, reordered snapshots and reset boundaries generating false attribution — tasks 3/4 account fixtures.
4. Large UTF-8 sessions, copied sources and index rebuilds losing content or comment anchors — tasks 1/5 fidelity cases.
5. Offline machines, deleted captures and rollback restoring forbidden content — tasks 5/6 recovery cases.

## Task 1 — Prove upstream central transport on synthetic data

**Files:** create `deploy/agentsview-pilot/README.md`, `deploy/agentsview-pilot/compose.yml`, `scripts/verify_agentsview_pg.py`, `compatibility/agentsview-central.json`; extend `tests/test_agentsview_compatibility.py` if present, otherwise create `tests/test_agentsview_central.py` for the probe. Do not modify production Compose yet.

**Interface:** verifier accepts `--agentsview-bin`, `--database-url-env`, `--receipt`; reads DSN from the named environment variable, never logs it. Receipt contains upstream version/checksum, test names, pass/fail, coverage and timestamps, never DSN/content. Manifest declares tested released capabilities; do not guess an official container image.

- [ ] Add failing probe assertions for two distinct synthetic hosts, repeated push idempotency, same basename across hosts, interrupted push/retry and fidelity hash of a synthetic UTF-8/tool session exceeding the old 256 MiB JSONB failure boundary.
- [ ] Run the probe against an isolated pinned binary/database; record baseline failures and actual released CLI flags.
- [ ] Package the private-network upstream push/serve pilot using a verified binary/checksum. Separate DB roles and schema ownership from the existing application.
- [ ] Probe minimum writer privileges: device revocation, cross-host read/write access and schema migration requirements. Fail the transport gate if devices cannot be adequately isolated; write a scoped-transport alternative before task 2, rather than granting broad production access.
- [ ] Run verifier; record hash equality, discovered omissions, peak resources and recovery. Passing CLI help alone cannot check this task off. Commit synthetic assets and safe receipt.

**Gate:** accepted transport/access evidence or explicit failed gate. No production routing change.

## Task 2 — Authenticated central viewer entry point

**Files:** create `internal/web/agentsview_proxy.go`, `internal/web/agentsview_proxy_test.go`; modify `internal/web/server.go`, `cmd/team-server/main.go`, `internal/web/assets/app.js`, `docs/deployment.md`. Extend pilot Compose only after task 1 passes.

**Interface:** `AGENTSVIEW_CENTRAL_URL` and server-side `AGENTSVIEW_CENTRAL_TOKEN_FILE` configure an optional viewer under `/agentsview/`; absent configuration keeps current behavior. Handler uses existing workspace authentication/roles and explicit policy check. Upstream supports base-path/public-origin only when confirmed in the pinned release.

- [ ] Add failing HTTP tests for anonymous, revoked, expired-debug-link, wrong-workspace and non-`full/none/team` requests; include deep links, API/search/export and all methods. Assert no upstream token in responses and no client-supplied upstream Authorization passthrough.
- [ ] Run `go test ./internal/web -run AgentsViewProxy -count=1` with the disposable DB; record expected failure.
- [ ] Implement gateway and an explicit “Open central AgentsView” link, restricted upstream read operations, trusted-origin handling and private upstream bind. Do not introduce a second login or promise per-session upstream ACLs.
- [ ] Integrate upload-service lifecycle with existing acknowledgement/revocation controls. Test that unacknowledged, metadata-only or self/managers policy cannot start this raw upstream sync path; policy changes stop it and cannot relabel queued content.
- [ ] Run HTTP tests and synthetic browser login → viewer → deep link/copy → logout/revocation flows. Verify upstream port unreachable externally. Commit receipt separately from live approval/cutover.

## Task 3 — Read-only Codex profile evidence adapter

**Files:** create `internal/quota/codex.go`, `internal/quota/codex_test.go`, `internal/quota/types.go`, `compatibility/fixtures/codex-account-quota.json`; extend `cmd/team-agent/main.go` and companion configuration using existing patterns.

**Interface:** `ReadCodexSnapshot(ctx context.Context, executable, profileHome string) (CodexSnapshot, error)` owns a bounded stdio child and always terminates it. `CodexSnapshot` holds observed time, source version, observed email/plan (optional), profile identity and `[]QuotaWindow`. `QuotaWindow` holds bucket ID, window name, nullable used percentage, duration minutes and reset Unix timestamp. No credentials in returned data. Session binding is absent unless separately proven.

- [ ] Build a fake JSON-RPC subprocess fixture. Assert initialize/initialized, `account/read` with `refreshToken:false`, `account/rateLimits/read`, notification handling, timeout/EOF cleanup; assert no login/turn/reset calls.
- [ ] Cover multiple buckets, absent windows, nullable values, newer-map precedence, malformed percentages, provider error, mid-read account change and two independently scoped profiles. Mid-read identity change rejects the ambiguous observation; missing evidence is not zero.
- [ ] Run `go test ./internal/quota -count=1` to capture failure, then implement adapter with explicit executable/profile selection. Notifications are evidence only within the observed process; do not label unrelated IDE sessions.
- [ ] Rerun with race detection; perform one owner-authorized real profile read without printing identity/tokens. Record command version, field availability and coverage only. Commit source and synthetic fixtures.

## Task 4 — Central quota timeline and account binding provenance

**Files:** create `internal/store/quota_observations.go`, `internal/store/quota_observations_test.go`, `internal/contracts/quota.go`, `internal/web/quota.go`, `internal/web/quota_test.go`; modify existing account store, companion upload flow and web account view. Follow existing store migration convention.

**Interface:** versioned observations include event ID, workspace/device/profile, account evidence reference, observed/received UTC times, source version, bucket/window/reset and nullable used percentage. Bindings include session/turn when known, interval, method and confidence. An assignment is a different record. Retry event ID is stable. Preserve device provenance while aggregating one account-window timeline.

- [ ] Add failing DB/HTTP tests: duplicate replay, simultaneous device readings, out-of-order/offline snapshots, disagreeing readings, reset change with observation gap, account switch and unknown session binding. A 40→48 example yields account delta 8 percentage points and no employee/project debit.
- [ ] Run focused quota tests using `TEST_DATABASE_URL`; record expected failure.
- [ ] Implement authenticated upload, append-only provenance and bounded permissioned queries. Unknown identity remains unresolved; stale/error readings do not erase last-good evidence. Never sum shared quota across devices or turn a scheduled reset into an observed reset.
- [ ] Show account used/remaining, next reset, last observed time, error/stale state and observed reset history. Session tokens/models remain separately labelled. Use Impeccable for the view and a synthetic browser test for contradictory/stale/unknown states.
- [ ] Run quota tests with race detection and browser flow; commit. Claude/Antigravity adapters remain explicitly unsupported until their own contracts pass.

## Task 5 — Review anchors, retention and permissioned agent access

**Files:** create `internal/agentsview/references.go`, `internal/agentsview/references_test.go`, `internal/store/source_links.go`, `internal/store/source_links_test.go`; modify `internal/store/review.go`, `internal/store/retention.go`, relevant MCP reads and `docs/coaching.md`.

**Interface:** link existing workspace/source/revision/ordinal to upstream instance/source plus verified content fingerprint. Matching failure returns unresolved, never a heuristic reassignment. Keep old reviewed captures accessible under their existing permissions. Resolve upstream text through tested public interfaces, not SQL table assumptions.

- [ ] Add failing cases for upstream ID change after rebuild, duplicate/forked content, moving session revisions and missing messages. Assert old comments/ratings still point at exact reviewed text.
- [ ] Test deletion and restore across both stores, offline re-upload, source tombstones, policy changes and MCP access parity. If released upstream lacks required deletion/isolation controls, keep affected workspaces on existing path and stop cutover.
- [ ] Implement reference mapping and joined read context; preserve existing review writes, separate ratings and explicit unverified agent drafts.
- [ ] Run store/adapter/MCP tests and a synthetic browser old-comment → exact capture → upstream source flow; compare content fingerprints. Commit only after integrity gates pass.

## Task 6 — Device pilot, canary and reversible cutover

**Files:** update `deploy/compose.yml`, `internal/macsetup/setup.go`, `docs/device-install.md`, `docs/verification.md`, `docs/status.md`, `compatibility/support-matrix.json`; add tested Windows/Linux service packaging under `deploy/` as needed.

- [ ] Run full documented regression suite and isolated backup → delete → restore rehearsal for both stores and deletion ledger.
- [ ] Verify macOS, Windows and Ubuntu install/start/sleep/wake/offline/retry/uninstall independently, including upstream process policy acknowledgement and revocation. Record unsupported platforms instead of claiming cross-build proves them.
- [ ] Back up existing archive, review records and queues; record sync ownership to prevent duplicate accounting. Canary one owner device and isolated central instance; measure integrity, search latency and memory before routing default session links.
- [ ] Verify authenticated production viewer, quota freshness, logout/revocation, copied permalinks, old review anchors and read-only agent access. Only then enable default central viewer. Retain original route/archive for rollback until retention and migration validation are complete.
- [ ] Rehearse rollback without deleting queues or recollecting tombstoned sessions; update exact tested versions and public status. Publish a release only with actual deployment/device receipts.

Optional LAN peer sharing is a separate later feature with scoped OS permissions and allowlisted destinations. It does not block central outbound sync. Automatic upgrades remain canary-gated; no recurring task is enabled by this plan.

## Plan review receipt

All design sections map to tasks 1–6; each review-focus failure has an owning test gate. Private operations/history remain in the private management repository. No task is checked merely because a source or command exists. This revision changes documentation and project direction only; runtime work and production migration remain pending.
