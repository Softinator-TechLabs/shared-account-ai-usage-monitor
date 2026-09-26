import { test, expect } from "@playwright/test";
test("enrolled device shows separate collection health, project context and unknown personal quota", async ({
  page,
}) => {
  const now = new Date().toISOString();
  await page.route("**/api/v1/people", (r) =>
    r.fulfill({
      json: [
        {
          id: "owner",
          name: "Synthetic Owner",
          role: "owner",
          active: true,
          sessions: 3,
          projects: [
            { name: "synthetic-repo", sessions: 3, last_day: "2026-09-26" },
          ],
          devices: [
            {
              id: "synthetic-token-digest",
              name: "synthetic-mac",
              os: "darwin",
              last_seen: "2026-01-01T00:00:00Z",
            },
          ],
        },
      ],
    }),
  );
  await page.route("**/api/v1/quota-observations", (r) =>
    r.fulfill({
      json: [
        {
          provider: "codex",
          profile: "default",
          device: "synthetic-mac",
          person: "owner",
          email: "shared@example.test",
          plan: "pro",
          observed_at: now,
          received_at: now,
          windows: [{ bucket: "codex", name: "primary", used_percent: 86 }],
        },
      ],
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await expect(
    page.getByRole("button", { name: "Add another device", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Quota connected · session collector stale", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByText("Personal quota share: unknown", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("86% account total", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("synthetic-repo · 3 sessions", { exact: true }),
  ).toBeVisible();
  for (const width of [1440, 1000, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBeTruthy();
    await expect(
      page.getByText("Personal quota share: unknown", { exact: true }),
    ).toBeVisible();
    await page.screenshot({
      path: `.local/people-evidence-${width}.png`,
      fullPage: true,
    });
  }
});

test("same device name on two people retains both observed account associations", async ({
  page,
}) => {
  const now = new Date().toISOString();
  await page.route("**/api/v1/people", (r) =>
    r.fulfill({
      json: ["alice", "bob"].map((id) => ({
        id,
        name: id,
        role: "member",
        active: true,
        sessions: 0,
        projects: [],
        devices: [
          { id: `synthetic-${id}`, name: "mac", os: "darwin", last_seen: null },
        ],
      })),
    }),
  );
  await page.route("**/api/v1/quota-observations", (r) =>
    r.fulfill({
      json: ["alice", "bob"].map((person) => ({
        provider: "codex",
        profile: "default",
        device: "mac",
        person,
        email: "shared@example.test",
        observed_at: now,
        received_at: now,
        windows: [{ name: "primary", used_percent: 86 }],
      })),
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await expect(
    page.getByText("shared@example.test", { exact: true }),
  ).toHaveCount(2);
  await expect(
    page.getByText("86% account total", { exact: true }),
  ).toHaveCount(2);
  await expect(
    page.getByText("Personal quota share: unknown", { exact: true }),
  ).toHaveCount(2);
});
