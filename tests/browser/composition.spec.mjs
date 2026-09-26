// Synthetic identities and usage only; no employee activity is embedded here.
import { test, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";

const person = "synthetic-member@example.test";
const rows = [
  {
    client: "codex",
    model: "synthetic-model-a",
    effort: "high",
    project: "synthetic/api",
    branch: "main",
    weight: 30,
    prompts: 3,
    generated_lines: 24,
    points: 3,
    unweighted_points: 0,
  },
  {
    client: "codex",
    model: "synthetic-model-b",
    effort: "low",
    project: "synthetic/web",
    weight: 10,
    prompts: 1,
    generated_lines: 0,
    points: 1,
    unweighted_points: 0,
  },
  {
    client: "codex",
    model: "",
    effort: "",
    project: "",
    weight: null,
    prompts: null,
    generated_lines: null,
    points: 2,
    unweighted_points: 2,
  },
  {
    client: "claude",
    model: "synthetic-claude",
    effort: "",
    project: "synthetic/web",
    weight: 900,
    prompts: 2,
    generated_lines: 8,
    points: 2,
    unweighted_points: 0,
  },
];
const estimates = [
  {
    provider: "codex",
    email: "shared@example.test",
    bucket: "codex",
    window: "primary",
    from: "2026-09-26T08:00:00Z",
    to: "2026-09-26T09:00:00Z",
    resets_at: 1790427600,
    observed_percentage_points: 8,
    estimated_percentage_points: 6,
    status: "estimated",
    reason: "Captured associated activity only",
    allocations: [
      {
        person,
        project: "synthetic/api",
        model: "synthetic-model-a",
        effort: "high",
        percentage_points: 6,
      },
    ],
  },
  {
    provider: "codex",
    email: "other@example.test",
    bucket: "codex",
    window: "secondary",
    from: "2026-09-26T08:00:00Z",
    to: "2026-09-26T09:00:00Z",
    observed_percentage_points: null,
    estimated_percentage_points: null,
    status: "unavailable",
    reason: "Incomplete interval boundaries",
    allocations: [],
  },
];
async function fixture(
  page,
  { compositionRows = rows, quotaEstimates = estimates } = {},
) {
  const queries = [];
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.startsWith("/api/v1/")) {
      const path = url.pathname.slice(7);
      let json = [];
      if (path === "/me")
        json = {
          principal: { person, role: "member", token_kind: "human" },
          policy: {
            content: "full",
            redaction: "none",
            visibility: "self_managers",
            version: 1,
          },
          demo: true,
        };
      if (path === "/people")
        json = [
          {
            id: person,
            name: "Synthetic Member",
            role: "member",
            active: true,
            projects: [],
            devices: [],
          },
        ];
      if (path === "/analytics") {
        queries.push(url.searchParams);
        json = {
          start: "2026-09-25T18:30:00Z",
          end: "2026-09-26T10:00:00Z",
          timezone: "Asia/Kolkata",
          granularity: "hour",
          coverage: { sources: 3 },
          totals: {},
          series: [],
          composition: {
            rate_version: "synthetic-rates-v1",
            rows: compositionRows,
            activity_sources: 2,
            activity_unavailable_sources: 1,
          },
          quota_estimates: quotaEstimates,
        };
      }
      return route.fulfill({ json });
    }
    const name = url.pathname === "/" ? "index.html" : url.pathname.slice(1);
    const file = resolve("internal/web/assets", name);
    if (file.startsWith(resolve("internal/web/assets") + "/"))
      return route.fulfill({
        body: await readFile(file),
        contentType: name.endsWith(".js")
          ? "text/javascript"
          : name.endsWith(".css")
            ? "text/css"
            : name.endsWith(".woff2")
              ? "font/woff2"
              : "text/html",
      });
    return route.abort();
  });
  await page.goto(`/#person/${encodeURIComponent(person)}`);
  return queries;
}

