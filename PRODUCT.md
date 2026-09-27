# Product

<!-- impeccable:product-schema 1 -->

## Platform

Web workspace, public website and documentation; local macOS menu app and Windows/Linux collectors.

## Stack

Go and PostgreSQL for the private workspace, native/local collectors, and a separately deployed static public website with VitePress documentation. The owner delegated implementation choices in this conversation.

## Users

Small teams using native subscription-based Codex, Claude Code and Antigravity clients across shared accounts, projects and devices. Owners, PMs and employees need to understand observed AI activity and review work without treating token counts as productivity scores.

## Product Purpose

Connect device-observed sessions, projects, models, prompts and account-quota readings in one permissioned workspace. Make coaching and work discussion possible using the actual session evidence. Give visitors a clear public explanation, installation instructions, limitations and open-source code without exposing private telemetry.

## Capabilities and Constraints

The current macOS menu app installs background collection and can open its local AgentsView reader. Windows and Ubuntu use documented CLI setup; their employee-device behavior is not yet verified. Account quota is an account-wide observation. A project or employee share is displayed only when captured evidence supports a conditional allocation, with unknown and missing coverage visible. Model and project charts describe captured activity, not subscription allowance or payroll productivity. Public illustrations use synthetic examples. Prompt content and real employee identities never enter the public website repository.

The central app is at `usage.softinator.org`; the independent public website and docs are at `usage.softinator.ai`. The public repository is `Softinator-TechLabs/shared-account-ai-usage-monitor`.

## Brand Commitments

The product name is Shared Account AI Usage Monitor. The public website should be lively, modern and concise while the private workspace stays focused on evidence. Feedbacks is a quality and information-architecture reference, not a visual template. Attribution for independent upstream components belongs in notices and integration documentation, not persistent marketing footers.

## Product Principles

- Explain what is observed, estimated and unknown.
- Keep native subscription clients native and self-hosting possible.
- Preserve user-selected collection and visibility policies.
- Make installation and documentation easy to find.
- Separate source, tests, packaged app, installed device and live deployment evidence.
