# Native subscription observations

The companion's independent `quota-run` worker reads the configured Codex profile every 60 seconds. It calls the documented local App Server account/rate-limit methods, starts no model turn and performs no login, logout or reset-credit redemption. OAuth credentials remain on the device. It does not enumerate Cockpit credentials.

The Accounts page displays observed sign-in email/plan, quota windows, last-read freshness, provider-reported next reset, recent history and collecting device/profile. Shared readings are grouped by observed email, not added together. Email is the identity evidence currently available; it is not a verified provider workspace/account UUID. Missing windows and percentages remain unknown. Errors preserve the last successful reading and show a failed/stale state. A scheduled reset is not proof that a reset happened.

This is current-profile evidence, **not historical session-account attribution**. Concurrent IDE/Cockpit instances need explicit profile configuration and separate runtime attribution evidence. Claude and Antigravity quotas are not implemented. Per-project quota shares are not inferred from tokens.

## Device setup

New Mac installations include the quota service and Pause/Resume controls. The default detector checks an installed Codex executable and the current `CODEX_HOME` (otherwise `~/.codex`). A missing executable produces no fake observation. Windows/Linux operators can run `team-agent quota-run --config <private-config>` with explicit native executable paths under their OS service manager; native startup on those systems remains unverified.

To monitor explicit profiles, add `codex_profiles` to the private enrollment config. Paths are machine-local; never commit this file. Synthetic shape:

```json
{
  "codex_profiles": [
    {"label": "office-profile", "executable": "/absolute/path/to/codex", "home": "/absolute/path/to/codex-profile"}
  ]
}
```

Use `team-agent quota-once --config <private-config>` for one read/upload. The quota queue uses a separate `.quota.queue` directory and a 16 MiB capacity, so failed transcript backfill cannot strand quota updates. Exact replay is idempotent; conflicting event IDs are rejected. Policy changes/revocation stop uploads. During a genuine central outage, capture uses the last acknowledged policy and queues observations; explicit denial stops capture. Before acknowledging a new policy, pause both collectors and deliver or explicitly discard both queues. `status` reports both. `discard-quota-queue --confirm` irreversibly discards pending quota observations, which cannot be recreated; it does not delete transcript checkpoints. Resume after acknowledgement. Preserve pending observations; do not delete the queue to hide an error.

An already installed older Mac app needs an updated bundle/service before these observations appear; a web deployment alone cannot upgrade its collector. Enrollment remains attached to the existing device—do not mint a new employee identity for an upgrade.

[Protocol source](https://learn.chatgpt.com/docs/app-server), checked 2026-09-26. Source/fixture tests do not certify all provider profiles or OS devices.
