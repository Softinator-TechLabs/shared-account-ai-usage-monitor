import { test, expect } from "@playwright/test";
test("activity charts and Mac setup replace the invitation-only action", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await expect(
    page.getByRole("heading", { name: "Workspace activity" }),
  ).toBeVisible();
  await expect(page.getByText("One account. More context.")).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "Sessions started", exact: true }),
  ).toBeVisible();
  await page.getByLabel("Activity period").selectOption("30");
  await expect(page.locator(".activity-chart svg")).toHaveAttribute(
    "aria-label",
    /Asia\/Kolkata/,
  );
  const connect = page.getByRole("button", { name: "Connect device" }).first();
  let downloads = 0;
  page.on("download", () => downloads++);
  await connect.click();
  const dialog = page.getByRole("dialog", { name: "Connect a Mac" });
  await expect(dialog).toBeVisible();
  expect(downloads).toBe(0);
  await expect(
    dialog.getByRole("link", { name: "Download for Apple silicon" }),
  ).toHaveAttribute("href", /AI-Usage-Monitor-mac-arm64.zip$/);
  const response = page.waitForResponse((r) =>
    r.url().endsWith("/invitations"),
  );
  const file = page.waitForEvent("download");
  await dialog
    .getByRole("button", { name: "Download connection file" })
    .click();
  expect((await file).suggestedFilename()).toBe(
    "Connect AI Usage Monitor.aiusage",
  );
  const invitation = await (await response).json();
  expect(invitation.server).toMatch(/^http:\/\/127\.0\.0\.1/);
  expect(invitation.person).toBeTruthy();
  expect(invitation.policy.version).toBeGreaterThan(0);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(dialog).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await page.keyboard.press("Escape");
  await expect(dialog).not.toBeVisible();
  await expect(connect).toBeFocused();
});

test("large prompts stay collapsed behind a bounded work summary", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await page.route("**/api/v1/sessions/*", async (route) => {
    if (new URL(route.request().url()).pathname.endsWith("/reviews"))
      return route.continue();
    const response = await route.fetch();
    const json = await response.json();
    if (json.messages?.length)
      json.messages[0].content =
        "Synthetic long prompt " + "context ".repeat(20000);
    await route.fulfill({ response, json });
  });
  await page.locator(".session-link").first().click();
  await expect(page.locator(".work-summary")).toBeVisible();
  await expect(page.locator(".transcript")).not.toHaveAttribute("open", "");
  await expect(page.locator(".message-content").first()).toBeHidden();
  await page.locator(".transcript > summary").click();
  await page.locator(".full-message > summary").first().click();
  await expect(page.locator(".message-content").first()).toBeVisible();
  expect(
    await page
      .locator(".transcript-messages")
      .evaluate((n) => n.clientHeight <= innerHeight * 0.62 + 2),
  ).toBeTruthy();
});
