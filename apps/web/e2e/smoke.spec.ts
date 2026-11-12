import { expect, test } from "@playwright/test";
import { catalogueCompatibility } from "../lib/catalogueCompatibility";
import { loadRegistry } from "../lib/registry";
import { latestToolPageNotes } from "./catalogue-page-notes";
import { readReleaseFiles } from "../lib/releaseFiles";

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
  await expect(page.getByRole("region", { name: "Know what you are installing" }).getByText("Not verified", { exact: true })).toBeVisible();
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
  const copyButtons = page.locator(".install-inline .copy-button");

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
  const preview = page.locator(".detail-disclosure");
  const workflowToggle = preview.locator(":scope > summary");
  await expect(preview).not.toHaveAttribute("open", "");
  await workflowToggle.focus();
  await workflowToggle.press("Enter");
  await expect(preview.getByRole("heading", { name: "What it coordinates" })).toBeVisible();
  await expect(preview.getByText("QA gating and blocking decisions")).toBeVisible();
  await expect(preview.getByRole("heading", { name: "Expected outputs" })).toBeVisible();
  await expect(preview.getByText("Semi-autonomous", { exact: true })).toBeVisible();
  await expect(preview.getByText(/require human approval before any external publish/)).toBeVisible();
  await preview.getByText("Read agent instructions (AGENT.md)", { exact: true }).click();
  await expect(preview.getByText("Never bypass critical QA failures.")).toBeVisible();
  const usability = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "How You Can Use This" }) }).first();
  await expect(usability.getByText("Setup required", { exact: true }).first()).toBeVisible();
  await expect(usability.getByText("Resource form: Agent workflow", { exact: true })).toBeVisible();
  await expect(usability.getByText("GA4 and BigQuery credentials", { exact: false })).toBeVisible();
  await expect(page.getByText("Install agent files", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Install (Codex)" })).toBeVisible();
  await expect(page.locator(".install-inline .copy-button")).toHaveCount(3);
  await expect(page.locator("pre.install").first()).toContainText("--module agents");

  await page.goto("/agents/marketing/campaign-qa-supervisor");
  await page.getByText("View agent workflow", { exact: true }).click();
  await expect(preview.getByText(/campaign configuration reader is not yet available/)).toBeVisible();
});

test("instruction copy preserves released source and recovers from clipboard failure", async ({ page }) => {
  await page.addInitScript(() => {
    const state = window as Window & { __copiedInstructions?: string; __clipboardBlocked?: boolean };
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: async (value: string) => {
          if (state.__clipboardBlocked) throw new Error("Clipboard blocked");
          state.__copiedInstructions = value;
        }
      }
    });
  });

  await page.setViewportSize({ width: 390, height: 844 });
  for (const { module, route, filename } of [
    { module: "skills", route: sampleSkillPath, filename: "SKILL.md" },
    { module: "agents", route: sampleAgentPath, filename: "AGENT.md" },
  ] as const) {
    const registry = await loadRegistry(module);
    const entry = registry.skills.find((entry) => entry.id === route.split("/").slice(2).join("/"))!;
    const files = await readReleaseFiles(module, entry, [filename]);
    const source = files?.get(filename);
    expect(source).toBeTruthy();

    await page.goto(route);
    const copy = page.getByRole("button", { name: `Copy full ${filename} text` });
    await expect(copy).not.toBeVisible();
    await page.locator(".detail-disclosure > summary").click();
    if (module === "agents") {
      await expect(copy).not.toBeVisible();
      await page.getByText("Read agent instructions (AGENT.md)", { exact: true }).click();
    }
    await copy.focus();
    await copy.press("Enter");
    await expect(copy).toHaveText("Copied");
    await expect.poll(() => page.evaluate(() => (window as Window & { __copiedInstructions?: string }).__copiedInstructions)).toBe(source);

    await page.evaluate(() => { (window as Window & { __clipboardBlocked?: boolean }).__clipboardBlocked = true; });
    await copy.click();
    const fallback = page.getByRole("textbox", { name: `${filename} source text` });
    await expect(fallback).toHaveValue(source!);
    await expect(page.getByText("Clipboard unavailable. Select and copy the source below.", { exact: true })).toBeVisible();
    await expect(copy).toHaveText("Copy full text");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);

    await page.evaluate(() => { (window as Window & { __clipboardBlocked?: boolean }).__clipboardBlocked = false; });
    await copy.click();
    await expect(copy).toHaveText("Copied");
    await expect(fallback).toHaveCount(0);
  }
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
  const preview = page.locator(".detail-disclosure");
  await expect(preview).not.toHaveAttribute("open", "");
  await preview.locator("summary").first().click();
  await expect(page.getByRole("heading", { name: "Referenced Skills" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Referenced Tools" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Reference template" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Install (Codex)" })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Packaged Hooks" })).toBeVisible();
  await expect(page.getByRole("link", { name: "marketing/meta-google-weekly-performance-review" })).toBeVisible();
});

