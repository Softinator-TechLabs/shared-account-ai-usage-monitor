import { test, expect } from "@playwright/test";
test("synthetic login, full session, prompt feedback and mobile navigation", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");
  await expect(
    page.getByRole("button", { name: "Explore as owner" }),
  ).toBeVisible();
  await page.screenshot({
    path: "../.impeccable/review/login-desktop.png",
    fullPage: true,
  });
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await expect(
    page.getByRole("heading", { name: "People", exact: true }),
  ).toBeVisible();
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.screenshot({
    path: "../.impeccable/review/people-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.screenshot({
    path: "../.impeccable/review/people-mobile.png",
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await page.getByLabel("Search sessions").fill("duplicate charge");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await page.locator(".session-link").first().click();
  await expect(
    page.getByText("Checkout ki duplicate charge bug fix karo.", {
      exact: false,
    }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Discuss message 1" }).click();
  await page.getByLabel("Review type").selectOption("prompt_rating");
  await page.getByLabel("Rating", { exact: true }).selectOption("4");
  await page
    .getByLabel("Comment or rationale")
    .fill(
      "Synthetic review: clear acceptance criteria; add the timeout reproduction.",
    );
  await page.getByRole("button", { name: "Post review" }).click();
  await expect(
    page
      .getByText(
        "Synthetic review: clear acceptance criteria; add the timeout reproduction.",
        { exact: true },
      )
      .first(),
  ).toBeVisible();
  await expect(page.locator("#feedback")).toContainText(
    "attached to message 1",
  );
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.screenshot({
    path: "../.impeccable/review/desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.screenshot({
    path: "../.impeccable/review/mobile.png",
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await page.getByRole("button", { name: "Accounts", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Shared accounts" }),
  ).toBeVisible();
});

test("owner creates an email user and user sets a password privately", async ({
  page,
  browser,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await page.getByRole("button", { name: "Add person" }).click();
  const email = `synthetic-${Date.now()}@example.com`;
  await page
    .locator("#person-dialog")
    .getByLabel("Name", { exact: true })
    .fill("Synthetic Employee");
  await page
    .locator("#person-dialog")
    .getByLabel("Email", { exact: true })
    .fill(email);
  const response = page.waitForResponse((r) =>
    r.url().endsWith("/user-invitations"),
  );
  await page.getByRole("button", { name: "Create setup link" }).click();
  const { url } = await (await response).json();
  await expect(page.getByText(email, { exact: true })).toBeVisible();
  const context = await browser.newContext();
  const member = await context.newPage();
  await member.goto(url);
  await member
    .getByLabel("New password", { exact: true })
    .fill("Synthetic browser passphrase");
  await member
    .getByRole("button", { name: "Set password", exact: true })
    .click();
  await expect(member.locator("#feedback")).toContainText("Password set");
  await member
    .locator("#password-login")
    .getByLabel("Email", { exact: true })
    .fill(email);
  await member
    .getByLabel("Password", { exact: true })
    .fill("Synthetic browser passphrase");
  await member.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(member.locator("#identity")).toContainText(email);
  await expect(member.getByRole("button", { name: "Add person" })).toBeHidden();
  await context.close();
});

test("switching people clears unrelated filters and the previous review target", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await page.getByLabel("Search sessions").fill("duplicate charge");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await page.locator(".session-link").first().click();
  await expect(page.locator("#conversation")).toContainText("sample-checkout");
  await page.getByRole("button", { name: "People", exact: true }).click();
  await page
    .locator(".directory tbody tr")
    .filter({ hasText: "bob" })
    .getByRole("button", { name: /sessions/ })
    .click();
  await expect(page.getByLabel("Search sessions")).toHaveValue("");
  await expect(page.locator("#conversation")).not.toContainText(
    "sample-checkout",
  );
  await expect(page.locator("#session-list")).toContainText(
    "sample-publishing",
  );
});

test("user setup submission locks the submit action and inline form restores focus", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await page.getByRole("button", { name: "Add person", exact: true }).click();
  const form = page.locator("#user-form");
  await form.getByLabel("Name", { exact: true }).fill("Pending fixture");
  await form.getByLabel("Email", { exact: true }).fill("pending@example.com");
  let release;
  const pending = new Promise((resolve) => {
    release = resolve;
  });
  let entered;
  const intercepted = new Promise((resolve) => {
    entered = resolve;
  });
  await page.route("**/api/v1/user-invitations", async (route) => {
    entered();
    await pending;
    await route.fulfill({
      status: 400,
      contentType: "application/json",
      body: "{}",
    });
  });
  await form.getByRole("button", { name: "Create setup link" }).click();
  await intercepted;
  try {
    await expect(
      form.getByRole("button", { name: "Create setup link" }),
    ).toBeDisabled();
    await expect(
      form.getByRole("button", { name: "Close add person" }),
    ).toBeEnabled();
  } finally {
    release();
  }
  await expect(
    form.getByRole("button", { name: "Create setup link" }),
  ).toBeEnabled();
  const panel = await page.locator("#person-dialog").boundingBox();
  const roster = await page.locator("#people-list").boundingBox();
  expect(panel.y).toBeLessThan(roster.y);
  await page.getByRole("button", { name: "Close add person" }).click();
  await expect(
    page.getByRole("button", { name: "Add person", exact: true }),
  ).toBeFocused();
});

test("session permalink survives reload", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await page.locator(".session-link").first().click();
  await expect(page).toHaveURL(/#session\//);
  const url = page.url();
  await page.reload();
  await expect(page.locator("#conversation .message").first()).toBeVisible();
  await expect(page).toHaveURL(url);
});

test("seven-day agent link opens a read-only session and can be revoked", async ({
  page,
  browser,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await expect(
    page.getByRole("heading", { name: "People", exact: true }),
  ).toBeVisible();
  const grant = await page.evaluate(async () => {
    const r = await fetch("/api/v1/debug-links", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ label: "Synthetic agent QA" }),
    });
    if (!r.ok) throw Error("grant");
    return r.json();
  });
  const context = await browser.newContext();
  const agent = await context.newPage();
  await agent.goto(grant.url);
  await expect(agent.locator("#policy-banner")).toContainText(
    "Agent access: read only",
  );
  await expect(
    agent.getByRole("button", { name: "Add person", exact: true }),
  ).toBeHidden();
  await agent.getByRole("button", { name: "Sessions", exact: true }).click();
  await agent.locator(".session-link").first().click();
  await expect(
    agent.getByRole("button", { name: "Post review", exact: true }),
  ).toBeDisabled();
  await page.evaluate(async (id) => {
    const r = await fetch("/api/v1/debug-links/" + id, { method: "DELETE" });
    if (!r.ok) throw Error("revoke");
  }, grant.id);
  await agent.reload();
  await expect(
    agent.getByRole("button", { name: "Explore as owner" }),
  ).toBeVisible();
  await context.close();
});
