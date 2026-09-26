// Presentation only: retained source evidence is never rewritten or re-attributed.
export function readableContent(content) {
  if (typeof content !== "string") return "";
  let value;
  try {
    value = JSON.parse(content);
  } catch {
    return content;
  }
  const text = (v, depth = 0) => {
    if (depth > 5) return null;
    if (typeof v === "string") return v;
    if (Array.isArray(v)) {
      const parts = v.map((x) => text(x, depth + 1));
      return parts.every((x) => x !== null) ? parts.join("\n\n") : null;
    }
    if (!v || typeof v !== "object") return null;
    if (
      typeof v.text === "string" &&
      ["text", "input_text", "output_text"].includes(v.type)
    )
      return v.text;
    return null;
  };
  return text(value) ?? content;
}
const recorded = (raw, field, flag) =>
  raw[flag] === true && Number.isSafeInteger(raw[field]) && raw[field] >= 0
    ? raw[field]
    : null;
export function describeSession(session) {
  const raw = session.raw || {};
  const messages = session.messages || [];
  const initial = messages.find(
    (m) => m.role === "user" && m.raw?.is_system !== true && m.content?.trim(),
  );
  const last = [...messages]
    .reverse()
    .find(
      (m) =>
        m.role === "assistant" && m.content?.trim() && !m.raw?.has_tool_use,
    );
  return {
    title:
      raw.display_name ||
      session.title ||
      session.project ||
      "Untitled session",
    initial,
    last,
    cwd: raw.cwd || "",
    branch: session.branch || raw.git_branch || raw.branch || "",
    outputTokens: recorded(
      raw,
      "total_output_tokens",
      "has_total_output_tokens",
    ),
    peakContext: recorded(
      raw,
      "peak_context_tokens",
      "has_peak_context_tokens",
    ),
    models: [
      ...new Set(
        messages
          .filter((m) => m.role === "assistant")
          .map((m) => m.model)
          .filter(Boolean),
      ),
    ],
    automated: raw.is_automated === true,
  };
}
export function sessionTools(session) {
  const nested = (session.messages || []).flatMap((m) =>
    (m.raw?.tool_calls || []).map((c) => ({ ...c, ordinal: m.ordinal })),
  );
  const seen = new Set();
  // Match copies between representations, preserving repeated id-less calls.
  return [nested, session.tool_calls || []].flatMap((calls) => {
    const occurrences = new Map();
    return calls.filter((c) => {
      if (!c || typeof c !== "object") return false;
      const identity = JSON.stringify([
        c.ordinal,
        c.call_index ?? null,
        c.tool_name,
        c.input_json,
      ]);
      const occurrence = occurrences.get(identity) || 0;
      occurrences.set(identity, occurrence + 1);
      const key = c.tool_use_id
        ? "id:" + c.tool_use_id
        : JSON.stringify([identity, occurrence]);
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  });
}
export function agentsViewURL(origin, source = "") {
  const url = new URL(origin);
  if (
    !["http:", "https:"].includes(url.protocol) ||
    !["127.0.0.1", "[::1]", "localhost"].includes(url.hostname) ||
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    url.pathname !== "/"
  )
    throw new Error(
      "Use a local address, such as http://127.0.0.1:8080, without a path or token.",
    );
  if (!source) return url.origin + "/";
  const split = source.indexOf(":");
  const path =
    split < 0
      ? encodeURIComponent(source)
      : encodeURIComponent(source.slice(0, split)) +
        "/" +
        encodeURIComponent(source.slice(split + 1));
  return url.origin + "/sessions/" + path;
}
