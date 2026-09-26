// All identities, device addresses and token counts in these fixtures are synthetic.
import { test, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";

const personID = "alice+analytics@example.test";
const project = "synthetic/repo & tools";
const counters = {
  input_tokens: 3120000,
  output_tokens: 850,
  cache_read_tokens: 1374596160,
  cache_write_tokens: null,
};
const people = [
  {
    id: personID,
    name: "Synthetic Alice",
    role: "member",
    active: true,
    sessions: 2,
    projects: [{ name: project, sessions: 2, last_day: "2026-09-26" }],
    devices: [
      {
        id: "synthetic-device",
        name: "Synthetic Mac",
        os: "darwin",
        last_seen: "2026-09-26T10:00:00Z",
      },
    ],
  },
];
const analytics = {
  start: "2026-09-13",
  end: "2026-09-26",
  timezone: "Asia/Kolkata",
  coverage: {
    sources: 2,
    unavailable_sources: 1,
    undated_points: 3,
    attribution_conflicts: 1,
    last_observed_at: "2026-09-26T10:00:00Z",
  },
  totals: { ...counters, points: 4 },
  daily: Array.from({ length: 14 }, (_, i) => ({
    day: `2026-09-${13 + i}`,
    input_tokens: i === 2 ? null : (i + 1) * 350,
    output_tokens: i * 42,
    cache_read_tokens: i * 650,
    cache_write_tokens: null,
  })),
  projects: [{ project, ...counters, people: [personID], sources: 2 }],
  people: [{ person: personID, ...counters, sources: 2 }],
  clients: [
    { client: "codex", ...counters },
    { client: "claude", ...counters, input_tokens: 1500 },
  ],
  models: [{ model: "synthetic-model", ...counters }],
};
async function fixture(
  page,
  {
    readOnly = false,
    empty = false,
    failure = false,
    principalPerson = personID,
    role = "owner",
  } = {},
) {
  const requests = [];
  let devices = [
    {
      id: "synthetic-device",
      person: personID,
      device: "Synthetic Mac",
      url: "http://127.0.0.1:8080/",
      has_key: true,
    },
    {
      id: "synthetic-lan",
      person: personID,
      device: "Synthetic Ubuntu",
      url: "http://192.168.1.12:8080/",
      has_key: false,
    },
  ];
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.startsWith("/api/v1/")) {
      requests.push({
        path: url.pathname,
        search: url.search,
        method: route.request().method(),
        body: route.request().postDataJSON(),
      });
      const path = url.pathname.slice(7);
      let json;
      if (path === "/me")
        json = {
          principal: {
            person: principalPerson,
            role,
            token_kind: readOnly ? "read" : "human",
          },
          policy: {
            content: "full",
            redaction: "none",
            visibility: "team",
            version: 1,
          },
          demo: true,
        };
      else if (path === "/people") json = people;
      else if (path === "/activity") json = [];
      else if (path === "/accounts")
        json = [
          {
            alias: "synthetic-claude",
            provider: "claude",
            plan: "Team",
            assigned: [personID],
          },
        ];
      else if (path === "/quota-observations")
        json = ["codex", "claude"].map((provider) => ({
          provider,
          person: personID,
          email: "shared@example.test",
          profile: "default",
          device: "Synthetic Mac",
          plan: "Synthetic plan",
          observed_at: new Date().toISOString(),
          windows: [
            {
              name: "primary",
              bucket: provider,
              used_percent: provider === "codex" ? 62 : 24,
              window_duration_mins: 300,
            },
          ],
        }));
      else if (path === "/analytics") {
        if (failure)
          return route.fulfill({
            status: 503,
            json: { error: "synthetic outage" },
          });
        json = empty
          ? {
              ...analytics,
              totals: {},
              daily: [],
              people: [],
              projects: [],
              models: [],
              clients: [],
              coverage: {
                sources: 0,
                unavailable_sources: 0,
                undated_points: 0,
                attribution_conflicts: 0,
              },
            }
          : analytics;
        const hours = Number(url.searchParams.get("hours"));
        if (hours) {
          const end = new Date("2026-09-26T16:07:00Z");
          const start = new Date(end - hours * 3600000);
          const step = hours === 1 ? 300000 : 3600000;
          json = {
            ...json,
            start: start.toISOString(),
            end: end.toISOString(),
            granularity: hours === 1 ? "5m" : "hour",
            daily: [],
            series: empty
              ? []
              : Array.from({ length: (hours * 3600000) / step }, (_, i) => ({
                  timestamp: new Date(+start + i * step).toISOString(),
                  ...counters,
                })),
          };
        }
      } else if (path === "/device-viewers") json = devices;
      else if (path.endsWith("/viewer-key"))
        json = { key: "synthetic-secret-never-in-url" };
      else if (
        path.endsWith("/viewer") &&
        route.request().method() === "POST"
      ) {
        const body = route.request().postDataJSON();
        devices = devices.map((d) =>
          path.includes(d.id)
            ? { ...d, url: body.url, has_key: Boolean(body.url) }
            : d,
        );
        json = { ok: true };
      } else if (path === "/dashboard")
        json = {
          sessions: 0,
          prompts: 0,
          projects: {},
          days: {},
          models: {},
          clients: {},
          indexed_sources: 0,
          unknown_dates: 0,
        };
      else json = [];
      return route.fulfill({ json });
    }
    if (url.pathname === "/" || /\.(js|css|woff2)$/.test(url.pathname)) {
      const name = url.pathname === "/" ? "index.html" : url.pathname.slice(1);
      const file = resolve("internal/web/assets", name);
      if (file.startsWith(resolve("internal/web/assets") + "/")) {
        const type = name.endsWith(".js")
          ? "text/javascript"
          : name.endsWith(".css")
            ? "text/css"
            : name.endsWith(".woff2")
              ? "font/woff2"
              : "text/html";
        return route.fulfill({ body: await readFile(file), contentType: type });
      }
    }
    return route.continue();
  });
  await page.addInitScript(() => {
    window.syntheticClipboard = [];
    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText: async (value) => window.syntheticClipboard.push(value),
      },
    });
  });
  return requests;
}

