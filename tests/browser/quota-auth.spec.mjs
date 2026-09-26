import { test, expect } from "@playwright/test";
test("signed-out login and setup have no application sidebar on desktop or mobile", async ({
  page,
}) => {
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/#settings");
    await expect(page.locator("#login")).toBeVisible();
    await expect(page.locator("body > header")).toBeHidden();
    expect(
      await page.locator("main").evaluate((el) => el.getBoundingClientRect().x),
    ).toBeLessThan(60);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBeTruthy();
  }
  await page.goto("/#setup/synthetic-invalid");
  await expect(page.locator("body > header")).toBeHidden();
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await expect(page.locator("body > header")).toBeVisible();
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await expect(page.locator("body > header")).toBeHidden();
});
test("native quota cards show observed usage, resets and device history without summing", async ({
  page,
}) => {
  const now = new Date().toISOString();
  const row = {
    provider: "codex",
    plan: "pro",
    email: "synthetic-quota@example.test",
    profile: "default",
    person: "alice",
    device: "synthetic-mac",
    observed_at: now,
    windows: [
      {
        bucket: "codex",
        name: "primary",
        used_percent: 48,
        window_duration_mins: 10080,
        resets_at: Math.floor(Date.now() / 1000) + 3600,
      },
    ],
  };
  await page.route("**/api/v1/quota-observations", (r) =>
    r.fulfill({
      json: [
        row,
        { ...row, device: "synthetic-pc" },
        {
          ...row,
          email: "",
          device: "synthetic-offline",
          windows: [],
          error: "unavailable",
        },
      ],
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await page.getByRole("button", { name: "Accounts", exact: true }).click();
  await expect(page.locator(".native-quota")).toHaveCount(1);
  await expect(page.locator(".quota-read-error")).toContainText(
    "synthetic-offline",
  );
  await expect(page.locator(".quota-amount strong")).toHaveText("48%");
  await expect(page.getByText("52% remaining at that reading")).toBeVisible();
  await page.getByText("Devices & history · 2 profiles").click();
  await expect(
    page.getByText("synthetic-pc · default · collector: alice"),
  ).toBeVisible();
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBeTruthy();
    await page.screenshot({
      path: `.local/quota-${width}.png`,
      fullPage: true,
    });
  }
});
