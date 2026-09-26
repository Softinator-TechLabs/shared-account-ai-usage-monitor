// Provider quota is account-wide evidence, never a sum of per-device percentages.
export function quotaAccounts(observations, now = Date.now()) {
  const sorted = observations
    .map((row) => ({
      ...row,
      windows: Array.isArray(row.windows) ? row.windows : [],
    }))
    .sort((a, b) => Date.parse(b.observed_at) - Date.parse(a.observed_at));
  const latestProfiles = new Map();
  for (const row of sorted) {
    const key = JSON.stringify([
      row.provider,
      row.person,
      row.device,
      row.profile,
    ]);
    if (!latestProfiles.has(key)) latestProfiles.set(key, row);
  }
  const groups = new Map();
  for (const row of sorted) {
    if (!row.email || row.error) continue;
    const key = JSON.stringify([row.provider, row.email]);
    if (!groups.has(key))
      groups.set(key, {
        latest: row,
        history: [],
        devices: [],
        health: "Observed",
      });
    const group = groups.get(key);
    group.history.push(row);
    if (
      !group.devices.some(
        (d) =>
          d.device === row.device &&
          d.profile === row.profile &&
          d.person === row.person,
      )
    )
      group.devices.push({
        device: row.device,
        profile: row.profile,
        person: row.person,
      });
  }
  for (const group of groups.values()) {
    const age = now - Date.parse(group.latest.observed_at);
    if (
      age > 180000 ||
      group.latest.windows.some((w) => w.resets_at && w.resets_at * 1000 <= now)
    )
      group.health = "Needs refresh";
    if (
      group.devices.some(
        (d) =>
          latestProfiles.get(
            JSON.stringify([
              group.latest.provider,
              d.person,
              d.device,
              d.profile,
            ]),
          )?.error,
      )
    )
      group.health = "Read failed";
  }
  return [...groups.values()];
}
function el(tag, text, cls) {
  const n = document.createElement(tag);
  if (text !== undefined) n.textContent = text;
  if (cls) n.className = cls;
  return n;
}
const format = (v) =>
  new Date(v).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
export function renderQuotas(root, rows) {
  root.replaceChildren();
  const accounts = quotaAccounts(rows);
  const heading = el("div", undefined, "quota-heading");
  heading.append(
    el("h2", "Observed subscriptions"),
    el("span", "Provider account observations", "muted"),
  );
  root.append(heading);
  const latestProfiles = new Map();
  for (const row of [...rows].sort(
    (a, b) => Date.parse(b.observed_at) - Date.parse(a.observed_at),
  )) {
    const key = JSON.stringify([
      row.provider,
      row.person,
      row.device,
      row.profile,
    ]);
    if (!latestProfiles.has(key)) latestProfiles.set(key, row);
  }
  for (const row of latestProfiles.values()) {
    if (row.error)
      root.append(
        el(
          "p",
          `${row.device} · ${row.profile}: ${row.provider || "Provider"} read failed at ${format(row.observed_at)}. Check that profile’s sign-in.`,
          "notice quota-read-error",
        ),
      );
  }

  if (!accounts.length) {
    root.append(
      el(
        "p",
        rows.some((r) => r.error)
          ? "The provider could not be read on the connected profile. Last attempt: " +
              format(rows[0].observed_at)
          : "Waiting for a connected device’s first account observation.",
        "notice",
      ),
    );
    return;
  }
  const grid = el("div", undefined, "native-quota-grid");
  root.append(grid);
  for (const group of accounts) {
    const { latest, health } = group;
    const card = el("article", undefined, "native-quota");
    const top = el("div", undefined, "quota-heading");
    top.append(
      el("span", latest.plan || "Subscription", "role-tag"),
      el(
        "span",
        health,
        health === "Observed" ? "quota-health" : "quota-health stale",
      ),
    );
    card.append(
      top,
      el("h3", `${latest.provider} · ${latest.email}`),
      el("p", `Read ${format(latest.observed_at)}`, "muted"),
    );
    for (const win of latest.windows) {
      const wrap = el("section", undefined, "quota-window");
      const duration = win.window_duration_mins;
      const label = duration
        ? duration % 1440 === 0
          ? `${duration / 1440}-day window`
          : duration % 60 === 0
            ? `${duration / 60}-hour window`
            : `${duration}-minute window`
        : win.name;
      wrap.append(el("span", `${win.bucket} · ${label}`, "muted"));
      const current = typeof win.used_percent === "number";
      const amount = el("div", undefined, "quota-amount");
      amount.append(
        el("strong", current ? `${win.used_percent}%` : "Unknown"),
        el("span", current ? "used when observed" : "No percentage reported"),
      );
      wrap.append(amount);
      if (current) {
        const meter = el("meter");
        meter.min = 0;
        meter.max = 100;
        meter.value = win.used_percent;
        meter.setAttribute("aria-label", `${win.bucket} ${label} used`);
        wrap.append(
          meter,
          el(
            "small",
            `${Math.max(0, 100 - win.used_percent)}% remaining at that reading`,
          ),
        );
      }
      wrap.append(
        el(
          "p",
          win.resets_at
            ? `Next reset reported: ${format(win.resets_at * 1000)}`
            : "Reset time not reported",
          "muted",
        ),
      );
      const points = group.history
        .filter((r) =>
          r.windows.some(
            (w) =>
              w.bucket === win.bucket &&
              w.name === win.name &&
              typeof w.used_percent === "number",
          ),
        )
        .slice(0, 40)
        .reverse();
      if (points.length > 1) {
        const svg = document.createElementNS(
          "http://www.w3.org/2000/svg",
          "svg",
        );
        svg.setAttribute("viewBox", "0 0 300 50");
        svg.setAttribute("preserveAspectRatio", "none");
        svg.setAttribute("role", "img");
        svg.setAttribute(
          "aria-label",
          "Recent account usage observations; resets may change the window",
        );
        svg.classList.add("quota-spark");
        const poly = document.createElementNS(svg.namespaceURI, "polyline");
        poly.setAttribute(
          "points",
          points
            .map((r, i) => {
              const w = r.windows.find(
                (w) => w.bucket === win.bucket && w.name === win.name,
              );
              return `${(i * 300) / (points.length - 1)},${48 - w.used_percent * 0.44}`;
            })
            .join(" "),
        );
        svg.append(poly);
        wrap.append(svg);
      }
      card.append(wrap);
    }
    if (!latest.windows.length)
      card.append(el("p", "No quota windows reported.", "notice"));
    const details = el("details");
    details.append(
      el(
        "summary",
        `Devices & history · ${group.devices.length} profile${group.devices.length === 1 ? "" : "s"}`,
      ),
    );
    for (const d of group.devices)
      details.append(
        el("p", `${d.device} · ${d.profile} · collector: ${d.person}`),
      );
    details.append(
      el(
        "small",
        "Profile sign-in evidence. This does not identify the account behind earlier prompts.",
      ),
    );
    const list = el("ol", undefined, "quota-history");
    for (const r of group.history.slice(0, 30)) {
      list.append(
        el(
          "li",
          `${format(r.observed_at)} · ${r.device} · ${r.windows.map((w) => `${w.bucket}/${w.name} ${w.used_percent ?? "unknown"}${w.used_percent == null ? "" : "%"}${w.resets_at ? " · reset " + format(w.resets_at * 1000) : ""}`).join("; ")}`,
        ),
      );
    }
    details.append(list);
    card.append(details);
    grid.append(card);
  }
}
