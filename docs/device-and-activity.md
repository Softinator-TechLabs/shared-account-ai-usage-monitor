# Device onboarding and activity overview

The People page replaces the decorative connection banner with period-selectable activity charts. Counts cover sessions started in the selected IST period, their prompt count, projects, clients and recorded assistant models. One longest capture per source is used; later short/offline imports do not multiply sessions. Copied native sources can still overlap. Unknown dates/models are explicit. These are activity counts, not productivity, subscription cost or quota shares.

Small derived summaries are written transactionally with ingestion. A bounded background job indexes existing captures; coverage shows pending captures. Permissions match session visibility and source deletion cascades to derived summaries. No transcripts are returned by the dashboard endpoint.

## Connect device

Connect device opens a three-step flow, without downloading anything automatically:

1. Download the Mac setup application for Apple silicon or Intel.
2. Download a private `.aiusage` connection file and open it with the app. Existing JSON invitations are supported with an explicit workspace address.
3. Review server, identity and collection policy, agree, and check the device heartbeat in the dashboard.

The app installs the pinned, checksum-verified official AgentsView release and the companion as per-user LaunchAgents. No administrator password or provider API key is needed. It preserves an already configured Mac instead of silently switching accounts. Its return screen offers workspace access and pause/resume. Initial indexing may take time. Native Windows and Ubuntu installers are not advertised; their CLI guide remains available.

Preview Mac builds have ad-hoc integrity signatures, **not Apple Developer ID signatures or notarization**. macOS may require approval in Privacy & Security. No Gatekeeper bypass/disable command is provided. A Developer ID certificate and notarization credentials are required before claiming a frictionless public installer. Build with `python3 scripts/build_macos_app.py --arch arm64 --output .local/mac-arm64` (or amd64) on macOS with Go and Xcode command-line tools. Dependency notices are included in each bundle.

## Compact session review

A session opens with project, recorded workspace/repository path, branch and a file-evidence summary. Full conversation and export controls are collapsed. Messages have bounded previews and individually expandable content in a scrollable transcript pane. Discussion still anchors to immutable message ordinals; full content remains available through expansion/export.

Structured edit/write/patch tool inputs can identify intended file edits. They **do not prove successful writes, committed line changes or authorship**. No shell command or prose claim is treated as a confirmed file change. The pinned upstream tool-call endpoint exposes inputs and result length, not a verified Git diff. Therefore missing evidence is shown as unavailable, never zero work. Historical captures without tool evidence remain readable; the adapter checkpoint version forces a new capture on upgraded collectors.

Acceptance checks: aggregate visibility/revision/deletion/backfill integration tests; retained tool-input/incomplete-list fixtures; secret/metadata policy tests; browser chart, download, compact reader and review flows; macOS bundle signatures/builds; live route and real source checks. Build proof is distinct from clean-machine or reboot/sleep/wake proof.
