import { compactNumber as number, exactNumber } from "./number-format.js";
import { renderQuotas } from "./quota-view.js";
import { renderComposition, renderQuotaEstimates } from "./composition-view.js";

const el = (tag, text, cls) => {
  const n = document.createElement(tag);
  if (text !== undefined) n.textContent = text;
  if (cls) n.className = cls;
  return n;
};
const categories = [
  ["input_tokens", "Input"],
  ["output_tokens", "Output"],
  ["cache_read_tokens", "Cache read"],
  ["cache_write_tokens", "Cache write"],
];
const numberNode = (tag, value) => {
  const node = el(tag, number(value));
  node.title = exactNumber(value);
  node.setAttribute("aria-label", exactNumber(value));
  return node;
};
const series = (data) => data.series || data.daily || [];
const isHourly = (data) =>
  data.granularity === "hour" || data.granularity === "5m";
const chartHeading = (data) =>
  data.granularity === "5m"
    ? "Tokens every 5 minutes"
    : isHourly(data)
      ? "Hourly tokens"
      : "Daily tokens";
const pointLabel = (point, data, short = false) => {
  if (!point.timestamp)
    return short ? point.day.slice(5).replace("-", "/") : point.day;
  return new Intl.DateTimeFormat("en-GB", {
    timeZone: data.timezone || "Asia/Kolkata",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    ...(short && data.granularity === "5m"
      ? {}
      : { day: "2-digit", month: "short" }),
  }).format(new Date(point.timestamp));
};
const rangeLabel = (data, value) =>
  isHourly(data) ? pointLabel({ timestamp: value }, data) : value;
const link = (kind, id, text) => {
  const a = el("a", text);
  a.href = `#${kind}/${encodeURIComponent(id)}`;
  return a;
};
const timestamp = (value) =>
  value ? new Date(value).toLocaleString() : "Not reported";
let requestID = 0;
const selection = { period: "24h", client: "", category: "input_tokens" };

function selectControl(label, options, value, change) {
  const wrap = el("label", label);
  const select = el("select");
  select.setAttribute("aria-label", label);
  for (const [key, title] of options) {
    const option = el("option", title);
    option.value = key;
    select.append(option);
  }
  select.value = value;
  select.onchange = () => change(select.value);
  wrap.append(select);
  return wrap;
}

function usageTable(title, rows, nameKey, makeName, exact = false) {
  const section = el("section", undefined, "usage-section");
  section.append(el("h2", title));
  if (!rows.length) {
    section.append(el("p", "No recorded usage in this period.", "usage-empty"));
    return section;
  }
  const wrap = el("div", undefined, "usage-table-wrap");
  wrap.tabIndex = 0;
  wrap.setAttribute("role", "region");
  wrap.setAttribute("aria-label", title);
  const table = el("table", undefined, "usage-table");
  const head = el("thead");
  const tr = el("tr");
  for (const text of [title, ...categories.map(([, label]) => label)]) {
    const th = el("th", text);
    th.scope = "col";
    tr.append(th);
  }
  head.append(tr);
  const body = el("tbody");
  for (const row of rows) {
    const tr = el("tr");
    const name = el("th");
    name.scope = "row";
    name.append(
      makeName ? makeName(row) : el("span", row[nameKey] || "Unknown"),
    );
    tr.append(name);
    for (const [key] of categories) {
      const cell = numberNode("td", row[key]);
      if (exact) cell.textContent = exactNumber(row[key]);
      tr.append(cell);
    }
    body.append(tr);
  }
  table.append(head, body);
  wrap.append(table);
  section.append(wrap);
  return section;
}