test("plugin listing and detail distinguish templates from file-installable bundles", async ({ page }) => {
  for (const example of [
    { name: "Performance Reporting Plugin", path: samplePluginPath, template: true },
    { name: "Content Repurposing Plugin", path: "/plugins/marketing/content-repurposing-plugin", template: false },
    { name: "Code Maintenance Plugin", path: sampleEngineeringPluginPath, template: false },
  ]) {
    await page.goto("/plugins");
    const introduction = page.locator(".catalogue-heading > p");
    await expect(introduction).toContainText("installable bundles and reference templates");
    await expect(page.locator('meta[name="description"]')).toHaveAttribute("content", (await introduction.textContent())!);
    await page.getByRole("searchbox", { name: "Search the catalogue" }).fill(example.name);
    const card = page.locator(".catalog-card").filter({ has: page.getByRole("heading", { name: example.name, exact: true }) });
    await expect(card.locator(".usability-badge")).toHaveText(example.template ? "Template only" : "Not verified");
    await card.getByRole("link").first().click();
    await expect(page).toHaveURL(example.path);
    if (example.template) {
      await expect(page.getByRole("heading", { name: "Reference composition" })).toBeVisible();
      await expect(page.locator(".install-inline")).toContainText("Implementation is required before the described workflow can run.");
      await expect(page.getByRole("button", { name: /installation command/ })).toHaveCount(0);
      await page.getByText("View reference contents", { exact: true }).click();
      await expect(page.getByRole("heading", { name: "Referenced skills" })).toBeVisible();
    } else {
      await expect(page.getByRole("heading", { name: "Install plugin files" })).toBeVisible();
      await expect(page.getByRole("button", { name: /installation command/ })).toHaveCount(3);
      await expect(page.locator(".install-explanation")).toContainText("Confirm discovery in your agent app");
      await expect(page.locator("#usability")).toContainText("Current operational evidence for your environment is not shown here.");
      await page.getByText("View package contents", { exact: true }).click();
      await expect(page.getByRole("heading", { name: "Bundled skills" })).toBeVisible();
    }
  }
});

test("instruction use and program evidence remain separate", async ({ page }) => {
  await page.goto(sampleSkillPath);
  await expect(page.locator(".detail-use-summary")).toContainText("Agent instructions");
  await expect(page.getByRole("button", { name: "Copy full SKILL.md text" })).not.toBeVisible();
  await page.getByText("Read full instructions", { exact: true }).click();
  await expect(page.getByRole("button", { name: "Copy full SKILL.md text" })).toBeVisible();
  await expect(page.locator(".install-explanation")).toContainText("commands copy guidance");
  const usability = page.locator("#usability");
  const helper = usability.locator("section").filter({ has: page.locator("code", { hasText: "scripts/compute_metrics.py" }) });
  await expect(helper.locator(".usability-badge")).toHaveText("Not verified");
  const explanation = usability.locator("details");
  await explanation.locator("summary").focus();
  await explanation.locator("summary").press("Enter");
  for (const term of ["File installation", "Native discovery", "Mock execution", "Connected operation"]) {
    await expect(explanation.getByText(term, { exact: true })).toBeVisible();
  }
  await page.goto("/tools-mcp/adtech/openai-ads-api-client");
  await expect(page.locator("#usability")).toContainText("Automatic pagination, retries and rate-limit backoff are not included.");
  await expect(page.getByRole("button", { name: "Copy codex installation command" })).toBeVisible();
});