test("person detail deep link separates provider accounts, unknown counters and device viewer secrets", async ({
  page,
}) => {
  const requests = await fixture(page);
  await page.goto(`/#person/${encodeURIComponent(personID)}`);
  const surface = page.locator("#analytics-page");
  await expect(
    surface.getByRole("heading", { name: "Synthetic Alice", exact: true }),
  ).toBeVisible();
  await expect(
    surface.getByRole("heading", { name: "codex · shared@example.test" }),
  ).toBeVisible();
  await expect(
    surface.getByRole("heading", { name: "claude · shared@example.test" }),
  ).toBeVisible();
  await expect(
    surface.getByText(/Personal quota share: unknown/),
  ).toBeVisible();
  await expect(surface.locator(".token-totals > div").last()).toContainText(
    "Unknown",
  );
  await surface
    .getByRole("button", { name: "Cache write", exact: true })
    .click();
  await expect(
    surface.getByText(
      "Cache write token counts are not reported for this period.",
    ),
  ).toBeVisible();
  await surface.getByRole("button", { name: "Output", exact: true }).click();
  await expect(
    surface.getByRole("img", { name: /Hourly output tokens/ }),
  ).toBeVisible();
  await expect(
    surface.getByText("This computer only · loopback address"),
  ).toBeVisible();
  await expect(
    surface.getByText("Configured network address · reachability not verified"),
  ).toBeVisible();
  expect(requests.filter((r) => r.path.endsWith("viewer-key"))).toHaveLength(0);
  const viewer = surface.locator(".viewer-device").first();
  await expect(
    viewer.getByRole("link", { name: "Open AgentsView" }),
  ).toHaveAttribute("href", "http://127.0.0.1:8080/");
  await expect(
    viewer.getByRole("link", { name: "Open AgentsView" }),
  ).toHaveAttribute("rel", "noopener noreferrer");
  await expect(
    viewer.getByRole("link", { name: "Open AgentsView" }),
  ).toHaveAttribute("target", "_blank");
  await viewer.getByRole("button", { name: "Copy URL" }).click();
  await viewer.getByRole("button", { name: "Copy key" }).click();
  await expect
    .poll(() => page.evaluate(() => window.syntheticClipboard))
    .toEqual(["http://127.0.0.1:8080/", "synthetic-secret-never-in-url"]);
  expect(await page.locator("body").textContent()).not.toContain(
    "synthetic-secret-never-in-url",
  );
  await page.reload();
  await expect(
    surface.getByRole("heading", { name: "Synthetic Alice", exact: true }),
  ).toBeVisible();
});

