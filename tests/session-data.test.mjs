import test from "node:test";
import assert from "node:assert/strict";
import {
  describeSession,
  readableContent,
  sessionTools,
  agentsViewURL,
} from "../internal/web/assets/session-data.js";

test("overview exposes native prompt and token semantics without invented totals", () => {
  const s = {
    project: "synthetic-project",
    raw: {
      display_name: "Fix synthetic links",
      cwd: "/work/synthetic",
      git_branch: "fix/links",
      total_output_tokens: 140,
      has_total_output_tokens: true,
      peak_context_tokens: 900,
      has_peak_context_tokens: true,
    },
    messages: [
      { ordinal: 0, role: "user", content: "Links ko fix karo." },
      {
        ordinal: 1,
        role: "assistant",
        content: "Done.",
        model: "synthetic-model",
      },
    ],
  };
  const d = describeSession(s);
  assert.equal(d.title, "Fix synthetic links");
  assert.equal(d.initial.content, "Links ko fix karo.");
  assert.equal(d.branch, "fix/links");
  assert.equal(d.outputTokens, 140);
  assert.equal(d.peakContext, 900);
  assert.deepEqual(d.models, ["synthetic-model"]);
  assert.equal(
    describeSession({
      raw: { total_output_tokens: 0, has_total_output_tokens: false },
      messages: [],
    }).outputTokens,
    null,
  );
  assert.equal(
    describeSession({
      raw: { total_output_tokens: 0, has_total_output_tokens: true },
      messages: [],
    }).outputTokens,
    0,
  );
});
test("readable envelopes preserve all text and do not execute embedded markup", () => {
  assert.equal(
    readableContent(
      '[{"type":"text","text":"Hinglish prompt"},{"type":"text","text":"<script>bad()</script>"}]',
    ),
    "Hinglish prompt\n\n<script>bad()</script>",
  );
  assert.equal(
    readableContent('{"arbitrary":"unknown"}'),
    '{"arbitrary":"unknown"}',
  );
});
test("nested tool evidence joins older captures without counting copies twice", () => {
  const c = {
    tool_use_id: "synthetic-call",
    tool_name: "Edit",
    input_json: '{"file_path":"/work/a.ts"}',
  };
  const calls = sessionTools({
    tool_calls: [{ ...c, ordinal: 3 }],
    messages: [{ ordinal: 3, raw: { tool_calls: [c, c] } }],
  });
  assert.equal(calls.length, 1);
  assert.equal(calls[0].ordinal, 3);
});
test("AgentsView links stay local, carry no credentials and preserve opaque native IDs", () => {
  assert.equal(
    agentsViewURL("http://127.0.0.1:18087", "codex:some/opaque?x"),
    "http://127.0.0.1:18087/sessions/codex/some%2Fopaque%3Fx",
  );
  for (const u of [
    "https://evil.example",
    "http://127.0.0.1.evil.example",
    "http://user:secret@127.0.0.1:8080",
    "javascript:alert(1)",
    "http://127.0.0.1:8080/?token=x",
  ])
    assert.throws(() => agentsViewURL(u, "id"));
});