test("new plugin domains smoke", async ({ page }) => {
  await page.goto(sampleEngineeringPluginPath);
  await expect(page.getByRole("heading", { name: "Code Maintenance Plugin" })).toBeVisible();
  await expect(page.getByText("Engineering Plugins / Maintenance")).toBeVisible();
  await page.getByText("View package contents", { exact: true }).click();
  await expect(page.getByRole("heading", { name: "Bundled skills" })).toBeVisible();
  await expect(page.getByRole("listitem").filter({ has: page.getByRole("link", { name: "engineering/code-change-verification" }) })).toContainText("v0.1.0");
  await page.locator("summary").filter({ hasText: "verification-before-complete" }).click();
  await expect(page.getByText("Before marking work complete:", { exact: true })).toBeVisible();

  await page.goto(sampleSecurityPluginPath);
  await expect(page.getByRole("heading", { name: "Runtime Safety Plugin" })).toBeVisible();
  await expect(page.getByText("Security Plugins / Runtime Safety")).toBeVisible();
  const securityRequirements = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "Requirements" }) }).first();
  await expect(securityRequirements.getByRole("heading", { name: "Requirements" })).toBeVisible();
  await expect(securityRequirements.getByText("human-review-for-live-remediation")).toBeVisible();
  await page.getByText("View package contents", { exact: true }).click();
  await expect(page.locator("summary").filter({ hasText: "quarantine-suspicious-instructions" })).toBeVisible();

  await page.goto(sampleHarnessPluginPath);
  await expect(page.getByRole("heading", { name: "Harness Governance Plugin" })).toBeVisible();
  await expect(page.getByText("AgentOps Plugins / Governance")).toBeVisible();
  await page.getByText("View package contents", { exact: true }).click();
  await expect(page.getByRole("heading", { name: "Bundled skills" })).toBeVisible();
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
  for (const name of ["Mock lab", "Live read"]) {
    const capability = readiness.locator("section").filter({ has: page.getByRole("heading", { name, exact: true }) });
    await expect(capability.locator(".usability-badge")).toHaveText("Not verified");
  }
  const writes = readiness.locator("section").filter({ has: page.getByRole("heading", { name: "Live write", exact: true }) });
  await expect(writes.getByText("Unavailable", { exact: true })).toBeVisible();
  await expect(writes.getByText("Live writes unavailable: no supported OpenAI Ads mutation adapter is included in this release.", { exact: true })).toBeVisible();
  await expect(writes.getByText("Recorded classification · not-verified", { exact: false })).toBeVisible();
  await expect(writes.getByRole("link", { name: "Release declaration" })).toHaveAttribute("href", /\/0a81fa98c8c47cebab16f93bdaf1283b21ed88c4\/releases\/plugins\/marketing\/chatgpt-advertising-experiment-plugin\/0\.2\.0\/plugin\.yaml$/);
  await expect(readiness.locator(".is-usable-now, .is-setup-required")).toHaveCount(0);
  await expect(page.locator("#usability .usability-badge").first()).toHaveText("Not verified");
  await expect(writes.getByRole("listitem").filter({ hasText: "No supported OpenAI Ads mutation adapter" })).toBeVisible();
});