function dailyChart(data, category) {
  const section = el("section", undefined, "usage-section daily-usage");
  section.append(el("h2", chartHeading(data)));
  const controls = el("div", undefined, "token-tabs");
  controls.setAttribute("role", "group");
  controls.setAttribute("aria-label", "Token category");
  const chartRoot = el("div");
  const draw = (key) => {
    selection.category = key;
    for (const b of controls.children)
      b.setAttribute("aria-pressed", String(b.dataset.key === key));
    chartRoot.replaceChildren(tokenChart(data, key));
  };
  for (const [key, label] of categories) {
    const b = el("button", label);
    b.dataset.key = key;
    b.onclick = () => draw(key);
    controls.append(b);
  }
  section.append(controls, chartRoot);
  draw(category);
  const details = el("details", undefined, "usage-daily-data");
  details.append(
    el("summary", "View exact token counts"),
    usageTable(
      isHourly(data) ? "Time" : "Date",
      series(data),
      "day",
      (row) => el("span", pointLabel(row, data)),
      true,
    ),
  );
  section.append(details);
  return section;
}
function tokenChart(data, key) {
  const root = el("div", undefined, "token-chart");
  const label = categories.find(([name]) => name === key)[1];
  const points = series(data);
  root.append(
    el(
      "p",
      `${label} tokens · ${rangeLabel(data, data.start)} to ${rangeLabel(data, data.end)} · ${data.timezone || "Timezone not reported"}`,
      "chart-caption",
    ),
  );
  if (!points.some((p) => typeof p[key] === "number")) {
    root.append(
      el(
        "p",
        `${label} token counts are not reported for this period.`,
        "usage-empty",
      ),
    );
    return root;
  }
  const ns = "http://www.w3.org/2000/svg";
  const svg = (tag, attrs = {}) => {
    const n = document.createElementNS(ns, tag);
    for (const [k, v] of Object.entries(attrs)) n.setAttribute(k, v);
    return n;
  };
  const graph = svg("svg", {
    viewBox: "0 0 900 230",
    role: "img",
    "aria-label": `${isHourly(data) ? "Hourly" : "Daily"} ${label.toLowerCase()} tokens, ${data.timezone || "timezone unknown"}. Missing counts are unknown.`,
  });
  const peak = Math.max(1, ...points.map((p) => p[key] || 0));
  for (const fraction of [0, 0.5, 1]) {
    const y = 184 - fraction * 152;
    graph.append(
      svg("line", { x1: 74, x2: 882, y1: y, y2: y, class: "chart-rule" }),
    );
    const text = svg("text", {
      x: 64,
      y: y + 4,
      "text-anchor": "end",
      class: "chart-label",
    });
    text.textContent = number(Math.round(peak * fraction));
    graph.append(text);
  }
  const step = 800 / points.length;
  points.forEach((point, i) => {
    const known = typeof point[key] === "number";
    const height = known ? (point[key] / peak) * 152 : 0;
    const mark = svg("rect", {
      x: 78 + i * step,
      y: known ? 184 - Math.max(1, height) : 178,
      width: Math.max(1, step - 4),
      height: known ? Math.max(1, height) : 6,
      rx: 1,
      class: known ? "chart-bar" : "token-unknown",
    });
    const title = svg("title");
    title.textContent = `${pointLabel(point, data)}: ${exactNumber(point[key])} ${label.toLowerCase()} tokens`;
    mark.append(title);
    graph.append(mark);
    if (
      i === 0 ||
      i === points.length - 1 ||
      (i % Math.ceil(points.length / (isHourly(data) ? 4 : 5)) === 0 &&
        points.length - 1 - i > 1)
    ) {
      const text = svg("text", {
        x: 78 + i * step + step / 2,
        y: 211,
        "text-anchor": "middle",
        class: "chart-label",
      });
      text.textContent = pointLabel(point, data, true);
      graph.append(text);
    }
  });
  const scroll = el("div", undefined, "token-chart-scroll");
  scroll.tabIndex = 0;
  scroll.setAttribute("role", "region");
  scroll.setAttribute(
    "aria-label",
    `${label} token chart; scroll to view dates`,
  );
  scroll.append(graph);
  root.append(
    scroll,
    el(
      "p",
      "Outlined marks indicate unknown counts. Recorded tokens are separate from subscription quota.",
      "chart-caption",
    ),
  );
  return root;
}

function coverage(data) {
  const c = data.coverage || {};
  const box = el("details", undefined, "usage-coverage");
  box.append(
    el(
      "summary",
      `Import coverage incomplete · ${exactNumber(c.sources)} recorded sources`,
    ),
  );
  box.append(
    el(
      "p",
      `Unavailable sources: ${exactNumber(c.unavailable_sources)}. Undated points excluded: ${exactNumber(c.undated_points)}. Attribution conflicts: ${exactNumber(c.attribution_conflicts)}. Conflicting copies excluded: ${exactNumber(c.content_conflicts ?? 0)}. Last observed: ${timestamp(c.last_observed_at)}.`,
    ),
  );
  box.append(
    el(
      "p",
      "Usage is grouped by enrolled device owner; historical prompt authors are unverified. This is available imported history, not total account usage. Unknown counters stay unknown; missing imports are not zero use. Tokens do not measure productivity.",
    ),
  );
  return box;
}