test("project overview, contributors and filters link to distinct reloadable pages", async ({
  page,
}) => {
  const requests = await fixture(page);
  await page.goto("/#projects");
  await expect(page.locator('[data-tab="projects"]')).toHaveAttribute(
    "aria-current",
    "page",
  );
  await page
    .locator("#analytics-page")
    .getByRole("link", { name: project, exact: true })
    .click();
  await expect(page).toHaveURL(new RegExp("#project/"));
  await expect(
    page.getByRole("heading", { name: project, exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText(
      "Grouped by recorded project name; repository identity is not verified.",
    ),
  ).toBeVisible();
  await page.getByLabel("Period", { exact: true }).selectOption("90");
  await page.getByLabel("Client", { exact: true }).selectOption("claude");
  await expect
    .poll(() =>
      requests.some(
        (r) =>
          r.path.endsWith("/analytics") &&
          new URLSearchParams(r.search).get("project") === project &&
          new URLSearchParams(r.search).get("days") === "90" &&
          new URLSearchParams(r.search).get("client") === "claude",
      ),
    )
    .toBeTruthy();
  await page.reload();
  await expect(
    page.getByRole("heading", { name: project, exact: true }),
  ).toBeVisible();
  await page
    .locator("#analytics-page")
    .getByRole("link", { name: "Synthetic Alice" })
    .click();
  await expect(page).toHaveURL(new RegExp("#person/"));
  await expect(
    page.getByRole("heading", { name: "Synthetic Alice", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "People", exact: true }).click();
  await page
    .locator("#people-list")
    .getByRole("link", { name: "Synthetic Alice" })
    .click();
  await expect(page.locator("#analytics-page")).toBeVisible();
});

test("coverage, desktop and mobile pages stay readable without document overflow", async ({
  page,
}) => {
  await fixture(page);
  await page.goto(`/#person/${encodeURIComponent(personID)}`);
  await expect(page.locator(".viewer-device")).toHaveCount(2);
  await page.locator(".usage-coverage > summary").click();
  await expect(page.getByText(/Undated points excluded: 3/)).toBeVisible();
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBeTruthy();
    await expect(
      page
        .locator("#analytics-page")
        .getByRole("button", { name: "Cache write", exact: true }),
    ).toBeVisible();
    await page.screenshot({
      path: `.impeccable/review/analytics-${width === 1440 ? "desktop" : "mobile"}.png`,
      fullPage: true,
    });
  }
});

test("workspace viewer settings preserve blank keys and explicitly clear the connection", async ({
  page,
}) => {
  const requests = await fixture(page);
  await page.goto("/#settings");
  const setting = page.locator(".viewer-setting").first();
  await setting.locator("summary").click();
  await setting.getByLabel("AgentsView URL").fill("http://192.168.1.14:8080");
  await setting.getByRole("button", { name: "Save viewer" }).click();
  await expect
    .poll(() => requests.filter((r) => r.method === "POST").length)
    .toBe(1);
  expect(requests.filter((r) => r.method === "POST")[0].body).toEqual({
    url: "http://192.168.1.14:8080",
  });
  await setting.getByLabel("Access key").fill("synthetic-new-key");
  await setting.getByRole("button", { name: "Save viewer" }).click();
  await expect(setting.getByLabel("Access key")).toHaveValue("");
  await setting.getByLabel("AgentsView URL").fill("");
  await setting.getByRole("button", { name: "Save viewer" }).click();
  await expect(page.locator("#feedback")).toContainText(
    "Device viewer and key removed.",
  );
  expect(requests.filter((r) => r.method === "POST").at(-1).body).toEqual({
    url: "",
  });
});

test("read tokens never request viewer keys or expose mutation controls", async ({
  page,
}) => {
  const requests = await fixture(page, { readOnly: true });
  await page.goto(`/#person/${encodeURIComponent(personID)}`);
  await expect(page.locator(".viewer-device")).toHaveCount(2);
  await expect(page.getByRole("button", { name: "Copy key" })).toHaveCount(0);
  await page.getByRole("button", { name: "Workspace", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Device viewers" }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Save viewer" })).toHaveCount(
    0,
  );
  expect(
    requests.filter(
      (r) => r.path.endsWith("viewer-key") || r.method === "POST",
    ),
  ).toHaveLength(0);
});

for (const [name, options, message] of [
  [
    "empty imports",
    { empty: true },
    "Input token counts are not reported for this period.",
  ],
  [
    "failed analytics",
    { failure: true },
    "Token counts could not load. Use Refresh to retry.",
  ],
])
  test(`${name} remain explicit and preserve device access`, async ({
    page,
  }) => {
    await fixture(page, options);
    await page.goto(`/#person/${encodeURIComponent(personID)}`);
    await expect(page.getByText(message, { exact: true })).toBeVisible();
    await expect(
      page.getByRole("link", { name: "Open AgentsView" }),
    ).toHaveCount(2);
  });

test("another member can open a viewer but cannot copy its key", async ({
  page,
}) => {
  const requests = await fixture(page, {
    principalPerson: "synthetic-bob@example.test",
    role: "member",
  });
  await page.goto(`/#person/${encodeURIComponent(personID)}`);
  await expect(page.getByRole("link", { name: "Open AgentsView" })).toHaveCount(
    2,
  );
  await expect(page.getByRole("button", { name: "Copy key" })).toHaveCount(0);
  expect(requests.some((r) => r.path.endsWith("viewer-key"))).toBe(false);
});

test("viewer forms reject credential-bearing and disguised LAN URLs", async ({
  page,
}) => {
  const requests = await fixture(page);
  await page.goto("/#settings");
  const setting = page.locator(".viewer-setting").first();
  await setting.locator("summary").click();
  for (const address of [
    "http://127.example.test:8080",
    "http://10.example.test",
    "https://user:secret@example.test",
    "https://example.test/?key=secret",
    "https://example.test/#secret",
  ]) {
    await setting.getByLabel("AgentsView URL").fill(address);
    await setting.getByRole("button", { name: "Save viewer" }).click();
    await expect(page.locator("#feedback")).toContainText(
      "Use a valid viewer URL",
    );
  }
  expect(requests.filter((r) => r.method === "POST")).toHaveLength(0);
});

test("compact counts retain exact values and rolling hours reach analytics API", async ({
  page,
}) => {
  const requests = await fixture(page);
  await page.goto("/#projects");
  const root = page.locator("#analytics-page");
  await expect(page.getByLabel("Period", { exact: true })).toHaveValue("24h");
  await expect(root.locator(".token-totals dd").first()).toHaveText("3.12M");
  await expect(root.locator(".token-totals dd").first()).toHaveAttribute(
    "title",
    "3,120,000",
  );
  await expect(root.locator(".token-totals dd").nth(1)).toHaveText("850");
  await expect(root.locator(".token-totals dd").nth(2)).toHaveText("1.37B");
  for (const hours of [48, 24, 12, 1]) {
    await page.getByLabel("Period", { exact: true }).selectOption(`${hours}h`);
    await expect(
      root.getByRole("heading", {
        name: hours === 1 ? "Tokens every 5 minutes" : "Hourly tokens",
        exact: true,
      }),
    ).toBeVisible();
    await expect
      .poll(
        () =>
          requests.filter((r) => r.path === "/api/v1/analytics").at(-1)?.search,
      )
      .toContain(`hours=${hours}`);
    const query = new URLSearchParams(
      requests.filter((r) => r.path === "/api/v1/analytics").at(-1).search,
    );
    expect(query.has("days")).toBeFalsy();
  }
  await root.getByText("View exact token counts", { exact: true }).click();
  await expect(root.locator(".usage-daily-data td").first()).toHaveText(
    "3,120,000",
  );
  await page.setViewportSize({ width: 390, height: 844 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
});
