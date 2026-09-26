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
  for (const call of session.tool_calls || []) {
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
export function renderWorkSummary(session, node) {
  const evidence = workEvidence(session);
  const root = node("section", undefined, "work-summary");
  root.append(node("h3", "Work summary"));
  const facts = node("dl", undefined, "work-context");
  for (const [label, value] of [
    ["Project", session.project || "Not recorded"],
    ["Repository / workspace", evidence.cwd || "Not recorded"],
    ["Branch", evidence.branch || "Not recorded"],
  ]) {
    const pair = node("div");
    pair.append(node("dt", label), node("dd", value));
    facts.append(pair);
  }
  root.append(facts);
  if (evidence.files.length) {
    root.append(
      node(
        "h4",
        `${evidence.files.length} files with recorded edit operations`,
      ),
      node(
        "p",
        "Tool inputs show intended edits. Successful writes, committed changes and line counts are not verified.",
        "muted",
      ),
    );
    const table = node("table", undefined, "work-files");
    const head = node("tr");
    for (const x of ["File", "Operations", "Tool"]) head.append(node("th", x));
    table.append(head);
    for (const f of evidence.files) {
      const row = node("tr");
      row.append(
        node("td", f.path),
        node("td", String(f.operations)),
        node("td", f.tools.join(", ")),
      );
      table.append(row);
    }
    const scroll = node("div", undefined, "file-table-scroll");
    scroll.append(table);
    root.append(scroll);
  } else {
    root.append(
      node(
        "p",
        "File changes are not verified for this capture.",
        "missing-evidence",
      ),
      node(
        "p",
        session.tool_coverage
          ? "No supported structured edit input is recorded here. This does not prove that no files changed."
          : "This capture does not include structured tool evidence. New imports collect it when the client provides it.",
        "muted",
      ),
    );
  }
  const last = [...session.messages]
    .reverse()
    .find((m) => m.role === "assistant" && m.content);
  if (last) {
    root.append(
      node("h4", "Latest agent response"),
      node(
        "p",
        last.content.slice(0, 280) + (last.content.length > 280 ? "…" : ""),
        "result-excerpt",
      ),
    );
  }
  return root;
}
