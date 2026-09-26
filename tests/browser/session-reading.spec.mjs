import { test, expect } from "@playwright/test";

test("a session explains the request, recorded usage and links without opening JSON", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/");
  await page.getByRole("button", { name: "Explore as owner" }).click();
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await page.route("**/api/v1/sessions/*", async (route) => {
    if (new URL(route.request().url()).pathname.endsWith("/reviews"))
      return route.continue();
    const response = await route.fetch(),
      json = await response.json();
    Object.assign(json, {
      project: "synthetic-journal",
      branch: "fix/synthetic-links",
      raw: {
        display_name: "Fix synthetic PDF links",
        cwd: "/workspace/synthetic-journal",
        has_total_output_tokens: true,
        total_output_tokens: 321,
        has_peak_context_tokens: true,
        peak_context_tokens: 12345,
      },
    });
    json.messages = [
      {
        ordinal: 0,
        role: "user",
        content:
          "Initial Hinglish request: is link ko fix karo https://example.com/paper?a=1&b=2 .",
        model: "synthetic-model",
      },
      {
        ordinal: 1,
        role: "assistant",
        content:
          '[{"type":"text","text":"Verified synthetic result. <script>alert(1)</script>"}]',
        model: "synthetic-model",
        raw: {
          tool_calls: [
            {
              tool_use_id: "synthetic-call",
              tool_name: "Edit",
              input_json:
                '{"file_path":"/workspace/synthetic-journal/links.ts","old_string":"before","new_string":"after"}',
            },
          ],
        },
      },
      {
        ordinal: 2,
        role: "assistant",
        content: '{"outcome":"Completed","rationale":"Synthetic test passed"}',
        model: "synthetic-model",
      },
    ];
    await route.fulfill({ response, json });
  });
  await page.locator(".session-open").first().click();
  await expect(
    page.getByRole("heading", { name: "Fix synthetic PDF links" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Initial prompt", exact: true }),
  ).toBeVisible();
  await expect(page.locator(".initial-prompt")).toContainText(
    "Initial Hinglish request",
  );
  await expect(page.locator(".session-context")).toContainText(
    "fix/synthetic-links",
  );
  await expect(page.locator(".session-context")).toContainText("321");
  await expect(page.locator(".session-context")).toContainText("12,345");
  await expect(page.locator(".latest-response")).toContainText(
    "Synthetic test passed",
  );
  await expect(page.locator(".latest-response pre")).toHaveCount(0);
  await page
    .getByRole("button", { name: "Copy session link", exact: true })
    .click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    page.url(),
  );
  const link = page.locator(".initial-prompt").getByRole("link");
  await expect(link).toHaveAttribute(
    "href",
    "https://example.com/paper?a=1&b=2",
  );
  await page
    .locator(".initial-prompt")
    .getByRole("button", { name: "Copy URL", exact: true })
    .click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    "https://example.com/paper?a=1&b=2",
  );
  await page.locator(".local-viewer > summary").click();
  await page
    .getByLabel("AgentsView address on this computer")
    .fill("http://127.0.0.1:18087");
  await expect(
    page.getByRole("link", { name: "Open local session" }),
  ).toHaveAttribute("href", /http:\/\/127\.0\.0\.1:18087\/sessions\//);
  await page.locator(".files-summary > summary").click();
  await expect(page.locator(".work-files")).toContainText("links.ts");
  await page.locator(".transcript > summary").click();
  await page
    .locator(".message")
    .nth(1)
    .locator(".full-message > summary")
    .click();
  await expect(
    page.locator(".message").nth(1).locator(".message-content"),
  ).toContainText("<script>alert(1)</script>");
  await expect(page.locator("#conversation script")).toHaveCount(0);
  await page.locator(".local-viewer > summary").click();
  await page.locator(".transcript > summary").click();
  await page.screenshot({
    path: "../.impeccable/review/session-reading-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await page.screenshot({
    path: "../.impeccable/review/session-reading-mobile.png",
    fullPage: true,
  });
  const permalink = page.url();
  await page.getByRole("button", { name: "People", exact: true }).click();
  await expect(page).toHaveURL(/#people$/);
  await page.goto(permalink);
  await expect(
    page.getByRole("heading", { name: "Fix synthetic PDF links" }),
  ).toBeVisible();
  // Browser clipboard denial still leaves exact content selectable and copyable.
  await page.evaluate(() =>
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: () => Promise.reject(new Error("denied")) },
      configurable: true,
    }),
  );
  await page
    .locator(".initial-prompt")
    .getByRole("button", { name: "Copy initial prompt", exact: true })
    .click();
  await expect(
    page.getByRole("textbox", {
      name: "Copy initial prompt value",
      exact: true,
    }),
  ).toHaveValue(/Initial Hinglish request/);
});
