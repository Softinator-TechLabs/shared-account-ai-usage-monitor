import { readableContent } from "./session-data.js";
export function element(tag, text, cls) {
  const n = document.createElement(tag);
  if (text !== undefined) n.textContent = text;
  if (cls) n.className = cls;
  return n;
}
// Clipboard denial is recoverable without losing the exact URL/prompt.
export function copyButton(label, value) {
  const group = element("span", undefined, "copy-control");
  const button = element("button", label, "text-button");
  button.type = "button";
  group.append(button);
  button.onclick = async () => {
    group.querySelector(".copy-fallback")?.remove();
    try {
      await navigator.clipboard.writeText(value);
      button.textContent = "Copied";
      button.setAttribute("aria-label", label + ": copied");
    } catch {
      const wrap = element("span", undefined, "copy-fallback");
      const help = element("span", "Select and copy:", "muted");
      const field = element("textarea");
      field.value = value;
      field.readOnly = true;
      field.setAttribute("aria-label", label + " value");
      wrap.append(help, field);
      group.append(wrap);
      field.focus();
      field.select();
    }
  };
  return group;
}
function inline(target, text) {
  // All content is text nodes. Only explicit HTTP(S) links become anchors.
  const re = /\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)|(https?:\/\/[^\s<>"`]+)/g;
  let end = 0,
    count = 0;
  for (const m of text.matchAll(re)) {
    if (++count > 500) break;
    let href = m[2] || m[3],
      suffix = "";
    if (!m[2]) {
      const trimmed = href.replace(/[.,;!?)\]]+$/, "");
      suffix = href.slice(trimmed.length);
      href = trimmed;
    }
    try {
      const u = new URL(href);
      if (!["http:", "https:"].includes(u.protocol) || u.username || u.password)
        continue;
    } catch {
      continue;
    }
    target.append(document.createTextNode(text.slice(end, m.index)));
    const link = element("a", m[1] || href);
    link.href = href;
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    target.append(
      link,
      copyButton("Copy URL", href),
      document.createTextNode(suffix),
    );
    end = m.index + m[0].length;
  }
  target.append(document.createTextNode(text.slice(end)));
}
export function renderContent(content, cls = "readable-content") {
  const root = element("div", undefined, cls);
  const text = readableContent(content);
  try {
    const data = JSON.parse(text);
    if (data && typeof data === "object" && !Array.isArray(data)) {
      const list = element("dl", undefined, "structured-content");
      for (const [key, value] of Object.entries(data)) {
        const pair = element("div");
        pair.append(element("dt", key.replaceAll("_", " ")));
        const body = element("dd");
        if (value !== null && typeof value === "object") {
          const more = element("details");
          more.append(
            element("summary", "View details"),
            element("pre", JSON.stringify(value, null, 2)),
          );
          body.append(more);
        } else inline(body, value === null ? "Not recorded" : String(value));
        pair.append(body);
        list.append(pair);
      }
      root.append(list);
      return root;
    }
  } catch {}
  for (const block of text.split(/\n\s*\n/)) {
    if (block.includes("```")) {
      // Code stays selectable text; no HTML or scripts are interpreted.
      const pre = element("pre", block);
      pre.className = "code-content";
      root.append(pre);
      continue;
    }
    const h = block.match(/^#{1,6}\s+([^\n]+)$/);
    const p = element(h ? "h4" : "p");
    inline(p, h ? h[1] : block);
    root.append(p);
  }
  return root;
}
export function contentPreview(label, content, cls = "") {
  const root = element("section", undefined, "reading-section " + cls);
  const head = element("div", undefined, "reading-heading");
  head.append(
    element("h3", label),
    copyButton("Copy " + label.toLowerCase(), content),
  );
  root.append(head);
  const text = readableContent(content);
  if (text.length <= 1000) root.append(renderContent(text));
  else {
    root.append(renderContent(text.slice(0, 700) + "…"));
    const more = element("details", undefined, "full-message");
    more.append(element("summary", "Read full " + label.toLowerCase()));
    more.addEventListener("toggle", () => {
      if (more.open && more.children.length === 1)
        more.append(renderContent(content, "message-content readable-content"));
    });
    root.append(more);
  }
  return root;
}
