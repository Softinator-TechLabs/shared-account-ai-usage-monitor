const el = (tag, text, cls) => {
  const n = document.createElement(tag);
  if (text !== undefined) n.textContent = text;
  if (cls) n.className = cls;
  return n;
};
const number = (n) => new Intl.NumberFormat("en-IN").format(n);
export function renderDashboard(root, data) {
  root.replaceChildren();
  const facts = el("p", undefined, "activity-facts");
  for (const [count, label] of [
    [data.sessions, "sessions"],
    [data.prompts, "prompts in these sessions"],
    [Object.keys(data.projects).length, "projects"],
  ]) {
    const item = el("span");
    item.append(
      el("strong", number(count)),
      document.createTextNode(` ${label}`),
    );
    facts.append(item);
  }
  root.append(facts);
  if (!data.sessions) {
    const empty = el("div", undefined, "chart-empty");
    empty.append(
      el("h3", "No captured sessions in this period"),
      el(
        "p",
        "Choose a longer period, or connect a device. Charts fill as history syncs.",
      ),
    );
    root.append(empty);
  } else {
    const grid = el("div", undefined, "analytics-grid");
    const activity = el("section", undefined, "activity-chart");
    activity.append(el("h3", "Sessions started"));
    const ns = "http://www.w3.org/2000/svg";
    const svg = (tag, attrs) => {
      const n = document.createElementNS(ns, tag);
      for (const [k, v] of Object.entries(attrs)) n.setAttribute(k, String(v));
      return n;
    };
    const points = Object.entries(data.days).sort(([a], [b]) =>
      a.localeCompare(b),
    );
    const graph = svg("svg", {
      viewBox: "0 0 760 220",
      role: "img",
      "aria-label": `Daily captured sessions from ${data.start} to ${data.end}, Asia/Kolkata`,
    });
    const peak = Math.max(1, ...points.map(([, n]) => n));
    const scale = 150 / peak;
    for (const fraction of [0, 0.5, 1]) {
      const y = 180 - fraction * 150;
      graph.append(
        svg("line", { x1: 35, x2: 748, y1: y, y2: y, class: "chart-rule" }),
      );
      const label = svg("text", {
        x: 28,
        y: y + 4,
        "text-anchor": "end",
        class: "chart-label",
      });
      label.textContent = number(Math.round(peak * fraction));
      graph.append(label);
    }
    const step = 704 / points.length;
    points.forEach(([day, value], i) => {
      const x = 40 + i * step;
      const bar = svg("rect", {
        x,
        y: 180 - value * scale,
        width: Math.max(2, step - 4),
        height: value * scale,
        rx: 2,
        class: "chart-bar",
      });
      const title = svg("title", {});
      title.textContent = `${day}: ${number(value)} sessions`;
      bar.append(title);
      graph.append(bar);
      if (
        i === 0 ||
        i === points.length - 1 ||
        (points.length <= 14 && i % 3 === 0)
      ) {
        const label = svg("text", {
          x: x + step / 2,
          y: 205,
          "text-anchor": "middle",
          class: "chart-label",
        });
        label.textContent = day.slice(5).replace("-", "/");
        graph.append(label);
      }
    });
    activity.append(
      graph,
      el("p", "Session start date · IST", "chart-caption"),
    );
    const detail = el("details");
    detail.append(el("summary", "View daily counts"));
    const table = el("table", undefined, "chart-data");
    const head = el("tr");
    head.append(el("th", "Date (IST)"), el("th", "Sessions"));
    table.append(head);
    for (const [day, n] of points) {
      const row = el("tr");
      row.append(el("td", day), el("td", number(n)));
      table.append(row);
    }
    detail.append(table);
    activity.append(detail);
    grid.append(activity);
    const bars = (title, values, unit) => {
      const panel = el("section", undefined, "breakdown");
      panel.append(el("h3", title), el("p", unit, "chart-caption"));
      const sorted = Object.entries(values).sort((a, b) => b[1] - a[1]);
      const top = sorted.slice(0, 6);
      if (sorted.length > 6)
        top.push(["Other", sorted.slice(6).reduce((a, x) => a + x[1], 0)]);
      const max = Math.max(1, ...top.map((x) => x[1]));
      if (!top.length)
        panel.append(
          el("p", "No recorded model information in these sessions.", "muted"),
        );
      for (const [name, count] of top) {
        const row = el("div", undefined, "breakdown-row");
        const label = el("div");
        const nameEl = el("span", name);
        nameEl.title = name;
        label.append(nameEl, el("strong", number(count)));
        const track = el("div", undefined, "bar-track");
        const fill = el(
          "span",
          undefined,
          name === "Unknown" ? "bar-fill unknown" : "bar-fill",
        );
        fill.style.width = `${(count / max) * 100}%`;
        track.append(fill);
        row.append(label, track);
        panel.append(row);
      }
      return panel;
    };
    grid.append(
      bars("Projects", data.projects, "Captured sessions"),
      bars(
        "Recorded models",
        data.models,
        "Assistant messages in these sessions",
      ),
      bars("Coding clients", data.clients, "Captured sessions"),
    );
    root.append(grid);
  }
  const coverage = el("details", undefined, "chart-coverage");
  coverage.append(
    el(
      "summary",
      `${number(data.indexed_sources)} sources indexed${data.pending_captures ? ` · ${number(data.pending_captures)} captures waiting` : ""} · About these counts`,
    ),
    el(
      "p",
      `One longest capture per source, not one count per import. Activity includes messages from sessions started in the selected period. ${number(data.unknown_dates)} sources have unknown start dates and are excluded. Copied sessions may overlap. Missing models stay unknown. These are activity counts, not quota percentages or productivity scores.`,
    ),
  );
  root.append(coverage);
}
