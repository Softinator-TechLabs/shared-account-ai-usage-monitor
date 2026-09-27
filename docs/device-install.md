# Device companion

For macOS, start with **People → Connect device**. Download the Apple-silicon or Intel app, then open its private `.aiusage` connection file. The app shows policy and installs the required services. See [onboarding details and preview limitations](device-and-activity.md). The CLI below remains available for advanced use and Windows/Ubuntu.


1. Install the pinned official AgentsView release separately; verify its published checksum. Configure only the native profile roots agreed for collection. Bind its daemon to loopback and require authentication. For full available tool content, use `archive_content = "full"` and `result_content_blocked_categories = []`. Upstream source omissions remain explicit limitations.
2. Build/download the matching `team-agent` binary. Keep it, its configuration and queue under the employee's private OS account. POSIX permissions are set to 0700/0600; on Windows restrict the directory ACL to that user with the normal Windows security settings. The app is visible and has an explicit status command.
3. Owner creates the member's device invitation in Workspace and privately provides the downloaded file. Employee reads its policy. Enroll with the exact acknowledged version:

```
team-agent enroll --server https://telemetry.example.com \
 --config /private/team-agent.json --invitation-file /private/device-invitation.json \
 --device alice-laptop --ack-version 1 \
 --upstream http://127.0.0.1:8080 --upstream-token-file /private/agentsview-token
team-agent once --config /private/team-agent.json
team-agent status --config /private/team-agent.json
team-agent run --config /private/team-agent.json
```

`run` delivers every 15 seconds and checks upstream metadata every 15 minutes. Unchanged sources with a native transcript revision use a durable local checkpoint. `once` is useful for first verification. During initial backfill it reports device presence before enumeration and after delivered sources. The companion does not read native login credential files. Full/no-redaction prompt content may itself contain secrets. Historical account identity stays unknown; manage declared assignments centrally. Per-device `declared_account` configuration is rejected because it would backfill unsupported account claims.

## Policy changes and recovery

Use `team-agent policy --config …` to inspect the current policy. Collection stops when a newer reachable policy is observed. Delivery requires an exact acknowledged version. Offline collection uses the last acknowledged policy, queues locally, and never silently re-labels old content. Review/discard an incompatible queue explicitly using `discard-queue --confirm`; then `ack --ack-version N`. Original upstream data can be recollected under the new policy if it still exists.

Queue full, malformed source, missing content, expired/revoked token and network errors are visible failures. Nothing unacknowledged is evicted. A `.lock` directory prevents duplicate writers. After a crash, verify no companion process is running before removing that stale lock. Back up upstream source history independently; a finite queue is not a backup.

## Startup and removal

Run the same foreground command with absolute paths under your user's OS service manager: a macOS LaunchAgent (RunAtLoad/KeepAlive), Ubuntu user systemd unit (Restart=on-failure), or Windows Task Scheduler logon task (restart on failure, user's identity). Capture stderr into an access-restricted log. Do not run as root/admin or share one machine's config across people. Service startup/sleep/wake behavior on each OS still needs a real-device canary; cross-compilation alone is not that proof.

To uninstall, stop/remove the service in that manager, revoke the employee/device access, then explicitly decide whether to retain or remove the local queue/config. No silent removal is performed by the binary. Central source deletion/retention is separate from removing a device.

## Mac background status (preview 0.6.0)

The native app shows its own version/build, four LaunchAgent states, last successful analytics/quota cycle, the latest acknowledged analytics upload during backfill, and one state-aware Pause/Resume control. Refresh runs automatically every ten seconds while the window is open. `team-setup status --json` returns the same local health evidence without credentials or transcript/log content. Older collectors with no receipt show “no successful sync recorded”; a running process is not a successful sync. Failed attempts retain the last successful timestamp.

You can close or quit the setup app. Enabled per-user LaunchAgents continue while that Mac account is logged in and the computer is awake. Pausing stops the local reader and collectors; resume explicitly restarts them. Sleeping/shutdown/logout stops collection until the Mac is active again. Previously uploaded history remains visible centrally even with the Mac off; local AgentsView links require the local service and a reachable device. The status display does not change firewall permissions or the collection policy.
