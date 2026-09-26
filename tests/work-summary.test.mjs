import test from "node:test";
import assert from "node:assert/strict";
import { workEvidence } from "../internal/web/assets/work-summary.js";
test("only structured edit evidence counts, not mentions or shell commands", () => {
  const session = {
    raw: { cwd: "/repo" },
    messages: [{ content: "I changed secret.txt" }],
    tool_calls: [
      {
        tool_name: "Edit",
        ordinal: 2,
        input_json: JSON.stringify({ file_path: "/repo/src/a.ts" }),
      },
      {
        tool_name: "functions.apply_patch",
        ordinal: 3,
        input_json: JSON.stringify({
          patch:
            "*** Begin Patch\n*** Update File: src/a.ts\n+synthetic\n*** Add File: src/b.ts\n+synthetic\n*** End Patch",
        }),
      },
      {
        tool_name: "exec_command",
        input_json: JSON.stringify({ cmd: "echo changed >/repo/c.ts" }),
      },
    ],
  };
  const r = workEvidence(session);
  assert.equal(r.files.length, 2);
  assert.equal(r.files.find((f) => f.path === "src/a.ts").operations, 2);
  assert.equal(r.branch, "");
  assert.equal(workEvidence({}).files.length, 0);
});
