import { expect, test } from "@playwright/test";

const sampleSkillPath = "/skills/marketing/meta-google-weekly-performance-review";
const sampleAgentPath = "/agents/marketing/weekly-performance-supervisor";
const samplePluginPath = "/plugins/marketing/performance-reporting-plugin";
const sampleEngineeringPluginPath = "/plugins/engineering/code-maintenance-plugin";
const sampleSecurityPluginPath = "/plugins/security/runtime-safety-plugin";
const sampleHarnessPluginPath = "/plugins/agentops/harness-governance-plugin";
const sampleAdvertisingExperimentPluginPath = "/plugins/marketing/chatgpt-advertising-experiment-plugin";
const sampleToolPath = "/tools-mcp/analytics/ga4-mcp-connector";

test("home route smoke", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Skills", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Agents", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Plugins", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Tools & MCP", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Explore skills" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Know what you are installing" })).toBeVisible();
  await expect(page.getByText("Not verified", { exact: true })).toBeVisible();
});

test("skills route filter interaction smoke", async ({ page }) => {
  await page.goto("/skills");
  await expect(page.getByRole("heading", { name: "Skills Catalog" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Runtime" })).toBeVisible();
  await expect(page.getByText(/\d+ result\(s\)/)).toBeVisible();
});

test("skill detail copy-button smoke", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async (value: string) => { (window as Window & { __copiedCommand?: string }).__copiedCommand = value; } }
    });
  });

  await page.goto(sampleSkillPath);
  await expect(page.getByRole("heading", { name: "Operational Summary" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "How You Can Use This" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Outputs" })).toBeVisible();
  await expect(page.getByText("may-run-local-verification")).toBeVisible();
  await expect(page.locator("#usability").getByText("Documentation", { exact: true })).toBeVisible();
  await expect(page.getByText("Instruction package only", { exact: false })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Packaged executable helpers" })).toBeVisible();
  await expect(page.getByText("scripts/compute_metrics.py", { exact: true })).toBeVisible();
  await expect(page.getByText("Not verified", { exact: true })).toBeVisible();
  const copyButtons = page.locator(".copy-button");

  await expect(copyButtons).toHaveCount(3);
  await expect(copyButtons.first()).toHaveText("Copy");
  const command = await page.locator("pre.install").first().textContent();
  await copyButtons.first().click();
  await expect(copyButtons.first()).toHaveText("Copied");
  await expect.poll(() => page.evaluate(() => (window as Window & { __copiedCommand?: string }).__copiedCommand)).toBe(command);
});

test("agents route and detail smoke", async ({ page }) => {
  await page.goto("/agents");
  await expect(page.getByRole("heading", { name: "Agents Catalog" })).toBeVisible();
  await expect(page.getByText(/\d+ result\(s\)/)).toBeVisible();

  await page.goto(sampleAgentPath);
  await expect(page.getByRole("heading", { name: "Weekly Performance Supervisor" })).toBeVisible();
  await expect(page.getByText("Marketing Agents / Performance")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Operational Summary" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Coordinates" })).toBeVisible();
  await expect(page.getByText("semi-autonomous")).toBeVisible();
  const usability = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "How You Can Use This" }) }).first();
  await expect(usability.getByText("Setup required", { exact: true }).first()).toBeVisible();
  await expect(usability.getByText("orchestrator", { exact: false })).toBeVisible();
  await expect(usability.getByText("GA4 and BigQuery credentials", { exact: false })).toBeVisible();
  await expect(page.getByText("Install this agent", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Install (Codex)" })).toBeVisible();
  await expect(page.locator(".copy-button")).toHaveCount(3);
  await expect(page.locator("pre.install").first()).toContainText("--module agents");
});

test("plugins route and detail smoke", async ({ page }) => {
  await page.goto("/plugins");
  await expect(page.getByRole("heading", { name: "Plugins Catalog" })).toBeVisible();
  await expect(page.getByText(/\d+ result\(s\)/)).toBeVisible();
  const performancePluginCard = page.locator(".catalog-card").filter({ hasText: "Performance Reporting Plugin" }).first();
  const packageFileLink = performancePluginCard.getByRole("link", { name: "View package file" });
  await expect(packageFileLink).toBeVisible();
  await expect(packageFileLink).toHaveAttribute(
    "href",
    "https://github.com/ai-knowledge-hub/ai-skills-guide/blob/main/plugins/marketing/performance-reporting-plugin/plugin.yaml"
  );
  await expect(page.getByText("Performance Reporting Plugin")).toBeVisible();

  await page.goto(samplePluginPath);
  await expect(page.getByRole("heading", { name: "Performance Reporting Plugin" })).toBeVisible();
  await expect(page.getByText("Marketing Plugins / Reporting")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Declared Composition" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Referenced Skills" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Reference template" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Install (Codex)" })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Packaged Hooks" })).toBeVisible();
  await expect(page.getByRole("link", { name: "marketing/meta-google-weekly-performance-review" })).toBeVisible();
});

