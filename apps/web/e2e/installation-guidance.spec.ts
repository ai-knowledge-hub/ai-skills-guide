import { expect, test } from "@playwright/test";
import { buildModuleInstallSnippet, loadRegistry, type ModuleKey } from "../lib/registry";

const subjects: { module: ModuleKey; id: string; directory: string; environment?: string }[] = [
  { module: "skills", id: "marketing/meta-google-weekly-performance-review", directory: "skills" },
  { module: "agents", id: "marketing/weekly-performance-supervisor", directory: "agents" },
  { module: "plugins", id: "engineering/code-maintenance-plugin", directory: "plugins", environment: "node22" },
  { module: "tools", id: "analytics/ga4-mcp-connector", directory: "tools-mcp", environment: "node22" },
  { module: "tools", id: "adtech/openai-ads-api-client", directory: "tools-mcp", environment: "python3" },
];

test("published commands pin module releases, HTTPS source, adapter and execution requirements", async () => {
  for (const subject of subjects) {
    const registry = await loadRegistry(subject.module);
    const entry = registry.skills.find((item) => item.id === subject.id)!;
    const original = structuredClone(entry);
    for (const runtime of ["codex", "claude", "generic"] as const) {
      const command = buildModuleInstallSnippet(subject.module, entry, runtime);
      const args = command.split(" ");
      const value = (flag: string) => args[args.indexOf(flag) + 1];
      expect(args.slice(0, 2)).toEqual(["skills-hub", "install"]);
      expect(value("--source")).toBe("remote");
      expect(value("--module")).toBe(subject.module);
      expect(value("--registry-url")).toBe(`https://skills.ai-knowledge-hub.org/registry/${subject.module}-index.json`);
      expect(value("--entry")).toBe(`${subject.id}@${entry.latest}`);
      expect(value("--runtime")).toBe(runtime);
      expect(args.includes("--target")).toBe(runtime === "generic");
      if (runtime === "generic") expect(value("--target")).toBe(`./my-agent/${subject.directory}`);
      expect(args.includes("--execution-runtime")).toBe(!!subject.environment);
      if (subject.environment) expect(value("--execution-runtime")).toBe(subject.environment);
      expect(command).not.toContain("./bin/");
      expect(command).not.toContain("@latest");
    }
    expect(entry).toEqual(original);
  }
});

test("installation starts with prerequisites and one keyboard-selected adapter command", async ({ page }) => {
  await page.goto("/skills/marketing/meta-google-weekly-performance-review");
  await page.locator(".resource-install > summary").click();
  const install = page.locator(".install-inline");
  await expect(install.getByText(/commands below assume skills-hub is on your PATH/)).toBeVisible();
  await expect(install.locator(".install-runtime-option[open]")).toHaveCount(0);
  await expect(install.locator("pre.install").first()).not.toBeVisible();
  const codex = install.locator(".install-runtime-option > summary").filter({ hasText: /^Codex$/ });
  await codex.focus();
  await codex.press("Enter");
  await expect(page.getByRole("region", { name: "codex installation command", exact: true })).toBeVisible();
  await expect(install.getByText(/~\/.codex\/skills\/marketing\/meta-google/)).toBeVisible();
  await install.locator(".install-runtime-option > summary").filter({ hasText: /^Claude$/ }).press("Space");
  await expect(install.locator(".install-runtime-option[open]")).toHaveCount(1);
  await expect(page.getByRole("region", { name: "codex installation command", exact: true })).not.toBeVisible();
  await expect(page.getByRole("region", { name: "claude installation command", exact: true })).toContainText("--runtime claude");
  await install.locator(".install-runtime-option > summary").filter({ hasText: /^Generic/ }).click();
  await expect(install.locator(".install-runtime-option[open]")).toHaveCount(1);
  await expect(page.getByRole("region", { name: "generic installation command", exact: true })).toContainText("--target ./my-agent/skills");
  await expect(install.getByText(/After file installation/).last()).toBeVisible();
  await expect(install.getByText("skills-hub help setup", { exact: true }).last()).toBeVisible();
});

test("each module shows matching selected commands and target effects", async ({ page }) => {
  for (const subject of subjects) {
    await page.goto(`/${subject.directory}/${subject.id}`);
    await page.locator(".resource-install > summary").click();
    await page.locator(".install-runtime-option > summary").filter({ hasText: /^Generic/ }).click();
    const selected = page.locator(".install-runtime-option[open]");
    await expect(selected.locator("pre.install")).toContainText(`--module ${subject.module}`);
    await expect(selected.locator("pre.install")).toContainText(`--target ./my-agent/${subject.directory}`);
    await expect(selected.getByText(`./my-agent/${subject.directory}/${subject.id}`, { exact: true })).toBeVisible();
    if (subject.environment) {
      await expect(selected.locator("pre.install")).toContainText(`--execution-runtime ${subject.environment}`);
      await expect(selected.getByText(/declares that environment; it does not install it/)).toBeVisible();
    }
    if (subject.module === "plugins") await expect(selected.getByText(/sibling skill, agent and tool directories/)).toBeVisible();
  }
});

test("native adapter choices and acquisition guidance remain usable without JavaScript", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false, viewport: { width: 390, height: 844 } });
  try {
    const page = await context.newPage();
    await page.goto("http://127.0.0.1:3000/skills/marketing/meta-google-weekly-performance-review");
    await page.locator(".resource-install > summary").click();
    await page.getByText("Download and prepare skills-hub", { exact: true }).click();
    await expect(page.getByRole("link", { name: "SHA256SUMS", exact: true })).toHaveAttribute("href", /\/v0\.4\.0-alpha\.1\/SHA256SUMS$/);
    await expect(page.getByText(/Rename the binary to/)).toBeVisible();
    const codex = page.locator(".install-runtime-option > summary").filter({ hasText: /^Codex$/ });
    await codex.focus();
    await codex.press("Enter");
    await expect(page.getByRole("region", { name: "codex installation command", exact: true })).toContainText("--source remote");
    await expect(page.getByRole("button", { name: "Copy codex installation command" })).toBeDisabled();
    await page.locator(".install-runtime-option > summary").filter({ hasText: /^Generic/ }).click();
    await expect(page.getByRole("region", { name: "generic installation command", exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false);
  } finally {
    await context.close();
  }
});

test("template compositions keep operational installation and download choices absent", async ({ page }) => {
  await page.goto("/plugins/marketing/performance-reporting-plugin");
  await page.locator(".resource-install > summary").click();
  await expect(page.locator(".install-runtime-option")).toHaveCount(0);
  await expect(page.locator("pre.install")).toHaveCount(0);
  await expect(page.getByText("Download and prepare skills-hub", { exact: true })).toHaveCount(0);
});
