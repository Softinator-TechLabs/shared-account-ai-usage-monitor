# Current support

This is an early open-source preview. Availability and proven behavior differ by platform.

| Area                                     | Current state                                                                                                                                  |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Mac menu app                             | Packaged for Apple silicon and Intel; owner-Mac installation verified. Ad-hoc signed, not notarized.                                           |
| Local Codex sessions                     | Observed on the owner Mac and uploaded to the central workspace.                                                                               |
| Local Claude Code / Antigravity sessions | Reader and collector compatibility depends on local source coverage; do not assume complete import.                                            |
| Codex account quota                      | Account-wide observations and reset times are available when the provider responds. Conditional project allocation remains evidence-dependent. |
| Claude / Antigravity quota               | Dedicated quota adapters remain pending.                                                                                                       |
| Windows / Ubuntu                         | CLI source and cross-builds exist; real employee-device installation and behavior remain unverified.                                           |
| Central deep reader                      | Session API archive is live. A broader upstream-reader migration is planned, not declared complete.                                            |

Use [release notes](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/releases) and the repository's [delivery status](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/status.md) for newer evidence. “Supported” should mean observed on an actual device, not merely compiled in CI.
