import { expect, test } from "@playwright/test";

const detail = "/skills/marketing/meta-google-weekly-performance-review";
const routes = ["/", "/skills", detail];

test("direct entry identifies Skills and separates family destinations from catalogue routes", async ({ page }) => {
  for (const route of ["/", "/skills", "/agents", "/plugins", "/tools-mcp", detail]) {
    await page.goto(route);
    await expect(page).toHaveTitle(/Skills.*AI Knowledge Hub/);
    await expect(page.locator(".wordmark")).toHaveAttribute("aria-label", "AI Knowledge Hub — Skills home");
    const family = page.getByRole("navigation", { name: "AI Knowledge Hub sites", exact: true });
    await expect(family.getByRole("link", { name: "Publication", exact: true })).toHaveAttribute("href", "https://ai-knowledge-hub.org/");
    await expect(family.getByRole("link", { name: "Harness", exact: true })).toHaveAttribute("href", "https://harness.ai-knowledge-hub.org/");
    await expect(family.getByRole("link", { name: "Skills", exact: true })).toHaveAttribute("aria-current", "true");
    await expect(family.getByRole("link", { name: "Skills", exact: true })).toHaveAttribute("href", "/");
    const catalogue = page.getByRole("navigation", { name: "Catalogue", exact: true });
    await expect(catalogue.getByRole("link")).toHaveCount(4);
    await expect(catalogue.getByRole("link", { name: "Skills", exact: true })).toHaveAttribute("href", "/skills");
    const footer = page.getByRole("navigation", { name: "Explore AI Knowledge Hub", exact: true });
    await expect(footer).toContainText("Reusable instructions, templates, packages and integrations.");
    await expect(footer).toContainText("Independent research, explanations and practical experiments.");
    await expect(footer).toContainText("Engineering guides, visualizers and worked examples.");
  }
});

for (const theme of ["light", "dark"]) {
  test(`the family row reflows independently of the masthead and local menu in ${theme} mode`, async ({ page }) => {
    await page.goto("/");
    if (await page.locator("html").getAttribute("data-theme") !== theme) {
      await page.getByRole("button", { name: `Switch to ${theme} mode` }).click();
    }
    for (const width of [320, 390, 761, 1440]) {
      await page.setViewportSize({ width, height: 900 });
      for (const route of routes) {
        await page.goto(route);
        await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
        const family = page.getByRole("navigation", { name: "AI Knowledge Hub sites", exact: true });
        for (const label of ["Publication", "Skills", "Harness"]) {
          await expect(family.getByRole("link", { name: label, exact: true })).toBeInViewport();
        }
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${route}: ${width}px ${theme}`).toBe(true);
        if (width <= 760) {
          await page.locator(".mobile-menu summary").click();
          await expect(family).toBeInViewport();
          await expect(page.getByRole("navigation", { name: "Catalogue", exact: true }).getByRole("link", { name: "Tools & MCP", exact: true })).toBeInViewport();
        }
      }
    }
  });
}

test("keyboard users can choose either family or catalogue links and recover menu focus", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(detail);
  const menu = page.locator(".mobile-menu summary");
  await menu.focus();
  await menu.press("Enter");
  await page.keyboard.press("Tab");
  const skills = page.getByRole("navigation", { name: "Catalogue", exact: true }).getByRole("link", { name: "Skills", exact: true });
  await expect(skills).toBeFocused();
  await expect(skills).toHaveCSS("outline-style", "solid");
  await skills.press("Escape");
  await expect(menu).toBeFocused();
  await expect(page.locator(".mobile-menu")).not.toHaveAttribute("open", "");
  await page.keyboard.press("Tab");
  const publication = page.getByRole("navigation", { name: "AI Knowledge Hub sites", exact: true }).getByRole("link", { name: "Publication", exact: true });
  await expect(publication).toBeFocused();
  await expect(publication).toHaveCSS("outline-style", "solid");
  await menu.focus();
  await menu.press("Enter");
  await skills.click();
  await expect(page).toHaveURL(/\/skills$/);
  await expect(page.locator(".mobile-menu")).not.toHaveAttribute("open", "");
  await expect(page.getByRole("navigation", { name: "AI Knowledge Hub sites", exact: true }).getByRole("link", { name: "Skills", exact: true })).toHaveAttribute("aria-current", "true");
});

test("family links and local discovery work without JavaScript", async ({ browser }) => {
  for (const width of [320, 1440]) {
    const context = await browser.newContext({ javaScriptEnabled: false, viewport: { width, height: 900 } });
    const page = await context.newPage();
    await page.goto(`http://127.0.0.1:3000${detail}`);
    const family = page.getByRole("navigation", { name: "AI Knowledge Hub sites", exact: true });
    await expect(family.getByRole("link", { name: "Publication", exact: true })).toBeVisible();
    await expect(family.getByRole("link", { name: "Harness", exact: true })).toBeVisible();
    if (width === 320) await page.locator(".mobile-menu summary").click();
    await page.getByRole("navigation", { name: "Catalogue", exact: true }).getByRole("link", { name: "Skills", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Skills Catalog" })).toBeVisible();
    await page.locator(".catalog-card-link").first().click();
    await expect(page.locator(".detail-lead h1")).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await context.close();
  }
});
