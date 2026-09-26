import { sessionTools } from "./session-data.js";
// Read only structured edit inputs. They record intent, not successful filesystem/Git changes.
export function workEvidence(session) {
  const raw = session.raw || {};
  const cwd = typeof raw.cwd === "string" ? raw.cwd : "";
  const files = new Map();
  const add = (path, tool, ordinal) => {
    if (typeof path !== "string" || !path || path.length > 4096) return;
    const relative =
      cwd && path.startsWith(cwd + "/") ? path.slice(cwd.length + 1) : path;
    const f = files.get(relative) || {
      path: relative,
      operations: 0,
      tools: new Set(),
      ordinal,
    };
    f.operations++;
    f.tools.add(tool);
    files.set(relative, f);
  };
  for (const call of sessionTools(session)) {
    const name = String(call.tool_name || "");
    const short = name.split(".").pop().toLowerCase();
    if (
      ![
        "edit",
        "write",
        "multiedit",
        "apply_patch",
        "str_replace_editor",
        "str_replace",
      ].includes(short)
    )
      continue;
    let input = call.input_json;
    try {
      input = JSON.parse(input);
    } catch {
      if (short !== "apply_patch") continue;
      input = { patch: input };
    }
    if (!input || typeof input !== "object") continue;
    if (
      short === "str_replace_editor" &&
      !["create", "str_replace", "insert", "undo_edit"].includes(input.command)
    )
      continue;
    if (short === "apply_patch") {
      const patch = input.patch || input.input || "";
      if (typeof patch !== "string") continue;
      for (const match of patch.matchAll(
        /^\*\*\* (?:Update|Add|Delete|Move to)(?: File)?: (.+)$/gm,
      ))
        add(match[1].trim(), name, call.ordinal);
    } else {
      add(
        input.file_path || input.path || input.target_file,
        name,
        call.ordinal,
      );
    }
  }
  return {
    cwd,
    branch: session.branch || raw.git_branch || raw.branch || "",
    files: [...files.values()].map((f) => ({ ...f, tools: [...f.tools] })),
    coverage: session.tool_coverage || "Not collected in this capture",
  };
}