test("new plugin domains smoke", async ({ page }) => {
  await page.goto(sampleEngineeringPluginPath);
  await expect(page.getByRole("heading", { name: "Code Maintenance Plugin" })).toBeVisible();
  await expect(page.getByText("Engineering Plugins / Maintenance")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Installed Skills" })).toBeVisible();
  await expect(page.getByRole("link", { name: "engineering/code-change-verification" })).toBeVisible();
  await expect(page.getByRole("link", { name: "verification-before-complete" })).toBeVisible();

  await page.goto(sampleSecurityPluginPath);
  await expect(page.getByRole("heading", { name: "Runtime Safety Plugin" })).toBeVisible();
  await expect(page.getByText("Security Plugins / Runtime Safety")).toBeVisible();
  const securityRequirements = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "Requirements" }) }).first();
  await expect(securityRequirements.getByRole("heading", { name: "Requirements" })).toBeVisible();
  await expect(securityRequirements.getByText("human-review-for-live-remediation")).toBeVisible();
  await expect(page.getByRole("link", { name: "quarantine-suspicious-instructions" })).toBeVisible();

  await page.goto(sampleHarnessPluginPath);
  await expect(page.getByRole("heading", { name: "Harness Governance Plugin" })).toBeVisible();
  await expect(page.getByText("AgentOps Plugins / Governance")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Installed Skills" })).toBeVisible();
  await expect(page.getByRole("link", { name: "agentops/harness-run-reflection" })).toBeVisible();
  const harnessRequirements = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "Requirements" }) }).first();
  await expect(harnessRequirements.getByText("human-approval-for-harness-update")).toBeVisible();
});

test("effectful plugin capabilities remain independently projected", async ({ page }) => {
  await page.goto(sampleAdvertisingExperimentPluginPath);
  await expect(page.getByRole("heading", { name: "ChatGPT Advertising Experiment Plugin" })).toBeVisible();
  const readiness = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "Capability readiness" }) }).first();
  await expect(readiness.getByRole("heading", { name: "Mock lab" })).toBeVisible();
  await expect(readiness.getByRole("heading", { name: "Live read" })).toBeVisible();
  await expect(readiness.getByRole("heading", { name: "Live write" })).toBeVisible();
  await expect(readiness.getByText("usable-now", { exact: false })).toBeVisible();
  await expect(readiness.getByText("setup-required", { exact: false })).toBeVisible();
  await expect(readiness.getByText("not-verified", { exact: false })).toBeVisible();
  await expect(readiness.getByText("No supported OpenAI Ads mutation adapter", { exact: false })).toBeVisible();
});

test("tools route and detail smoke", async ({ page }) => {
  await page.goto("/tools-mcp");
  await expect(page.getByRole("heading", { name: "Tools & MCP Catalog" })).toBeVisible();
  await expect(page.getByText(/\d+ result\(s\)/)).toBeVisible();

  await page.goto(sampleToolPath);
  await expect(page.getByRole("heading", { name: "GA4 MCP Connector" })).toBeVisible();
  await expect(page.getByText("tools-mcp/analytics").first()).toBeVisible();
  await expect(page.getByRole("heading", { name: "Operational Summary" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Capabilities" })).toBeVisible();
  await expect(page.getByText("remote-mcp-server")).toBeVisible();
  const ga4Usability = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "How You Can Use This" }) }).first();
  await expect(ga4Usability.getByText("Setup required", { exact: true }).first()).toBeVisible();
  await expect(ga4Usability.getByText("remote-integration", { exact: false })).toBeVisible();
  await expect(ga4Usability.getByText("GA4_CREDENTIAL runtime binding", { exact: false })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Install (Codex)" })).toBeVisible();
  await expect(page.locator(".copy-button")).toHaveCount(3);
});

