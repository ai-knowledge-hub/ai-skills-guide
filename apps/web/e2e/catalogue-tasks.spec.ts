import { expect, test } from "@playwright/test";

const weekly = "/skills/marketing/meta-google-weekly-performance-review";
const agent = "/agents/marketing/weekly-performance-supervisor";
const plugin = "/plugins/engineering/code-maintenance-plugin";
type ClipboardProbeWindow = Window & { rejectCopy?: boolean; copiedInstructions?: string };

// A short layout viewport reproduces the reading-space constraint; the dated
// acceptance report records actual browser 200%/400% zoom separately.
for (const viewport of [{ width: 320, height: 180 }, { width: 1440, height: 450 }]) {
  test(`short viewport ${viewport.width}x${viewport.height} lets the masthead scroll out of the reading area`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.goto("/skills");
    await page.evaluate(() => scrollTo(0, 500));
    await expect.poll(() => page.locator(".site-header").evaluate(el => el.getBoundingClientRect().bottom), { timeout: 1500 }).toBeLessThanOrEqual(0);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    if (viewport.width === 320) {
      await page.locator(".mobile-menu summary").click();
      const tools = page.getByRole("navigation", { name: "Catalogue", exact: true }).getByRole("link", { name: "Tools & MCP", exact: true });
      await tools.focus();
      await expect(tools).toBeInViewport();
      await tools.press("Enter");
      await expect(page).toHaveURL(/\/tools-mcp$/);
    }
  });
}

test("short-window filters keep their final action reachable by keyboard", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 180 });
  await page.goto("/skills");
  const runtime = page.getByRole("button", { name: "Runtime", exact: true });
  await runtime.press("Enter");
  for (let step = 0; step < 5; step++) await page.keyboard.press("Tab");
  const done = page.locator(".filter-menu").getByRole("button", { name: "Done", exact: true });
  await expect(done).toBeFocused();
  expect(await done.evaluate(el => {
    const box = el.getBoundingClientRect();
    return box.top >= 0 && box.bottom <= innerHeight
      && el.contains(document.elementFromPoint(box.x + box.width / 2, box.y + box.height / 2));
  })).toBe(true);
  await done.press("Escape");
  await expect(runtime).toBeFocused();
  await expect(page.locator(".filter-menu")).toHaveCount(0);
});

test("instruction copy controls truthfully disclose their no-JavaScript limit for skills and agents", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  try {
    const page = await context.newPage();
    for (const [path, filename] of [[weekly, "SKILL.md"], [agent, "AGENT.md"]]) {
      await page.goto(`http://127.0.0.1:3000${path}`);
      if (filename === "SKILL.md") await page.getByText("Read full instructions", { exact: true }).click();
      else {
        await page.getByText("View agent workflow", { exact: true }).click();
        await page.getByText("Read agent instructions (AGENT.md)", { exact: true }).click();
      }
      await expect(page.getByRole("button", { name: `Copy full ${filename} text` })).toBeDisabled();
      await expect(page.getByText("Select the instructions below to copy them. The copy button requires JavaScript.", { exact: true })).toBeVisible();
      await expect(page.getByText(filename === "SKILL.md" ? "Never fabricate numbers." : "Never bypass critical QA failures.", { exact: true })).toBeVisible();
    }
  } finally { await context.close(); }
});

