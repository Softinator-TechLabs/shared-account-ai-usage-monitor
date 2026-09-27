import { exactNumber } from "./number-format.js";

const el = (tag, text, cls) => {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = text;
  if (cls) node.className = cls;
  return node;
};
const known = (value) =>
  typeof value === "number" && Number.isFinite(value) && value >= 0;
const label = (value) => value || "Unknown";
const clientNames = new Map([
  ["codex", "Codex"],
  ["claude", "Claude"],
  ["antigravity", "Antigravity"],
]);
const clientName = (value) => clientNames.get(value) || label(value);
const colors = [
  "#305dd9",
  "#627891",
  "#976739",
  "#675ca0",
  "#377f84",
  "#a75168",
  "#626871",
];
const percent = (weight, total) =>
  known(weight) && total > 0
    ? `${exactNumber((weight / total) * 100)}%`
    : "Unknown";

function groups(rows, fields) {
  const result = new Map();
  for (const row of rows) {
    const key = JSON.stringify(fields.map((field) => row[field] || ""));
    if (!result.has(key))
      result.set(key, {
        labels: fields.map((field) => label(row[field])),
        weight: null,
        prompts: 0,
        generated_lines: 0,
      });
    const group = result.get(key);
    if (known(row.weight)) group.weight = (group.weight || 0) + row.weight;
    for (const field of ["prompts", "generated_lines"]) {
      group[field] =
        group[field] !== null && known(row[field])
          ? group[field] + row[field]
          : null;
    }
  }
  return [...result.values()].sort((a, b) => (b.weight || 0) - (a.weight || 0));
}

function pie(title, accessibleName, rows, total) {
  const figure = el("figure", undefined, "composition-chart");
  figure.append(el("figcaption", title));
  const content = el("div", undefined, "composition-chart-content");
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", "0 0 200 200");
  svg.setAttribute("class", "composition-pie");
  svg.setAttribute("role", "img");
  svg.setAttribute(
    "aria-label",
    `${accessibleName}: share of captured weighted usage. Values are listed in the adjacent legend.`,
  );
  const legend = el("ul", undefined, "composition-legend");
  let angle = -Math.PI / 2;
  const weighted = rows.filter((row) => known(row.weight) && row.weight > 0);
  weighted.forEach((row, index) => {
    const color = colors[index % colors.length];
    const next = angle + (row.weight / total) * Math.PI * 2;
    const slice = document.createElementNS(
      "http://www.w3.org/2000/svg",
      weighted.length === 1 ? "circle" : "path",
    );
    if (weighted.length === 1) {
      slice.setAttribute("cx", "100");
      slice.setAttribute("cy", "100");
      slice.setAttribute("r", "96");
    } else {
      slice.setAttribute(
        "d",
        `M 100 100 L ${100 + 96 * Math.cos(angle)} ${100 + 96 * Math.sin(angle)} A 96 96 0 ${next - angle > Math.PI ? 1 : 0} 1 ${100 + 96 * Math.cos(next)} ${100 + 96 * Math.sin(next)} Z`,
      );
    }
    slice.setAttribute("fill", color);
    slice.setAttribute("stroke", "var(--surface)");
    slice.setAttribute("stroke-width", "1.5");
    svg.append(slice);
    angle = next;
    const item = el("li");
    const swatch = el("span", undefined, "composition-swatch");
    swatch.style.backgroundColor = color;
    swatch.setAttribute("aria-hidden", "true");
    item.append(
      swatch,
      el("span", row.labels.join(" · ")),
      el("strong", percent(row.weight, total)),
    );
    legend.append(item);
  });
  content.append(svg, legend);
  figure.append(content);
  return figure;
}

function table(headers, rows, name) {
  const wrap = el("div", undefined, "usage-table-wrap");
  wrap.tabIndex = 0;
  wrap.setAttribute("role", "region");
  wrap.setAttribute("aria-label", name);
  const table = el("table", undefined, "usage-table composition-table");
  table.setAttribute("aria-label", name);
  const head = el("thead");
  const heading = el("tr");
  headers.forEach((text) => {
    const th = el("th", text);
    th.scope = "col";
    heading.append(th);
  });
  head.append(heading);
  const body = el("tbody");
  rows.forEach((values) => {
    const row = el("tr");
    values.forEach((value, index) => {
      const cell = el(index === 0 ? "th" : "td", value);
      if (index === 0) cell.scope = "row";
      row.append(cell);
    });
    body.append(row);
  });
  table.append(head, body);
  wrap.append(table);
  return wrap;
}