test("catalogue filters survive reload and detail navigation, then recover from no results", async ({ page }) => {
  await page.goto("/skills");
  const total = await page.locator(".results-count").textContent();
  await page.getByRole("searchbox", { name: "Search the catalogue" }).fill("weekly");
  await expect(page).toHaveURL(/q=weekly/);
  await page.getByRole("button", { name: "Runtime", exact: true }).click();
  await page.getByRole("checkbox", { name: "codex", exact: true }).check();
  await page.getByRole("checkbox", { name: "claude", exact: true }).check();
  await expect.poll(() => new URL(page.url()).searchParams.getAll("runtime")).toEqual(["codex", "claude"]);
  await page.getByRole("checkbox", { name: "claude", exact: true }).press("Escape");
  await expect(page.getByRole("button", { name: "Runtime", exact: true })).toBeFocused();
  await page.getByRole("button", { name: "Remove runtime codex", exact: true }).click();
  await page.reload();
  await expect(page.getByRole("searchbox")).toHaveValue("weekly");
  await expect(page.getByRole("button", { name: "Remove runtime claude", exact: true })).toBeVisible();
  await page.getByRole("link", { name: /Meta \+ Google Weekly Performance Review/ }).click();
  await expect(page).toHaveURL(new RegExp(sampleSkillPath));
  await expect(page.getByRole("heading", { name: "Meta + Google Weekly Performance Review", exact: true })).toBeVisible();
  await page.goBack();
  await expect(page).toHaveURL(/q=weekly/);
  await expect(page.getByRole("searchbox")).toHaveValue("weekly");
  await expect(page.getByRole("button", { name: "Remove runtime claude", exact: true })).toBeVisible();
  await page.getByRole("searchbox").fill("no-matching-package-987654");
  await expect(page.getByRole("heading", { name: "No matching entries" })).toBeVisible();
  await page.getByRole("button", { name: "Clear search and filters" }).click();
  await expect(page.getByRole("searchbox")).toHaveValue("");
  await expect(page.locator(".results-count")).toHaveText(total!);
  await expect(page).toHaveURL(/\/skills$/);
});

test("deep links retain combined domain, tag and runtime selections", async ({ page }) => {
  await page.goto("/skills?q=weekly&category=marketing-tools&tag=paid-media&runtime=claude");
  await expect(page.getByRole("link", { name: /Meta \+ Google Weekly Performance Review/ })).toBeVisible();
  await expect(page.getByRole("button", { name: "Remove category marketing-tools" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Remove tag paid-media" })).toBeVisible();
  await page.getByRole("button", { name: "Domain", exact: true }).click();
  await expect(page.getByRole("checkbox", { name: "Marketing Tools", exact: true })).toBeChecked();
  await page.getByRole("button", { name: "Clear domain" }).click();
  await expect.poll(() => new URL(page.url()).searchParams.has("category")).toBe(false);
  await expect.poll(() => new URL(page.url()).searchParams.get("tag")).toBe("paid-media");
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await page.getByRole("button", { name: "Reset Filters" }).click();
  await expect(page).toHaveURL(/\/skills$/);
});

test("theme preference survives navigation and reload", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Switch to light mode" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.getByRole("link", { name: "Explore skills", exact: true }).click();
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.getByRole("button", { name: "Switch to dark mode" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("storage and clipboard failures leave a usable page with truthful feedback", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, "localStorage", { get: () => { throw new Error("Storage blocked"); } });
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: async () => { throw new Error("Clipboard blocked"); } } });
  });
  await page.goto(sampleSkillPath);
  await page.getByRole("button", { name: "Switch to light mode" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  const copy = page.getByRole("button", { name: "Copy codex installation command" });
  await copy.click();
  await expect(page.getByText("Clipboard unavailable. Select and copy the command above.")).toBeVisible();
  await expect(copy).toHaveText("Copy");
  await expect(page.locator("pre.install").first()).toContainText("skills-hub install");
});

test("phone navigation works by keyboard and representative pages reflow at 320px", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 780 });
  await page.goto("/");
  const menu = page.locator(".mobile-menu summary");
  await menu.focus();
  await menu.press("Enter");
  await expect(page.locator(".mobile-menu")).toHaveAttribute("open", "");
  await menu.press("Escape");
  await expect(page.locator(".mobile-menu")).not.toHaveAttribute("open", "");
  await expect(menu).toBeFocused();
  for (const route of ["/", "/skills", sampleSkillPath, "/plugins", sampleToolPath]) {
    await page.goto(route);
    await expect(page.locator("main h1")).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  }
});

test("open filter menus stay within the viewport across catalogue breakpoints", async ({ page }) => {
  const menu = page.locator(".filter-menu");
  async function expectMenuToFit(context: string) {
    await expect(menu).toBeVisible();
    await expect.poll(() => menu.evaluate(el => {
      const bounds = el.getBoundingClientRect();
      const viewport = document.documentElement.clientWidth;
      return bounds.left >= 0 && bounds.right <= viewport
        && document.documentElement.scrollWidth <= viewport
        && el.scrollWidth <= el.clientWidth;
    }), { message: `${context} must not create horizontal overflow` }).toBe(true);
  }
  for (const route of ["/skills", "/agents", "/plugins", "/tools-mcp"]) {
    await page.goto(route);
    const widths = route === "/skills" ? [320, 390, 760, 761, 768, 1024, 1150, 1151, 1280] : [320, 768];
    for (const width of widths) {
      await page.setViewportSize({ width, height: 900 });
      for (const filter of ["Domain", "Tag", "Runtime"]) {
        const trigger = page.getByRole("button", { name: filter, exact: true });
        await trigger.click();
        await expectMenuToFit(`${route}: ${filter} at ${width}px`);
        await menu.getByRole("button", { name: "Done", exact: true }).click();
        await expect(trigger).toBeFocused();
      }
    }
  }
  // Reposition an already-open menu when the grid crosses its breakpoints.
  for (const filter of ["Domain", "Tag", "Runtime"]) {
    await page.getByRole("button", { name: filter, exact: true }).click();
    for (const width of [1280, 320, 768]) {
      await page.setViewportSize({ width, height: 900 });
      await expectMenuToFit(`Resizing open ${filter} to ${width}px`);
    }
    await menu.getByRole("button", { name: "Done", exact: true }).click();
  }
});