test("a reader finds weekly review, copies its procedure and separates the helper's evidence", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: async (text: string) => {
      if ((window as ClipboardProbeWindow).rejectCopy) throw new Error("Clipboard denied");
      (window as ClipboardProbeWindow).copiedInstructions = text;
    } } });
  });
  await page.goto("/skills");
  await page.getByRole("button", { name: "Switch to light mode" }).click();
  await page.getByRole("searchbox", { name: "Search the catalogue" }).fill("weekly performance");
  await page.getByRole("button", { name: "Runtime", exact: true }).click();
  const claude = page.getByRole("checkbox", { name: "claude", exact: true });
  await claude.check();
  await claude.press("Escape");
  await expect(page.getByRole("button", { name: "Runtime", exact: true })).toBeFocused();
  await page.getByRole("heading", { name: "Meta + Google Weekly Performance Review", exact: true }).click();
  await expect(page).toHaveURL(weekly);
  const first = page.getByRole("region", { name: "First use" });
  await expect(first).toContainText("Meta and Google Ads account IDs");
  await expect(first).toContainText("Executive summary and KPI table with period deltas");
  await expect(first).toContainText("chat that supports pasted instructions");
  await page.getByText("Read full instructions", { exact: true }).click();
  const copy = page.getByRole("button", { name: "Copy full SKILL.md text" });
  await copy.focus();
  await copy.press("Enter");
  await expect(page.getByText("Full instructions copied.", { exact: true })).toBeVisible();
  const source = await page.evaluate(() => (window as ClipboardProbeWindow).copiedInstructions as string);
  expect(source).toContain("# Meta/Google Weekly Performance Review");
  expect(source).toContain("Flag low confidence when conversion volume is low.");
  await page.evaluate(() => { (window as ClipboardProbeWindow).rejectCopy = true; });
  await copy.press("Enter");
  await expect(page.getByRole("textbox", { name: "SKILL.md source text" })).toHaveValue(source);
  await expect(page.getByText("Clipboard unavailable. Select and copy the source below.", { exact: true })).toBeVisible();
  await expect(page.locator("#usability")).toContainText("Packaged executable helpers");
  await expect(page.locator("#usability").getByText("Not verified", { exact: true })).toBeVisible();
  await page.goBack();
  await expect(page.getByRole("searchbox")).toHaveValue("weekly performance");
  await expect(page).toHaveURL(/runtime=claude/);
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});

test("a reader compares a reference composition with an installable bundle and its optional guide", async ({ page }) => {
  await page.goto("/plugins");
  await page.getByRole("searchbox").fill("Performance Reporting Plugin");
  await page.getByRole("heading", { name: "Performance Reporting Plugin", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Intended inputs" })).toBeVisible();
  await page.getByText("View reference contents", { exact: true }).click();
  await expect(page.getByRole("heading", { name: "Referenced skills" })).toBeVisible();
  await page.locator(".resource-install > summary").click();
  await expect(page.getByRole("heading", { name: "Reference composition" })).toBeVisible();
  await expect(page.locator("pre.install")).toHaveCount(0);
  await page.goBack();
  await page.getByRole("searchbox").fill("Code Maintenance Plugin");
  await page.getByRole("heading", { name: "Code Maintenance Plugin", exact: true }).click();
  await page.getByText("Inspect offline sample", { exact: true }).click();
  const first = page.getByRole("region", { name: "First use" });
  await expect(first).toContainText("installed package directory after file installation");
  await expect(first).toContainText("node scripts/first_use.mjs examples/first-use-input.json");
  await expect(first).toContainText("bounded plan on standard output");
  await page.locator(".resource-install > summary").click();
  await page.locator(".install-runtime-option > summary").filter({ hasText: /^Generic/ }).click();
  await expect(page.getByRole("region", { name: "generic installation command", exact: true })).toContainText("--target ./my-agent/plugins");
  await expect(page.locator(".install-runtime-option[open]")).toContainText("./my-agent/skills");
  await expect(page.locator("#usability")).toContainText("Markdown hook is advisory");
  await expect(page.locator("#related-reading")).toContainText("does not run the checks or authorize a merge");
});

test("a reader distinguishes offline samples, configurable reads and unavailable writes before selecting a runtime", async ({ page }) => {
  await page.goto("/plugins/marketing/chatgpt-advertising-experiment-plugin");
  await page.getByText("Inspect offline sample", { exact: true }).click();
  const first = page.getByRole("region", { name: "First use" });
  await expect(first).toContainText("python3 scripts/run_mock_lab.py");
  await expect(first).toContainText("with no network calls");
  await expect(first).toContainText("Credentials are required separately for live reads");
  const capabilities = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "Capability readiness", exact: true }) });
  for (const name of ["Mock lab", "Live read"]) {
    await expect(capabilities.locator("section").filter({ has: page.getByRole("heading", { name, exact: true }) }).getByText("Not verified", { exact: true })).toBeVisible();
  }
  await expect(capabilities).toContainText("Live writes unavailable: no supported OpenAI Ads mutation adapter is included in this release.");
  await page.goto("/tools-mcp");
  await page.getByRole("searchbox").fill("GA4");
  await page.getByRole("heading", { name: "GA4 MCP Connector", exact: true }).click();
  await page.getByRole("link", { name: "Review setup requirements", exact: true }).click();
  await expect(page.locator("#usability")).toContainText("GA4_CREDENTIAL runtime binding");
  await page.locator(".resource-install > summary").click();
  await page.locator(".install-runtime-option > summary").filter({ hasText: /^Claude$/ }).click();
  const command = page.getByRole("region", { name: "claude installation command", exact: true });
  await expect(command).toContainText("--entry analytics/ga4-mcp-connector@0.2.0 --runtime claude --execution-runtime node22");
  await expect(page.locator(".install-runtime-option[open]")).toContainText("Confirm that your app discovers this location");
});

