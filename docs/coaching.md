# Coaching with your native agent

The dashboard supports comments and separate 1–5 prompt/work ratings. A work rating requires evidence text (prefer repo/PR/SHA/file and test/deployment receipts). A concise Hinglish prompt can be excellent; writing more or spending more tokens is not a quality criterion.

In Workspace, download an eight-hour read token into a private file. Build `go build -o team-mcp ./cmd/team-mcp` and configure your agent's stdio MCP command:

```json
{"command":"/absolute/path/team-mcp","args":["--server","https://telemetry.example.com","--token-file","/private/path/team-access-token.txt"]}
```

Tools: search_sessions, get_session, get_reviews, list_accounts. The HTTP server enforces the token's read-only scope even if a client attempts a different method. Data is returned under the signed-in member's access, including self/managers visibility.

Ask: “Aaj ke sessions mein kis prompt ke acceptance criteria unclear the? Session/message evidence cite karo. Prompt clarity aur delivered work alag assess karo; incomplete coverage aur unknown model/account clearly likho.” Collect actual repository PR/check evidence separately before drawing conclusions about delivery.

A human can download coaching context from a session, review it using their existing native subscription agent, and submit the draft through a scoped analysis run. `POST /api/v1/sessions/{id}/analysis` with `{"ordinal":0}` returns a one-hour result token, full context and instructions. `POST /api/v1/analysis/result` with that token and `{"body":"draft..."}` accepts one draft. It cannot post human ratings, change policy, or attach results to another session. Policy changes, membership revocation, expiry and deletion invalidate outstanding runs.

This first release does not run an autonomous model service on Dokploy or obtain provider login secrets. External agent/model identity is declared and unverified. Native execution can remain in a subscribed desktop/CLI; [Codex non-interactive mode](https://learn.chatgpt.com/docs/non-interactive-mode) is one optional local execution route. Review untrusted transcripts without granting them write/network tools. The server has no paid inference API dependency.

## Optional local Codex review command

Use **Prepare native agent review** on a selected message. The downloaded request is private and expires after one hour. It includes a one-result capability; do not commit or paste that token into a model prompt. Then:

```
python3 scripts/run_review.py --request /private/private-review-request.json \
 --output /private/review-draft.txt --submit
```

This explicit command uses the existing local Codex ChatGPT login, clears API-key environment overrides, disables shell/web tools, ignores user configuration/rules, runs in an empty temporary directory with a read-only sandbox, and submits only the final draft if `--submit` is present. The capability is held by the script, not included in the model input. It does not log prompts or native stdout. Choose a model with `--model` if required. No background job or employee monitoring is enabled by downloading a request.

CLI argument construction and scoped central submission are tested; actual provider inference and model identity need a local subscribed-client canary. Other native agents can use the same context/result contract, but a Claude/Antigravity executor is not certified by this script.

## Seven-day debugging login

An owner can create a named debugging link in Workspace → Agent debugging links. This is a separate **read-only** browser/API credential, valid for seven days from issuance. It cannot change policy, create users, submit human ratings or create further credentials. Access remains subject to the issuing member's active status and current permissions. Audit entries identify the agent label and issuing person. Owners can revoke links; password reset and signing out with the link also revoke access. The URL fragment is exchanged only at the same-origin endpoint and immediately removed from browser history. Store the downloaded link privately, never in source control.