export function renderComposition(composition) {
  const section = el("section", undefined, "usage-section usage-composition");
  section.append(
    el("h2", "Usage breakdown"),
    el(
      "p",
      "Share of captured weighted usage · each client has its own total.",
      "chart-caption",
    ),
  );
  const byClient = new Map();
  for (const row of composition.rows || []) {
    const key = row.client || "";
    if (!byClient.has(key)) byClient.set(key, []);
    byClient.get(key).push(row);
  }
  if (!byClient.size)
    section.append(
      el("p", "No usage breakdown captured in this period.", "usage-empty"),
    );
  for (const [client, rows] of byClient) {
    const name = clientName(client);
    const group = el("article", undefined, "composition-client");
    group.append(el("h3", name));
    const total = rows.reduce(
      (sum, row) => sum + (known(row.weight) ? row.weight : 0),
      0,
    );
    const models = groups(rows, ["model", "effort"]);
    const projects = groups(rows, ["project"]);
    if (total > 0) {
      const charts = el("div", undefined, "composition-charts");
      charts.append(
        pie("Model & effort", `${name} model and effort`, models, total),
        pie("Recorded projects", `${name} recorded projects`, projects, total),
      );
      group.append(charts);
    } else
      group.append(
        el("p", "No weighted usage captured for this client.", "usage-empty"),
      );
    const excluded = rows.reduce(
      (sum, row) => sum + (row.unweighted_points || 0),
      0,
    );
    group.append(
      el(
        "p",
        `${exactNumber(excluded)} unweighted points excluded from shares.`,
        "chart-caption",
      ),
    );
    const detail = el("details", undefined, "composition-details");
    detail.append(
      el("summary", "Model and effort details"),
      table(
        ["Model", "Effort", "Weighted share", "Prompts", "Proposed lines"],
        models.map((row) => [
          ...row.labels,
          percent(row.weight, total),
          exactNumber(row.prompts),
          exactNumber(row.generated_lines),
        ]),
        `${name} model and effort details`,
      ),
    );
    group.append(detail);
    section.append(group);
  }
  const method = el("details", undefined, "composition-method");
  method.append(
    el("summary", "How these shares are estimated"),
    el(
      "p",
      "Codex uses standard-speed native credit proxies; Claude uses API rate proxies, including 5-minute cache-write rates. These are not dollars spent or subscription quota percentages. Clients are never combined. Effort is recorded metadata, with no added multiplier.",
    ),
    el(
      "p",
      "Shares cover only captured usage with known rates and counters. Unknown models, effort and projects stay Unknown. Recorded project names are not verified repository identities. Prompts count captured human messages; their model may come from the next assistant reply. Proposed lines count structured write/edit inputs, including unchanged context in replacement text, not accepted or delivered code. Missing activity counts stay Unknown.",
    ),
    el(
      "p",
      `${exactNumber(composition.activity_sources)} sources with activity · ${exactNumber(composition.activity_unavailable_sources)} unavailable · Rates: ${label(composition.rate_version)}`,
    ),
  );
  section.append(method);
  return section;
}

const points = (value) =>
  known(value) ? `${exactNumber(value)} pp` : "Unknown";
const when = (value) => {
  if (value === null || value === undefined || value === "") return "Unknown";
  const date = new Date(typeof value === "number" ? value * 1000 : value);
  if (!Number.isFinite(date.getTime())) return "Unknown";
  return date.toLocaleString("en-GB", {
    timeZone: "Asia/Kolkata",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    ...(typeof value === "number" ? { second: "2-digit" } : {}),
    hourCycle: "h23",
  });
};
const availableEstimate = (row) =>
  row.status === "estimated" &&
  known(row.observed_percentage_points) &&
  known(row.estimated_percentage_points);
const reasonLabel = (value) => label(value).replaceAll("_", " ");

