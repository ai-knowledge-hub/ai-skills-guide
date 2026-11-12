import { expect, test } from "@playwright/test";
import { loadRegistry, type ModuleKey } from "../lib/registry";
import { companionGuides, companionGuideUrl, companionResourceHref, matchesCompanionResource, relatedGuidesForResource } from "../lib/publicationLinks";

const pluginPath = "/plugins/engineering/code-maintenance-plugin";
const harnessUrl = "https://harness.ai-knowledge-hub.org/learn/ai-engineering/verify-small-code-change";

test("curated companions resolve to existing qualified resources with review provenance", async () => {
  const modules: ModuleKey[] = ["skills", "agents", "plugins", "tools"];
  const registries = await Promise.all(modules.map(module => loadRegistry(module)));
  const urls = companionGuides.map(companionGuideUrl);
  expect(new Set(urls).size).toBe(urls.length);
  for (const guide of companionGuides) {
    const url = new URL(companionGuideUrl(guide));
    expect(url.origin).toBe(guide.site === "publication" ? "https://ai-knowledge-hub.org" : "https://harness.ai-knowledge-hub.org");
    expect(guide.review.source).toBe(url.href);
    expect(guide.review.reviewedAt).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    const keys = guide.resources.map(resource => `${resource.module}:${resource.id}`);
    expect(new Set(keys).size).toBe(keys.length);
    for (const resource of guide.resources) {
      const entry = registries[modules.indexOf(resource.module)].skills.find(entry => entry.id === resource.id);
      expect(entry, `${resource.module}:${resource.id}`).toBeDefined();
      if (resource.version) expect(entry!.versions.some(version => version.version === resource.version)).toBe(true);
      expect(resource.context.length).toBeGreaterThan(20);
    }
  }
});

test("companion matching fences release-specific examples and keeps module namespaces distinct", async () => {
  const registry = await loadRegistry("plugins");
  const entry = registry.skills.find(entry => entry.id === "engineering/code-maintenance-plugin")!;
  const before = structuredClone(entry);
  expect(relatedGuidesForResource("plugins", entry.id, "0.2.0").map(companionGuideUrl)).toEqual([harnessUrl]);
  expect(relatedGuidesForResource("plugins", entry.id, "0.3.0")).toEqual([]);
  expect(relatedGuidesForResource("skills", entry.id, "0.2.0")).toEqual([]);
  expect(entry).toEqual(before);
  const expected = { skills: "/skills", agents: "/agents", plugins: "/plugins", tools: "/tools-mcp" };
  for (const module of Object.keys(expected) as ModuleKey[]) {
    const resource = { module, id: "same/example", context: "A useful reviewed relationship.", version: "1.0.0" };
    expect(companionResourceHref(resource)).toBe(`${expected[module]}/same/example`);
    expect(matchesCompanionResource(resource, module, "same/example", "1.0.0")).toBe(true);
    expect(matchesCompanionResource(resource, module === "agents" ? "plugins" : "agents", "same/example", "1.0.0")).toBe(false);
  }
});

test("the Harness companion has a reason and preserves the plugin's first action and evidence limits", async ({ page }) => {
  await page.goto("/");
  const paths = page.getByRole("region", { name: "From reading to practice" });
  await expect(paths.locator(`a[href="${harnessUrl}"]`)).toHaveText("Verify a small code change with an agent");
  await paths.locator(`a[href="${pluginPath}"]`).click();
  await expect(page).toHaveURL(pluginPath);
  await expect(page.locator(".detail-use-summary .usability-badge")).toHaveText("Not verified");
  await expect(page.getByRole("region", { name: "First use" })).toBeVisible();
  await expect(page.locator(".resource-install")).not.toHaveAttribute("open", "");
  await page.getByRole("link", { name: "Related reading", exact: true }).click();
  const related = page.locator("#related-reading");
  await expect(related.getByText("Harness", { exact: true })).toBeVisible();
  await expect(related.getByRole("link", { name: "Verify a small code change with an agent", exact: true })).toHaveAttribute("href", harnessUrl);
  await expect(related).toContainText("0.2.0 example prints a plan");
  await expect(related).toContainText("does not run the checks or authorize a merge");
  expect(await related.textContent()).not.toMatch(/[↗→]/);
  await page.locator(".resource-install > summary").click();
  await expect(page.locator(".install-runtime-option > summary").filter({ hasText: /^Codex$/ })).toBeVisible();
  await expect(page.locator("#usability")).toContainText("Current operational evidence for your environment is not shown here.");
});

test("unmapped resources in every module have no empty related panel or navigation dead end", async ({ page }) => {
  for (const path of ["/skills/engineering/implementation-strategy", "/agents/marketing/weekly-performance-supervisor", "/plugins/marketing/performance-reporting-plugin", "/tools-mcp/adtech/openai-ads-api-client"]) {
    await page.goto(path);
    await expect(page.locator(".detail-lead h1")).toBeVisible();
    await expect(page.getByRole("region", { name: "First use" })).toBeVisible();
    await expect(page.locator("#related-reading")).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Related reading", exact: true })).toHaveCount(0);
  }
});

test("related reading works on phones, by keyboard and without JavaScript", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false, viewport: { width: 320, height: 780 } });
  try {
    const page = await context.newPage();
    await page.goto(`http://127.0.0.1:3000${pluginPath}`);
    const jump = page.getByRole("link", { name: "Related reading", exact: true });
    await jump.focus();
    await jump.press("Enter");
    const link = page.locator(`#related-reading a[href="${harnessUrl}"]`);
    await expect(link).toBeInViewport();
    await link.focus();
    await link.press("Tab");
    await page.keyboard.press("Shift+Tab");
    await expect(link).toBeFocused();
    await expect(link).toHaveCSS("outline-style", "solid");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  } finally { await context.close(); }
});
