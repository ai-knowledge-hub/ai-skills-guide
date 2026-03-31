import { expect, test } from "@playwright/test";

const sampleSkillPath = "/skills/marketing/meta-google-weekly-performance-review";
const sampleAgentPath = "/agents/marketing/weekly-performance-supervisor";
const samplePluginPath = "/plugins/marketing/performance-reporting-plugin";
const sampleEngineeringPluginPath = "/plugins/engineering/code-maintenance-plugin";
const sampleSecurityPluginPath = "/plugins/security/runtime-safety-plugin";
const sampleHarnessPluginPath = "/plugins/agentops/harness-governance-plugin";
const sampleToolPath = "/tools-mcp/analytics/ga4-mcp-connector";

test("home route smoke", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Skills", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Agents", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Plugins", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Tools & MCP" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Explore skills" })).toBeVisible();
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
      value: { writeText: async () => {} }
    });
  });

  await page.goto(sampleSkillPath);
  await expect(page.getByRole("heading", { name: "Operational Summary" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Outputs" })).toBeVisible();
  await expect(page.getByText("may-run-local-verification")).toBeVisible();
  const copyButtons = page.locator(".copy-button");

  await expect(copyButtons).toHaveCount(3);
  await expect(copyButtons.first()).toHaveText("Copy");
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
  await expect(page.getByRole("heading", { name: "Install Summary" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Installed Skills" })).toBeVisible();
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
  await expect(page.getByRole("heading", { name: "Requirements" })).toBeVisible();
  await expect(page.getByText("human-review-for-live-remediation")).toBeVisible();
  await expect(page.getByRole("link", { name: "quarantine-suspicious-instructions" })).toBeVisible();

  await page.goto(sampleHarnessPluginPath);
  await expect(page.getByRole("heading", { name: "Harness Governance Plugin" })).toBeVisible();
  await expect(page.getByText("AgentOps Plugins / Governance")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Installed Skills" })).toBeVisible();
  await expect(page.getByRole("link", { name: "agentops/harness-run-reflection" })).toBeVisible();
  await expect(page.getByText("human-approval-for-harness-update")).toBeVisible();
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
});
