import {
  describeSession,
  agentsViewURL,
  readableContent,
  sessionTools,
} from "./session-data.js";
import {
  element as node,
  contentPreview,
  copyButton,
  renderContent,
} from "./reading.js";
import { workEvidence } from "./work-summary.js";
const count = (n) =>
  n === null ? "Not recorded" : new Intl.NumberFormat("en-IN").format(n);
export function localViewerControl(source = "") {
  const details = node("details", undefined, "local-viewer");
  details.append(node("summary", "Open in AgentsView"));
  const label = node("label", "AgentsView address on this computer");
  const input = node("input");
  input.type = "url";
  input.placeholder = "http://127.0.0.1:8080";
  try {
    input.value = localStorage.getItem("local-agentsview-origin") || "";
  } catch {}
  label.append(input);
  const note = node(
    "p",
    "Opens the local viewer on the source computer. Use the Mac app’s Open AgentsView button to find its address. Local access tokens stay on that computer.",
    "muted",
  );
  const link = node("a", "Open local session", "button-link");
  link.target = "_blank";
  link.rel = "noopener noreferrer";
  const error = node("p", "", "notice");
  error.setAttribute("role", "status");
  function update() {
    try {
      link.href = agentsViewURL(input.value, source);
      link.removeAttribute("aria-disabled");
      error.textContent = "";
    } catch {
      link.removeAttribute("href");
      link.setAttribute("aria-disabled", "true");
    }
  }
  input.addEventListener("input", update);
  link.onclick = (e) => {
    try {
      link.href = agentsViewURL(input.value, source);
      try {
        localStorage.setItem(
          "local-agentsview-origin",
          new URL(input.value).origin,
        );
      } catch {}
    } catch (err) {
      e.preventDefault();
      error.textContent = err.message;
      input.focus();
    }
  };
  update();
  details.append(note, label, link, error);
  return details;
}
export function sessionActions(s) {
  const bar = node("div", undefined, "session-actions");
  bar.append(
    copyButton(
      "Copy session link",
      `${location.origin}${location.pathname}#session/${encodeURIComponent(s.id)}`,
    ),
    localViewerControl(s.source_ref),
  );
  return bar;
}
export function renderOverview(s, discuss) {
  const d = describeSession(s),
    evidence = workEvidence(s);
  const root = node("section", undefined, "work-summary");
  const facts = node("dl", undefined, "session-context");
  for (const [label, value] of [
    ["Project", s.project || "Not recorded"],
    ["Branch", d.branch || "Not recorded in this capture"],
    ["Repository / workspace", d.cwd || "Not recorded"],
    ["Model", d.models.join(", ") || "Not recorded"],
    ["Output tokens", count(d.outputTokens)],
    ["Peak context tokens", count(d.peakContext)],
  ]) {
    const pair = node("div");
    pair.append(node("dt", label), node("dd", value));
    facts.append(pair);
  }
  if (d.initial) {
    const initial = contentPreview(
      "Initial prompt",
      d.initial.content,
      "initial-prompt",
    );
    const review = node("button", "Discuss initial prompt", "text-button");
    review.onclick = () => discuss(d.initial.ordinal);
    initial.append(review);
    root.append(initial);
  } else
    root.append(
      node(
        "p",
        "No user prompt was captured. Check the original session in AgentsView.",
        "missing-evidence",
      ),
    );
  root.append(
    facts,
    node(
      "p",
      "Output is generated text; peak context is the largest recorded context window. Total input and subscription-limit share are not supplied by this capture.",
      "token-note",
    ),
  );
  if (d.last)
    root.append(
      contentPreview("Latest response", d.last.content, "latest-response"),
    );
  const files = node("details", undefined, "files-summary");
  files.append(
    node(
      "summary",
      evidence.files.length
        ? `${evidence.files.length} ${evidence.files.length === 1 ? "file" : "files"} with recorded edit operations`
        : "Files · no structured edit evidence",
    ),
  );
  files.append(
    node(
      "p",
      "Recorded tool inputs describe intended edits. They do not verify successful writes, a Git commit or changed-line counts.",
      "muted",
    ),
  );
  if (evidence.files.length) {
    const table = node("table", undefined, "work-files");
    const head = node("tr");
    for (const v of ["File", "Edits", "Tool"]) head.append(node("th", v));
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
    files.append(scroll);
  } else
    files.append(
      node(
        "p",
        "No supported edit input is available here. This does not mean no files changed.",
      ),
    );
  root.append(files);
  return root;
}
export function renderToolActivity(message) {
  const calls = sessionTools({ messages: [message] });
  if (!calls.length) return null;
  const group = node("details", undefined, "tool-activity");
  group.append(node("summary", `${calls.length} tool actions`));
  group.addEventListener("toggle", () => {
    if (!group.open || group.children.length > 1) return;
    for (const call of calls) {
      const item = node("details");
      item.append(node("summary", call.tool_name || "Tool"));
      item.addEventListener("toggle", () => {
        if (!item.open || item.children.length > 1) return;
        const input =
          typeof call.input_json === "string"
            ? call.input_json
            : JSON.stringify(call.input_json);
        item.append(
          node("h4", "Input"),
          renderContent(input || "No input recorded"),
        );
        if (call.result_content)
          item.append(node("h4", "Result"), renderContent(call.result_content));
      });
      group.append(item);
    }
  });
  return group;
}
