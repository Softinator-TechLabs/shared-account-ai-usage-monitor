import { expect, test } from "@playwright/test";

test("download, docs and workspace are discoverable", async ({ page }) => {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Shared accounts. Clearer work." }),
  ).toBeVisible();
  const mac = page.getByRole("link", { name: /Download for Mac/ });
  await expect(mac).toHaveAttribute(
    "href",
    /v0\.7\.0-preview\/AI-Usage-Monitor-mac-arm64\.dmg$/,
  );
  await expect(page.getByRole("link", { name: "Intel Mac" })).toHaveAttribute(
    "href",
    /mac-amd64\.dmg$/,
  );
  await expect(
    page.getByRole("link", { name: /Open workspace/ }),
  ).toHaveAttribute("href", "https://usage.softinator.org/");
  await page.getByRole("link", { name: "Docs", exact: true }).first().click();
  await expect(page).toHaveURL(/\/docs\/?$/);
  await expect(page.getByText("The work behind the usage.")).toBeVisible();
  await page
    .locator("#VPContent")
    .getByRole("link", { name: "Get started", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: /Getting started/ }),
  ).toBeVisible();
});

test("site fits a narrow phone without sideways scrolling", async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Shared accounts. Clearer work." }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: /Download for Mac/ }),
  ).toBeVisible();
  const width = await page.evaluate(() => document.documentElement.scrollWidth);
  expect(width).toBeLessThanOrEqual(390);
});

test("docs disclose quota limits and current support", async ({ page }) => {
  await page.goto("/docs/guide/usage-and-limits");
  await expect(
    page.getByRole("heading", { name: /Usage and limits/ }),
  ).toBeVisible();
  await expect(
    page.getByText("the account", { exact: false }).first(),
  ).toBeVisible();
  await page.goto("/docs/guide/current-support");
  await expect(
    page.getByText("not notarized", { exact: false }).first(),
  ).toBeVisible();
  await page.goto("/docs/llms.txt");
  await expect(
    page.getByText("Shared Account AI Usage Monitor documentation"),
  ).toBeVisible();
});

test("documentation search finds the quota guide", async ({ page }) => {
  await page.goto("/docs/");
  await page.getByRole("button", { name: "Search" }).click();
  await page.getByRole("searchbox").fill("quota");
  await expect(
    page.getByRole("link", { name: /Usage and limits/ }).first(),
  ).toBeVisible();
});