const layoutRoutes = ["/", "/skills", "/agents", "/plugins", "/tools-mcp", weekly, agent, plugin,
  "/tools-mcp/adtech/openai-ads-api-client", "/skills?q=__no_catalogue_match__", "/plugins/missing/entry"];

for (const theme of ["dark", "light"] as const) for (const width of [320, 390, 1440]) {
  test(`complete catalogue surfaces remain readable at ${width}px in ${theme} mode`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.addInitScript(theme => localStorage.setItem("skills-hub-theme", theme), theme);
    for (const path of layoutRoutes) {
      await page.goto(path);
      await expect(page.locator("main h1")).toBeVisible();
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
      if (["/skills", "/agents", "/plugins", "/tools-mcp"].includes(path)) {
        const runtime = page.getByRole("button", { name: "Runtime", exact: true });
        await runtime.press("Enter");
        await expect(page.locator(".filter-menu")).toBeVisible();
        await runtime.press("Escape");
        await expect(runtime).toBeFocused();
      }
      if ([weekly, agent, plugin, "/tools-mcp/adtech/openai-ads-api-client"].includes(path)) {
        await page.locator(".resource-install > summary").click();
        const generic = page.locator(".install-runtime-option > summary").filter({ hasText: /^Generic/ });
        await generic.press("Enter");
        const code = page.getByRole("region", { name: "generic installation command", exact: true });
        await expect(code).toBeVisible();
        expect(await code.evaluate(el => el.scrollWidth <= el.clientWidth + 1)).toBe(true);
        if (path === weekly) await page.getByText("Read full instructions", { exact: true }).click();
        if (path === agent) {
          await page.getByText("View agent workflow", { exact: true }).click();
          await page.getByText("Read agent instructions (AGENT.md)", { exact: true }).click();
        }
      }
      if (width < 761) {
        const menu = page.locator(".mobile-menu summary");
        await menu.press("Enter");
        await expect(page.locator(".mobile-menu")).toHaveAttribute("open", "");
        await menu.press("Escape");
        await expect(menu).toBeFocused();
      }
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), path).toBe(true);
      const clipped = await page.locator('.family-nav a, .theme-toggle, .mobile-menu summary, .filter-trigger, .copy-button, .actions a').evaluateAll(elements => elements.filter(el => {
        const box = el.getBoundingClientRect();
        return box.width > 0 && box.height > 0 && (box.left < -1 || box.right > document.documentElement.clientWidth + 1);
      }).map(el => el.textContent));
      expect(clipped, path).toEqual([]);
    }
  });
}

test("reduced-motion preference keeps navigation and first-use controls operable", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  expect(await page.evaluate(() => matchMedia("(prefers-reduced-motion: reduce)").matches)).toBe(true);
  await expect(page.getByRole("link", { name: "Explore skills", exact: true })).toHaveCSS("transition-duration", "0s");
  await page.locator(".mobile-menu summary").press("Enter");
  await page.getByRole("navigation", { name: "Catalogue", exact: true }).getByRole("link", { name: "Agents", exact: true }).press("Enter");
  await page.getByRole("searchbox").fill("Weekly Performance Supervisor");
  await page.getByRole("heading", { name: "Weekly Performance Supervisor", exact: true }).click();
  await page.locator(".detail-disclosure > summary").filter({ hasText: "View agent workflow" }).press("Enter");
  await expect(page.getByRole("heading", { name: "What it coordinates", exact: true })).toBeVisible();
});
