# Contributing

Read AGENTS.md and docs/index.md. Open changes against the smallest boundary that owns the problem. Add a failing synthetic fixture or invariant test first, then implement and run the relevant checks. No production transcript, private account profile or authentication blob belongs in an issue or PR.

Go code uses gofmt and go vet. Keep dependencies isolated in this module and lock exact releases. Browser JavaScript is an ES module embedded in the server; Node is needed only for tests. Prefer small explicit functions and standard-library facilities over adding frameworks for one feature.

Integration tests require a disposable PostgreSQL DSN. Run the full checks in docs/verification.md before release. Mark unsupported OS/client combinations honestly. A cross-build cannot certify a native client parser or employee install.

When updating AgentsView, change the compatibility manifest, verify official archive hashes, run synthetic fidelity/replay tests and one-device canary, then release our compatibility update. Do not vendor broad parser forks. Propose general fixes upstream with license notices and synthetic reproductions only.

Reviewers should check authorization parity, immutable anchors, current-policy acknowledgement, queue replay/data loss, source attribution, deletion of derived quotes, prompt injection boundaries, synthetic fixture provenance, no secret-bearing logs, and usable keyboard/mobile states.