test("tools route and detail smoke", async ({ page }) => {
  await page.goto("/tools-mcp");
  await expect(page.getByRole("heading", { name: "Tools & MCP Catalog" })).toBeVisible();
  await expect(page.getByText(/\d+ result\(s\)/)).toBeVisible();

  await page.goto(sampleToolPath);
  await expect(page.getByRole("heading", { name: "GA4 MCP Connector" })).toBeVisible();
  await expect(page.getByText("tools-mcp/analytics").first()).toBeVisible();
  await expect(page.getByText("Read only · Google Analytics 4", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "What it can do" })).toHaveCount(0);
  await page.getByText("View capabilities and access", { exact: true }).click();
  await expect(page.getByRole("heading", { name: "What it can do" })).toBeVisible();
  await expect(page.getByText("Run bounded read-only reports for explicitly allowed GA4 properties.", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Access limits" })).toBeVisible();
  await expect(page.getByText("remote-mcp-server")).toBeVisible();
  const ga4Usability = page.locator(".detail-panel").filter({ has: page.getByRole("heading", { name: "How You Can Use This" }) }).first();
  await expect(ga4Usability.getByText("Setup required", { exact: true }).first()).toBeVisible();
  await expect(ga4Usability.getByText("Resource form: API or MCP integration", { exact: true })).toBeVisible();
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

for (const saved of ["dark", "light"] as const) {
  test(`theme choice survives navigation when saved ${saved} storage is readable but writes fail`, async ({ page }) => {
    const selected = saved === "dark" ? "light" : "dark";
    await page.addInitScript(saved => localStorage.setItem("skills-hub-theme", saved), saved);
    await page.goto("/");
    await expect(page.getByRole("button", { name: `Switch to ${selected} mode` })).toBeVisible();
    await page.evaluate(() => {
      Storage.prototype.setItem = () => { throw new DOMException("Storage full", "QuotaExceededError"); };
    });
    await page.getByRole("button", { name: `Switch to ${selected} mode` }).click();
    await expect(page.locator("html")).toHaveAttribute("data-theme", selected);
    const documentOrigin = await page.evaluate(() => performance.timeOrigin);
    await page.getByRole("navigation", { name: "Main navigation", exact: true }).getByRole("link", { name: "Skills", exact: true }).click();
    await expect(page).toHaveURL(/\/skills$/);
    expect(await page.evaluate(() => performance.timeOrigin)).toBe(documentOrigin);
    await expect(page.locator("html")).toHaveAttribute("data-theme", selected);
    await page.locator(".catalog-card").first().getByRole("link").first().click();
    await expect(page.locator("pre.install").first()).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("data-theme", selected);
    await page.goBack();
    await expect(page).toHaveURL(/\/skills$/);
    await expect(page.getByRole("button", { name: `Switch to ${saved} mode` })).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("data-theme", selected);
    expect(await page.evaluate(() => localStorage.getItem("skills-hub-theme"))).toBe(saved);
    // A new document can only restore the persisted value; the failed write
    // promises session continuity, not persistence across reloads.
    await page.reload();
    await expect(page.getByRole("button", { name: `Switch to ${selected} mode` })).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("data-theme", saved);
  });
}

test("theme changes from another tab survive subsequent navigation", async ({ page, context }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Switch to light mode" }).click();
  const otherTab = await context.newPage();
  await otherTab.goto("/");
  await otherTab.getByRole("button", { name: "Switch to dark mode" }).click();
  await expect(page.getByRole("button", { name: "Switch to light mode" })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.getByRole("navigation", { name: "Main navigation", exact: true }).getByRole("link", { name: "Skills", exact: true }).click();
  await expect(page).toHaveURL(/\/skills$/);
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await otherTab.close();
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
  for (const route of ["/", "/skills", "/agents", "/plugins", "/tools-mcp"]) {
    await page.goto(route);
    const rail = page.locator(".catalogue-artwork-panel");
    await expect(rail).toBeVisible();
    await expect.poll(() => page.evaluate(() => {
      const rail = document.querySelector(".catalogue-artwork-panel")!.getBoundingClientRect();
      const header = document.querySelector(".site-header")!.getBoundingClientRect();
      const reading = document.querySelector(".catalogue-content, .companion-content")!.getBoundingClientRect();
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
  for (const route of [sampleSkillPath, sampleAgentPath, samplePluginPath, "/tools-mcp/adtech/openai-ads-adapter-template"]) {
    await page.goto(route);
    await expect(page.getByRole("heading", { name: "Declared compatibility", exact: true })).toBeVisible();
    await expect(page.getByText("Compatibility unknown: versions and capabilities have not been declared.", { exact: true })).toBeVisible();
  }
});

test("reviewed tool pages disclose notes, limits and provenance without promoting readiness", async ({ page }) => {
  const tools = await loadRegistry("tools");
  for (const note of latestToolPageNotes(tools.skills, catalogueCompatibility)) {
    await page.goto(`/tools-mcp/${note.id}`);
    const panel = page.locator("#usability");
    await expect(panel.getByText(`Maintainer-reviewed information for version ${note.version} · Reviewed ${note.reviewed}. This is not a live provider check.`, { exact: true })).toBeVisible();
    for (const detail of [...note.details, ...note.limitations]) {
      await expect(panel.getByText(detail, { exact: true })).toBeVisible();
    }
    await expect(panel.getByRole("link", { name: note.source.label, exact: true })).toHaveAttribute("href", note.source.url);
    await expect(panel.getByText("Compatibility unknown:", { exact: false })).toHaveCount(0);
  }
  await page.goto("/tools-mcp/adtech/conversion-event-reconciler");
  await expect(page.locator("#usability").getByText("Not verified", { exact: true })).toBeVisible();
});


test("plugin listing and detail render current readiness rather than retained declarations", async ({ page }) => {
  await page.goto("/plugins");
  const plugin = page.locator(".catalog-card").filter({ has: page.getByRole("heading", { name: "Code Maintenance Plugin", exact: true }) });
  await expect(plugin.locator(".usability-badge")).toHaveText("Not verified");
  await plugin.getByRole("link", { name: /Code Maintenance Plugin/ }).click();
  await expect(page.locator("#usability .usability-badge").first()).toHaveText("Not verified");
});

for (const theme of ["light", "dark"] as const) {
  test(`saved ${theme} theme survives missing-entry pages and recovery`, async ({ page }) => {
    await page.addInitScript(theme => localStorage.setItem("skills-hub-theme", theme), theme);
    for (const route of ["/skills/qa81-missing", "/agents/qa81-missing", "/plugins/qa81-missing", "/tools-mcp/qa81-missing", "/qa81-missing"]) {
      const response = await page.goto(route);
      expect(response?.status(), route).toBe(404);
      await expect(page.getByRole("heading", { name: "Item not found" })).toBeVisible();
      // The control must reflect the restored preference after hydration;
      // observing the bootstrap script's transient attribute is insufficient.
      await expect(page.getByRole("button", { name: `Switch to ${theme === "light" ? "dark" : "light"} mode` })).toBeVisible();
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
      await page.getByRole("link", { name: "Home", exact: true }).click();
      await expect(page).toHaveURL(/\/$/);
      await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
    }
  });
}

test("catalogue downloads only the artwork used by its active layout", async ({ page }) => {
  for (const width of [1440, 390]) {
    await page.goto("about:blank");
    await page.setViewportSize({ width, height: 1000 });
    for (const route of ["/", "/skills", "/agents", "/plugins", "/tools-mcp"]) {
      await page.goto(route, { waitUntil: "networkidle" });
      const artwork = await page.evaluate(() => performance.getEntriesByType("resource")
        .map(resource => new URL(resource.name).pathname)
        .filter(path => /^\/brand\/(community|entrance|folded-field)-.*\.webp$/.test(path)));
      expect(artwork, `${route} at ${width}px`).toEqual([width === 1440 ? "/brand/community-1600.webp" : "/brand/community-1200.webp"]);
    }
  }
});

test("publication paths lead to related resources and back to the reviewed article without changing readiness", async ({ page }) => {
  // Reviewed against the three published guides; this inventory does not use
  // the production mapping to calculate its own expected destinations.
  const pairs = [
    ["/skills/marketing/meta-google-weekly-performance-review", "agent-architect-playbook-building-ai-skills-marketing-adtech"],
    ["/skills/marketing/ai-output-eval-scorecard", "agent-architect-playbook-building-ai-skills-marketing-adtech"],
    ["/tools-mcp/analytics/ga4-mcp-connector", "muscles-of-the-machine-tools-mcp-cli-first-principles-marketing"],
    ["/tools-mcp/warehouse/bigquery-mcp-query-runner", "muscles-of-the-machine-tools-mcp-cli-first-principles-marketing"],
    ["/skills/security/ad-platform-agent-auth-review", "building-secure-marketing-agents-ad-platform-authentication-architecture"],
    ["/skills/agentops/ad-platform-policy-gate-designer", "building-secure-marketing-agents-ad-platform-authentication-architecture"],
    ["/tools-mcp/adtech/ad-platform-executor-template", "building-secure-marketing-agents-ad-platform-authentication-architecture"]
  ];
  for (const [resource, article] of pairs) {
    await page.goto("/");
    const paths = page.getByRole("region", { name: "From reading to practice" });
    const articleURL = `https://ai-knowledge-hub.org/library/${article}/`;
    await expect(paths.locator(`a[href="${articleURL}"]`)).toBeVisible();
    await paths.locator(`a[href="${resource}"]`).click();
    await expect(page).toHaveURL(new RegExp(`${resource}$`));
    await page.getByRole("link", { name: "Related reading", exact: true }).click();
    await expect(page.locator("#related-reading")).toBeVisible();
    await expect(page.locator(`#related-reading a[href="${articleURL}"]`)).toBeVisible();
    await expect(page.locator("#usability .usability-badge").first()).toHaveText(resource.startsWith("/skills/") ? "Documentation" : "Setup required");
  }
  await page.goto("/skills/engineering/implementation-strategy");
  await expect(page.getByRole("heading", { name: "Related reading", exact: true })).toHaveCount(0);
});