export async function renderAnalyticsPage(
  root,
  { kind, id = "", api, notify, principal, readOnly },
) {
  const request = ++requestID;
  const current = () => request === requestID && !root.hidden;
  root.replaceChildren(el("p", "Loading recorded usage…", "usage-empty"));
  root.setAttribute("aria-busy", "true");
  try {
    const people = await api("/people");
    if (!current()) return;
    const person = kind === "person" ? people.find((p) => p.id === id) : null;
    if (kind === "person" && !person) {
      root.replaceChildren(
        el("h1", "Person unavailable"),
        el("p", "This person is no longer visible to your account."),
        link("", "", "Back to people"),
      );
      root.lastElementChild.href = "#people";
      return;
    }
    const heading = el("div", undefined, "page-heading usage-heading");
    const title = el("div");
    const back = el(
      "a",
      kind === "person" ? "People" : "Projects",
      "usage-back",
    );
    back.href = kind === "person" ? "#people" : "#projects";
    if (id) title.append(back);
    title.append(
      el(
        "h1",
        kind === "person"
          ? person.name || person.id
          : kind === "project"
            ? id
            : "Projects",
      ),
    );
    title.append(
      el(
        "p",
        kind === "person"
          ? `${person.id} · ${person.active ? person.role : "Access revoked"}`
          : "Grouped by recorded project name; repository identity is not verified.",
      ),
    );
    heading.append(title);
    const controls = el("div", undefined, "usage-controls");
    const reload = () =>
      renderAnalyticsPage(root, { kind, id, api, notify, principal, readOnly });
    controls.append(
      selectControl(
        "Period",
        [
          ["today", "Today (IST)"],
          ["1h", "Last hour"],
          ["12h", "Last 12 hours"],
          ["24h", "Last 24 hours"],
          ["48h", "Last 48 hours"],
          ["7", "Last 7 days"],
          ["14", "Last 14 days"],
          ["30", "Last 30 days"],
          ["90", "Last 90 days"],
        ],
        selection.period,
        (value) => {
          selection.period = value;
          reload();
        },
      ),
    );
    controls.append(
      selectControl(
        "Client",
        [
          ["", "All clients"],
          ["codex", "Codex"],
          ["claude", "Claude"],
          ["antigravity", "Antigravity"],
        ],
        selection.client,
        (value) => {
          selection.client = value;
          reload();
        },
      ),
    );
    const refresh = el("button", "Refresh");
    refresh.onclick = reload;
    controls.append(refresh);
    root.replaceChildren(heading, controls);
    const dataRoot = el("div", undefined, "usage-content");
    dataRoot.append(el("p", "Loading token counts…", "usage-empty"));
    root.append(dataRoot);
    const query = new URLSearchParams({ client: selection.client });
    if (selection.period === "today") query.set("period", "today");
    else
      query.set(
        selection.period.endsWith("h") ? "hours" : "days",
        selection.period.replace(/h$/, ""),
      );
    if (kind === "person") query.set("person", id);
    if (kind === "project") query.set("project", id);
    try {
      const data = await api("/analytics?" + query);
      if (!current()) return;
      dataRoot.replaceChildren(
        el(
          "p",
          "Usage from enrolled devices · historical authors unverified",
          "chart-caption",
        ),
        coverage(data),
      );
      const totals = el("dl", undefined, "token-totals");
      for (const [key, label] of categories) {
        const item = el("div");
        item.append(
          el("dt", `${label} tokens`),
          numberNode("dd", data.totals?.[key]),
        );
        totals.append(item);
      }
      dataRoot.append(totals);
      if (data.composition)
        dataRoot.append(renderComposition(data.composition));
      if (data.quota_estimates?.length)
        dataRoot.append(renderQuotaEstimates(data.quota_estimates));
      dataRoot.append(dailyChart(data, selection.category));
      if (kind !== "project")
        dataRoot.append(
          usageTable(
            "Recorded projects",
            data.projects || [],
            "project",
            (row) => {
              const name = el("div");
              name.append(
                row.project
                  ? link("project", row.project, row.project)
                  : el("span", "Project unknown"),
              );
              name.append(
                el(
                  "small",
                  `${exactNumber(row.sources)} sources · ${(row.people || []).length} observed contributors`,
                ),
              );
              return name;
            },
          ),
        );
      if (kind !== "person")
        dataRoot.append(
          usageTable(
            "Observed contributors",
            data.people || [],
            "person",
            (row) =>
              row.person
                ? link(
                    "person",
                    row.person,
                    people.find((p) => p.id === row.person)?.name || row.person,
                  )
                : el("span", "Unknown attribution"),
          ),
        );
      dataRoot.append(
        usageTable("Coding clients", data.clients || [], "client"),
        usageTable("Recorded models", data.models || [], "model"),
      );
    } catch (error) {
      if (current())
        dataRoot.replaceChildren(
          el(
            "p",
            "Token counts could not load. Use Refresh to retry.",
            "notice",
          ),
        );
    }
    if (!current() || kind !== "person") return;
    const quotaRoot = el(
      "section",
      undefined,
      "usage-section person-subscriptions",
    );
    const devicesRoot = el("section", undefined, "usage-section");
    root.append(quotaRoot, devicesRoot);
    quotaRoot.append(
      el("h2", "Observed subscriptions"),
      el("p", "Loading account observations…"),
    );
    devicesRoot.append(
      el("h2", "Devices & conversations"),
      el("p", "Loading viewer connections…"),
    );
    const results = await Promise.allSettled([
      api("/quota-observations"),
      api("/device-viewers"),
      api("/accounts"),
    ]);
    if (!current()) return;
    if (results[0].status === "fulfilled") {
      const observations = results[0].value;
      const observedAccounts = new Set(
        observations
          .filter((row) => row.person === id && row.email)
          .map((row) => JSON.stringify([row.provider, row.email])),
      );
      renderQuotas(
        quotaRoot,
        observations.filter(
          (row) =>
            observedAccounts.has(JSON.stringify([row.provider, row.email])) ||
            (row.person === id && row.error),
        ),
      );
    } else
      quotaRoot.replaceChildren(
        el("h2", "Observed subscriptions"),
        el(
          "p",
          "Account observations could not load. Use Refresh to retry.",
          "notice",
        ),
      );
    quotaRoot.append(
      el(
        "p",
        "Personal quota share: unknown. Provider percentages describe the shared account, not this person’s consumption.",
        "notice",
      ),
    );
    if (results[2].status === "fulfilled") {
      const assigned = results[2].value.filter((account) =>
        (account.assigned || []).includes(id),
      );
      if (assigned.length) {
        const list = el("div", undefined, "declared-accounts");
        list.append(el("h3", "Declared assignments"));
        for (const account of assigned)
          list.append(
            el(
              "p",
              `${account.provider} · ${account.alias} · ${account.plan || "Plan unknown"}`,
            ),
          );
        list.append(el("small", "Assignments do not prove session usage."));
        quotaRoot.append(list);
      }
    }
    if (results[1].status === "fulfilled")
      renderDeviceViewers(
        devicesRoot,
        results[1].value.filter((d) => d.person === id),
        { api, notify, readOnly, principal },
      );
    else
      devicesRoot.replaceChildren(
        el("h2", "Devices & conversations"),
        el(
          "p",
          "Viewer connections could not load. Use Refresh to retry.",
          "notice",
        ),
      );
  } catch (error) {
    if (!current()) return;
    const retry = el("button", "Retry");
    retry.onclick = () =>
      renderAnalyticsPage(root, { kind, id, api, notify, principal, readOnly });
    root.replaceChildren(
      el("h1", "Usage unavailable"),
      el("p", "Could not load this page. Check your connection and try again."),
      retry,
    );
  } finally {
    if (request === requestID) root.removeAttribute("aria-busy");
  }
}