test("member Today breakdown keeps client denominators separate and missing capture unknown", async ({
  page,
}) => {
  const queries = await fixture(page);
  await page.getByLabel("Period", { exact: true }).selectOption("today");
  await expect.poll(() => queries.at(-1)?.get("period")).toBe("today");
  expect(queries.at(-1).has("days")).toBe(false);
  expect(queries.at(-1).has("hours")).toBe(false);
  expect(queries.at(-1).get("person")).toBe(person);
  const codex = page
    .locator(".composition-client")
    .filter({ has: page.getByRole("heading", { name: "Codex", exact: true }) });
  await expect(
    codex.getByRole("img", {
      name: /Codex model and effort.*captured weighted usage/,
    }),
  ).toBeVisible();
  await expect(
    codex.getByRole("img", { name: /Codex recorded projects/ }),
  ).toBeVisible();
  await expect(codex.locator(".composition-legend").first()).toContainText(
    "75%",
  );
  await expect(codex.locator(".composition-legend").first()).toContainText(
    "25%",
  );
  await expect(codex).toContainText("2 unweighted points excluded");
  await codex.getByText("Model and effort details", { exact: true }).click();
  await expect(codex.getByRole("table")).toContainText("Prompts");
  await expect(codex.getByRole("table")).toContainText("Proposed lines");
  await expect(
    codex.getByRole("row", { name: /synthetic-model-a.*high/ }),
  ).toContainText("24");
  await expect(
    codex.getByRole("row", { name: /Unknown.*Unknown/ }),
  ).toContainText("Unknown");
  const claude = page.locator(".composition-client").filter({
    has: page.getByRole("heading", { name: "Claude", exact: true }),
  });
  await expect(claude.locator(".composition-legend").first()).toContainText(
    "100%",
  );
  await page
    .getByText("How these shares are estimated", { exact: true })
    .click();
  await expect(page.getByText(/Effort is recorded metadata/)).toBeVisible();
  await expect(
    page.getByText(/2 sources with activity.*1 unavailable/),
  ).toBeVisible();
});

test("quota estimates retain account and window boundaries and explain conditional attribution", async ({
  page,
}) => {
  await fixture(page);
  const intervals = page.locator(".quota-estimate-interval");
  await expect(intervals).toHaveCount(2);
  await expect(intervals.first()).toContainText(
    "Observed account increase: 8 pp",
  );
  await expect(intervals.first()).toContainText("Conditional estimate: 6 pp");
  await expect(intervals.last()).toContainText("Conditional estimate: Unknown");
  await expect(intervals.last()).toContainText(
    "Incomplete interval boundaries",
  );
  await intervals
    .first()
    .getByText("Allocation details", { exact: true })
    .click();
  await expect(intervals.first().getByRole("table")).toContainText(
    "synthetic/api",
  );
  await expect(
    page.getByText(/assumes captured associated activity represents/),
  ).toBeVisible();
});

test("unknown and empty composition never draw a zero-usage pie", async ({
  page,
}) => {
  await fixture(page, { compositionRows: [rows[2]], quotaEstimates: [] });
  await expect(page.locator(".composition-pie")).toHaveCount(0);
  await expect(
    page.getByText("No weighted usage captured for this client."),
  ).toBeVisible();
  await expect(
    page.getByText("Conditional quota estimates", { exact: true }),
  ).toHaveCount(0);
  await page.locator(".composition-client summary").click();
  await expect(page.locator(".composition-table")).toContainText("Unknown");
});

test("untrusted labels remain text, and chart legends wrap on desktop and mobile", async ({
  page,
}) => {
  await fixture(page, {
    compositionRows: [
      ...rows,
      {
        ...rows[0],
        model: '<img src=x onerror="window.injected=true">',
        project: "synthetic/" + "long-project-name".repeat(12),
        effort: "<script>bad()</script>",
      },
    ],
  });
  await expect(page.locator(".composition-client img")).toHaveCount(0);
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    await expect(page.locator(".composition-pie")).toHaveCount(4);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBeTruthy();
    await page.screenshot({
      path: `.impeccable/review/composition-${width === 1440 ? "desktop" : "mobile"}.png`,
      fullPage: true,
    });
  }
  expect(await page.evaluate(() => window.injected)).toBeUndefined();
});