test("desktop catalogue rail stays at the page edge and stops before the footer", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  for (const route of ["/skills", "/agents", "/plugins", "/tools-mcp"]) {
    await page.goto(route);
    const rail = page.locator(".catalogue-artwork-panel");
    await expect(rail).toBeVisible();
    await expect.poll(() => page.evaluate(() => {
      const rail = document.querySelector(".catalogue-artwork-panel")!.getBoundingClientRect();
      const header = document.querySelector(".site-header")!.getBoundingClientRect();
      const reading = document.querySelector(".catalogue-content")!.getBoundingClientRect();
      return Math.abs(rail.left) < 1 && Math.abs(rail.top - header.bottom) < 1
        && Math.abs(rail.bottom - window.innerHeight) < 1 && Math.abs(reading.left - rail.right) < 1;
    })).toBe(true);
    await page.evaluate(() => window.scrollTo(0, 500));
    await expect.poll(() => page.evaluate(() => {
      const rail = document.querySelector(".catalogue-artwork-panel")!.getBoundingClientRect();
      const header = document.querySelector(".site-header")!.getBoundingClientRect();
      return Math.abs(rail.top - header.bottom) < 1;
    })).toBe(true);
    await page.locator(".site-footer").scrollIntoViewIfNeeded();
    await expect.poll(() => page.evaluate(() => {
      const rail = document.querySelector(".catalogue-artwork-panel")!.getBoundingClientRect();
      const footer = document.querySelector(".site-footer")!.getBoundingClientRect();
      return rail.bottom <= footer.top + 1;
    })).toBe(true);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  }
  // Short filtered lists also contain the rail before the footer.
  await page.goto("/skills?q=no-matching-package-987654");
  await expect(page.getByRole("heading", { name: "No matching entries" })).toBeVisible();
  await page.locator(".site-footer").scrollIntoViewIfNeeded();
  expect(await page.evaluate(() => document.querySelector(".catalogue-artwork-panel")!.getBoundingClientRect().bottom <= document.querySelector(".site-footer")!.getBoundingClientRect().top + 1)).toBe(true);
  for (const size of [{ width: 390, height: 844 }, { width: 1100, height: 1450 }]) {
    await page.setViewportSize(size);
    await expect(page.locator(".catalogue-artwork")).toBeHidden();
    expect(await page.locator(".catalogue-heading").evaluate(el => getComputedStyle(el, "::before").content)).not.toBe("none");
  }
});

test("legacy details report unknown compatibility without inferring support", async ({ page }) => {
  for (const route of [sampleSkillPath, sampleAgentPath, samplePluginPath]) {
    await page.goto(route);
    await expect(page.getByRole("heading", { name: "Declared compatibility", exact: true })).toBeVisible();
    await expect(page.getByText("Compatibility unknown: versions and capabilities have not been declared.", { exact: true })).toBeVisible();
  }
});

test("maintainer compatibility is bound to the exact package and version", async ({ page }) => {
  const { findCatalogueCompatibility } = await import("../lib/catalogueCompatibility");
  expect(findCatalogueCompatibility("tools", "analytics/ga4-mcp-connector", "0.2.0")?.details).toContain("Google Analytics Data API: v1beta");
  expect(findCatalogueCompatibility("tools", "analytics/ga4-mcp-connector", "0.3.0")).toBeUndefined();
  expect(findCatalogueCompatibility("skills", "analytics/ga4-mcp-connector", "0.2.0")).toBeUndefined();
  await page.goto(sampleToolPath);
  await expect(page.getByText("Google Analytics Data API: v1beta", { exact: true })).toBeVisible();
  await expect(page.getByText(/Maintainer-reviewed information for version 0\.2\.0/)).toBeVisible();
  await expect(page.getByText(/This is not a live provider check/)).toBeVisible();
  await expect(page.getByRole("link", { name: "Packaged connector manifest" })).toBeVisible();
});