export function safeViewerURL(value) {
  try {
    const url = new URL(value);
    if (
      !["http:", "https:"].includes(url.protocol) ||
      url.username ||
      url.password ||
      url.search ||
      url.hash
    )
      return null;
    const host = url.hostname;
    const octets = host.split(".");
    const ipv4 =
      octets.length === 4 &&
      octets.every((part) => /^\d+$/.test(part) && Number(part) <= 255);
    const local =
      host === "localhost" ||
      host === "[::1]" ||
      (ipv4 && Number(octets[0]) === 127);
    const lan =
      ipv4 &&
      (Number(octets[0]) === 10 ||
        (Number(octets[0]) === 192 && Number(octets[1]) === 168) ||
        (Number(octets[0]) === 172 &&
          Number(octets[1]) >= 16 &&
          Number(octets[1]) <= 31));
    if (url.protocol === "http:" && !local && !lan) return null;
    return { url: url.href, local };
  } catch {
    return null;
  }
}
function copyAction(label, value, notify) {
  const button = el("button", label);
  button.onclick = async () => {
    button.disabled = true;
    try {
      await navigator.clipboard.writeText(await value());
      notify(`${label === "Copy key" ? "Viewer key" : "Viewer URL"} copied.`);
    } catch {
      notify(
        `Could not ${label.toLowerCase()}. Check access and clipboard permissions, then retry.`,
        true,
      );
    } finally {
      button.disabled = false;
    }
  };
  return button;
}
export function renderDeviceViewers(
  root,
  devices,
  { api, notify, readOnly, principal },
) {
  root.replaceChildren(
    el("h2", "Devices & conversations"),
    el("p", "Open the full conversation history in that device’s AgentsView."),
  );
  if (!devices.length)
    root.append(el("p", "No enrolled devices are visible.", "usage-empty"));
  for (const device of devices) {
    const row = el("article", undefined, "viewer-device");
    const identity = el("div");
    identity.append(el("h3", device.device || device.id));
    const safe = safeViewerURL(device.url);
    if (safe) {
      identity.append(
        el("p", safe.url, "viewer-address"),
        el(
          "small",
          safe.local
            ? "This computer only · loopback address"
            : "Configured network address · reachability not verified",
        ),
      );
      const actions = el("div", undefined, "viewer-actions");
      const open = el("a", "Open AgentsView", "primary");
      open.href = safe.url;
      open.target = "_blank";
      open.rel = "noopener noreferrer";
      actions.append(
        open,
        copyAction("Copy URL", async () => safe.url, notify),
      );
      if (!device.has_key)
        identity.append(el("small", "No viewer key configured"));
      if (
        !readOnly &&
        device.has_key &&
        (principal?.person === device.person ||
          ["owner", "manager"].includes(principal?.role))
      )
        actions.append(
          copyAction(
            "Copy key",
            async () => {
              const result = await api(
                `/devices/${encodeURIComponent(device.id)}/viewer-key`,
              );
              if (!result.key) throw new Error("No key configured");
              return result.key;
            },
            notify,
          ),
        );
      row.append(identity, actions);
    } else {
      identity.append(
        el(
          "p",
          "Viewer address not configured. An owner or manager can add it in Workspace.",
        ),
      );
      row.append(identity);
    }
    root.append(row);
  }
  root.append(
    el(
      "p",
      "LAN access needs a reachable authenticated listener and appropriate firewall permissions. This page does not change the device’s network settings.",
      "chart-caption",
    ),
  );
}

