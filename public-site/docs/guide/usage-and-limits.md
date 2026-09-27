# Usage and limits

The workspace joins several kinds of evidence. They answer different questions.

| Signal                  | What it means                                                                | What it cannot prove                                                |
| ----------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Captured session tokens | Tokens recorded in a coding-client session on an enrolled device             | A provider's subscription quota percentage or employee productivity |
| Project/model chart     | Share of captured activity in the selected period                            | Share of the provider's account allowance                           |
| Account quota reading   | Provider-reported account-wide usage and reset window, when available        | Which individual or project spent the entire change                 |
| Conditional allocation  | An estimate across a bounded account interval with matched captured activity | An exact bill, payroll measure or complete history                  |

When the account is shared, a provider reading belongs to **the account**, even if one employee's device observed it. A project share can be estimated only for intervals whose session evidence, account identity and timing are good enough. Missing sources, parallel users, unknown rates, resets and client format changes leave the estimate **unknown** or partial. An observed 3% account increase cannot be turned into a certain project-level 3% without those links.

The People and Projects pages expose rolling hour/day windows and recorded token categories. The account pages show the provider reading separately. Check collection time, period, coverage and provenance before comparing employees. Code volume, tokens and prompt fluency are coaching inputs, not a ranking or pay decision.

For the exact current calculation and exclusions, see [usage estimates](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/usage-estimates.md) and [native quota evidence](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/native-quota.md).
