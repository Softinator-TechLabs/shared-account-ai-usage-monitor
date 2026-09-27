# Getting started

Shared Account AI Usage Monitor has two pieces: a self-hosted team workspace and a collector on each employee device. It observes native coding-client sessions; it does not replace Codex, Claude Code or Antigravity logins, and it does not proxy inference through an API.

## 1. Host or join a workspace

For a new team, [self-host the server and database](/guide/self-host) under a domain you control. The owner creates their password with the one-use setup link, adds people in **People**, and shares each person's setup link privately. If your team already hosts a workspace, ask its owner for your workspace URL and invitation. This public site does not contain team telemetry or offer a shared hosted tenant.

## 2. Install your device collector

- **Apple silicon Mac:** [download the DMG](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/releases/download/v0.7.0-preview/AI-Usage-Monitor-mac-arm64.dmg).
- **Intel Mac:** [download the Intel DMG](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/releases/download/v0.7.0-preview/AI-Usage-Monitor-mac-amd64.dmg).
- **Windows and Ubuntu:** build/use the CLI collector from the [public source](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor) and follow [device installation](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/device-install.md). Native installer and employee-machine verification are still pending.

The Mac DMGs are **ad-hoc signed preview builds**, not Apple-notarized releases. [See all releases](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/releases) and verify the published SHA-256 checksums before installing.

## 3. Connect and explore

Follow [Connect a device](/guide/connect-device). Once uploaded sessions appear, use People and Projects for aggregate views, Sessions for the recorded work, and Accounts for provider-qualified quota observations. Read [Usage and limits](/guide/usage-and-limits) before interpreting percentages.