test("empty composition explains missing capture without manufacturing counts", async ({
  page,
}) => {
  await fixture(page, { compositionRows: [], quotaEstimates: [] });
  await expect(
    page.getByText("No usage breakdown captured in this period."),
  ).toBeVisible();
  await expect(page.locator(".composition-pie")).toHaveCount(0);
});

test("grouping across branches preserves unknown activity counts", async ({
  page,
}) => {
  await fixture(page, {
    compositionRows: [
      rows[0],
      {
        ...rows[0],
        branch: "feature/synthetic",
        prompts: null,
        generated_lines: null,
      },
    ],
    quotaEstimates: [],
  });
  await page.getByText("Model and effort details", { exact: true }).click();
  const row = page.getByRole("row", { name: /synthetic-model-a.*high/ });
  await expect(row).toContainText("100%");
  await expect(
    row.getByRole("cell", { name: "Unknown", exact: true }),
  ).toHaveCount(2);
});

test("quota summaries bound long interval lists, exclude unavailable deltas and preserve reset cycles", async ({
  page,
}) => {
  const many = Array.from({ length: 1000 }, (_, i) => ({
    ...estimates[0],
    from: new Date(Date.UTC(2026, 8, 26, 0, i)).toISOString(),
    to: new Date(Date.UTC(2026, 8, 26, 0, i + 1)).toISOString(),
    observed_percentage_points: 0.01,
    estimated_percentage_points: 0.005,
  }));
  await fixture(page, {
    quotaEstimates: [
      ...many,
      {
        ...estimates[0],
        status: "unavailable",
        observed_percentage_points: 90,
        estimated_percentage_points: null,
        reason: "missing_weighted_usage",
      },
      { ...estimates[0], window: "secondary" },
      { ...estimates[0], resets_at: estimates[0].resets_at + 86400 },
    ],
  });
  const groups = page.locator(".quota-estimate-interval");
  await expect(groups).toHaveCount(3);
  await expect(groups.first()).toContainText(
    "Scheduled reset: 26 Sept 2026, 18:30 IST",
  );
  await expect(groups.first()).toContainText(
    "Observed account increase: 10 pp",
  );
  await expect(groups.first()).toContainText("Conditional estimate: 5 pp");
  await expect(groups.first()).toContainText(
    "1,000 available intervals · 1 excluded intervals",
  );
  await expect(groups.first().getByRole("table")).toHaveCount(0);
  await groups
    .first()
    .getByText("Interval details (1,001)", { exact: true })
    .click();
  await expect(groups.first()).toContainText(
    "Showing latest 50 of 1,001 intervals",
  );
  await expect(
    groups.first().locator(".quota-interval-details tbody tr"),
  ).toHaveCount(50);
});

test("many account windows are loaded in explicit bounded batches", async ({
  page,
}) => {
  await fixture(page, {
    quotaEstimates: Array.from({ length: 45 }, (_, i) => ({
      ...estimates[0],
      email: `synthetic-${i}@example.test`,
    })),
  });
  await expect(page.locator(".quota-estimate-interval")).toHaveCount(20);
  await page.getByRole("button", { name: "Show more account windows" }).click();
  await expect(page.locator(".quota-estimate-interval")).toHaveCount(40);
  await page.getByRole("button", { name: "Show more account windows" }).click();
  await expect(page.locator(".quota-estimate-interval")).toHaveCount(45);
  await expect(
    page.getByRole("button", { name: "Show more account windows" }),
  ).toBeHidden();
});