function quotaGroup(group) {
  const estimate = group.rows[0];
  const available = group.rows.filter(availableEstimate);
  const observedIntervals = group.rows.filter((row) =>
    known(row.observed_percentage_points),
  );
  const observed = observedIntervals.length
    ? observedIntervals.reduce(
        (sum, row) => sum + row.observed_percentage_points,
        0,
      )
    : null;
  const estimated = available.length
    ? available.reduce((sum, row) => sum + row.estimated_percentage_points, 0)
    : null;
  const interval = el("article", undefined, "quota-estimate-interval");
  interval.append(
    el("h3", `${clientName(estimate.provider)} · ${label(estimate.email)}`),
    el(
      "p",
      `${label(estimate.bucket)} · ${label(estimate.window)} · Scheduled reset: ${when(estimate.resets_at)} IST`,
      "chart-caption",
    ),
    el(
      "p",
      `Observed account increase: ${points(observed)} · Conditional estimate: ${points(estimated)}`,
    ),
    el(
      "p",
      `${exactNumber(observedIntervals.length)} observed intervals · ${exactNumber(available.length)} allocated intervals · ${exactNumber(group.rows.length - available.length)} excluded from allocation. Totals cover these intervals only.`,
      "chart-caption",
    ),
  );
  const reasons = new Map();
  group.rows
    .filter((row) => !availableEstimate(row))
    .forEach((row) =>
      reasons.set(
        reasonLabel(row.reason),
        (reasons.get(reasonLabel(row.reason)) || 0) + 1,
      ),
    );
  if (reasons.size)
    interval.append(
      el(
        "p",
        [...reasons]
          .slice(0, 3)
          .map(([reason, count]) => `${reason} (${exactNumber(count)})`)
          .join(" · ") +
          (reasons.size > 3
            ? ` · ${reasons.size - 3} more reasons in interval details`
            : ""),
        "chart-caption",
      ),
    );
  const allocations = new Map();
  for (const row of available)
    for (const allocation of row.allocations || []) {
      const labels = [
        allocation.person,
        allocation.project,
        allocation.model,
        allocation.effort,
      ].map(label);
      const key = JSON.stringify(labels);
      if (!allocations.has(key)) allocations.set(key, { labels, value: 0 });
      if (known(allocation.percentage_points))
        allocations.get(key).value += allocation.percentage_points;
    }
  if (allocations.size) {
    const details = el("details", undefined, "composition-details");
    details.append(el("summary", "Allocation details"));
    details.addEventListener("toggle", () => {
      if (!details.open || details.dataset.loaded) return;
      details.dataset.loaded = "true";
      const rows = [...allocations.values()]
        .sort((a, b) => b.value - a.value)
        .slice(0, 50);
      details.append(
        el(
          "p",
          `Showing ${rows.length} of ${allocations.size} allocations from available intervals.`,
          "chart-caption",
        ),
        table(
          ["Person", "Project", "Model", "Effort", "Estimated pp"],
          rows.map((row) => [...row.labels, points(row.value)]),
          `${label(estimate.email)} ${label(estimate.window)} allocation`,
        ),
      );
    });
    interval.append(details);
  }
  const details = el(
    "details",
    undefined,
    "composition-details quota-interval-details",
  );
  details.append(
    el("summary", `Interval details (${exactNumber(group.rows.length)})`),
  );
  details.addEventListener("toggle", () => {
    if (!details.open || details.dataset.loaded) return;
    details.dataset.loaded = "true";
    const recent = [...group.rows]
      .sort((a, b) => new Date(b.to) - new Date(a.to))
      .slice(0, 50);
    details.append(
      el(
        "p",
        `Showing latest ${recent.length} of ${exactNumber(group.rows.length)} intervals · IST.`,
        "chart-caption",
      ),
      table(
        ["Interval", "Observed pp", "Estimated pp", "Status / reason"],
        recent.map((row) => [
          `${when(row.from)} – ${when(row.to)}`,
          points(row.observed_percentage_points),
          points(row.estimated_percentage_points),
          `${reasonLabel(row.status)} · ${reasonLabel(row.reason)}`,
        ]),
        `${label(estimate.email)} ${label(estimate.window)} intervals`,
      ),
    );
  });
  interval.append(details);
  return interval;
}

export function renderQuotaEstimates(estimates) {
  const section = el("section", undefined, "usage-section quota-estimates");
  section.append(
    el("h2", "Conditional quota estimates"),
    el(
      "p",
      "Allocation assumes captured associated activity represents each account interval. Accounts, windows and reset cycles stay separate.",
      "chart-caption",
    ),
  );
  const grouped = new Map();
  for (const row of estimates) {
    const key = JSON.stringify([
      row.provider || "",
      row.email || "",
      row.bucket || "",
      row.window || "",
      row.resets_at ?? null,
    ]);
    if (!grouped.has(key)) grouped.set(key, { rows: [] });
    grouped.get(key).rows.push(row);
  }
  const groups = [...grouped.values()];
  const count = el("p", undefined, "chart-caption");
  const root = el("div");
  const more = el("button", "Show more account windows");
  let visible = 0;
  const append = () => {
    const next = Math.min(visible + 20, groups.length);
    for (; visible < next; visible++) root.append(quotaGroup(groups[visible]));
    count.textContent = `Showing ${visible} of ${groups.length} account windows and reset cycles.`;
    more.hidden = visible === groups.length;
  };
  more.onclick = append;
  append();
  const method = el("details", undefined, "composition-method");
  method.append(
    el("summary", "Quota estimate method"),
    el(
      "p",
      "Observed totals include recorded same-cycle increases, even when allocation is withheld. Estimated totals include only allocated intervals. Neither total proves full-period coverage. Percentage points describe increases, not remaining allowance. Estimated allocations are proportional to captured associated weights; other devices or provider surfaces may be missing. Excluded intervals remain unallocated. Details show at most 50 recent intervals and 50 largest allocations per account window.",
    ),
  );
  section.append(count, root, more, method);
  return section;
}
