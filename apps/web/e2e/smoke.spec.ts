import { expect, test } from "@playwright/test";

const sampleSkillPath = "/skills/marketing/meta-google-weekly-performance-review";

test("home route smoke", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Browse Skills" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Explore catalog" })).toBeVisible();
});

test("skills route filter interaction smoke", async ({ page }) => {
  await page.goto("/skills");
  const runtimeTrigger = page.getByRole("button", { name: "Runtime" });

  await runtimeTrigger.click();
  await page.getByRole("button", { name: "codex" }).click();
  await page.getByRole("button", { name: "Apply Filters" }).click();

  await expect(runtimeTrigger).toContainText("codex");
  await expect(page.getByText(/\d+ result\(s\)/)).toBeVisible();
});

test("skill detail copy-button smoke", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async () => {} }
    });
  });

  await page.goto(sampleSkillPath);
  const copyButtons = page.locator(".copy-button");

  await expect(copyButtons).toHaveCount(3);
  await expect(copyButtons.first()).toHaveText("Copy");
  await copyButtons.first().click();
  await expect(copyButtons.first()).toHaveText("Copied");
});