export async function renderViewerSettings(
  root,
  { api, notify, readOnly, principal },
) {
  root.replaceChildren(
    el("h2", "Device viewers"),
    el(
      "p",
      "Save each device’s AgentsView address. Its access key is stored separately and copied only on request.",
    ),
  );
  try {
    const devices = await api("/device-viewers");
    if (!devices.length) {
      root.append(
        el("p", "Connect a device to configure its viewer.", "usage-empty"),
      );
      return;
    }
    for (const device of devices) {
      const item = el("details", undefined, "viewer-setting");
      item.append(
        el("summary", `${device.device || device.id} · ${device.person}`),
      );
      if (readOnly || !["owner", "manager"].includes(principal.role)) {
        item.append(el("p", device.url || "No viewer address configured"));
        root.append(item);
        continue;
      }
      const form = el("form", undefined, "viewer-form");
      const address = el("label", "AgentsView URL");
      const url = el("input");
      url.type = "url";
      url.name = "url";
      url.value = device.url || "";
      url.placeholder = "http://127.0.0.1:8080";
      address.append(url);
      const secret = el("label", "Access key");
      const key = el("input");
      key.type = "password";
      key.name = "key";
      key.autocomplete = "new-password";
      key.placeholder = "Leave blank to keep the key at the same origin";
      secret.append(key);
      const help = el(
        "small",
        "Use a loopback URL for this computer, a private LAN IP, or an HTTPS address. Changing the address origin clears its old key; clearing the URL removes the connection and key.",
      );
      const save = el("button", "Save viewer", "primary");
      save.type = "submit";
      form.append(address, secret, help, save);
      form.onsubmit = async (event) => {
        event.preventDefault();
        if (save.disabled) return;
        const value = url.value.trim();
        if (value && !safeViewerURL(value)) {
          notify(
            "Use a valid viewer URL without credentials, query parameters or a fragment.",
            true,
          );
          return;
        }
        save.disabled = true;
        try {
          const body = { url: value };
          if (key.value) body.key = key.value;
          await api(`/devices/${encodeURIComponent(device.id)}/viewer`, {
            method: "POST",
            body: JSON.stringify(body),
          });
          key.value = "";
          notify(
            value ? "Device viewer saved." : "Device viewer and key removed.",
          );
        } catch (error) {
          notify(error.message, true);
        } finally {
          save.disabled = false;
        }
      };
      item.append(form);
      root.append(item);
    }
  } catch {
    root.append(
      el(
        "p",
        "Device viewers could not load. Reopen Workspace to retry.",
        "notice",
      ),
    );
  }
}
