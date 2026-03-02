import { expect, test } from "@playwright/test";

const sampleSkillPath = "/skills/marketing/meta-google-weekly-performance-review";
const sampleAgentPath = "/agents/marketing/weekly-performance-supervisor";
const sampleToolPath = "/tools-mcp/analytics/ga4-mcp-connector";

test("home route smoke", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Skills", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Agents", exact: true })).toBeVisible();
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
  await expect(page.getByText("marketing-agents/performance")).toBeVisible();
});

test("tools route and detail smoke", async ({ page }) => {
  await page.goto("/tools-mcp");
  await expect(page.getByRole("heading", { name: "Tools & MCP Catalog" })).toBeVisible();
  await expect(page.getByText(/\d+ result\(s\)/)).toBeVisible();

  await page.goto(sampleToolPath);
  await expect(page.getByRole("heading", { name: "GA4 MCP Connector" })).toBeVisible();
  await expect(page.getByText("tools-mcp/analytics").first()).toBeVisible();
});
