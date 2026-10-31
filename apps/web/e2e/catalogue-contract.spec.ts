import fs from "node:fs/promises";
import path from "node:path";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { expect, test } from "@playwright/test";
import UsabilityPanel from "../components/UsabilityPanel";
import { catalogueCompatibility, findCatalogueCompatibility, type CatalogueCompatibility } from "../lib/catalogueCompatibility";
import { loadRegistry, withCatalogueCompatibility, type ModuleKey, type RegistryEntry, type RegistryIndex } from "../lib/registry";
import { latestToolPageNotes } from "./catalogue-page-notes";

// Independent review inventory: docs/tool-version-review.md and its retained
// releases. Additional entries/releases need not acquire a note automatically.
const reviewedTools = [
  ["analytics/ga4-mcp-connector", "0.2.0"],
  ["warehouse/bigquery-mcp-query-runner", "0.2.0"],
  ["ads/meta-ads-mcp-connector", "0.2.0"],
  ["adtech/openai-ads-api-client", "0.2.0"],
  ["adtech/ad-platform-executor-template", "0.2.0"],
  ["agentops/agent-control-plane-server", "0.2.1"],
  ["adtech/conversion-event-reconciler", "0.1.0"]
] as const;
const modules: ModuleKey[] = ["skills", "agents", "tools", "plugins"];
const subjectKey = (module: ModuleKey, id: string, version: string) => JSON.stringify([module, id, version]);

test("reviewed content has unique retained subjects and usable provenance", async () => {
  const seen = new Set<string>();
  for (const note of catalogueCompatibility) {
    const subject = subjectKey(note.module, note.id, note.version);
    expect(seen.has(subject), `duplicate annotation: ${subject}`).toBe(false);
    seen.add(subject);
    const moduleDir = note.module === "tools" ? "tools-mcp" : note.module;
    // Release records are independent of the current registry projection and
    // its mutable source manifests. Historical notes remain valid records.
    const release = path.resolve("../../releases", moduleDir, note.id, note.version);
    const retained = JSON.parse(await fs.readFile(path.join(release, "registry-entry.json"), "utf8"));
    expect(retained.id).toBe(note.id);
    await fs.access(path.join(release, "package.tar.gz"));
    expect(note.details.length).toBeGreaterThan(0);
    expect(note.details.every(detail => detail.trim().length > 0)).toBe(true);
    expect(note.source.label.trim()).not.toBe("");
    expect(new URL(note.source.url).protocol).toBe("https:");
    expect(note.reviewed).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(new Date(note.reviewed).toISOString().slice(0, 10)).toBe(note.reviewed);
  }
  for (const [id, version] of reviewedTools) {
    expect(seen.has(subjectKey("tools", id, version)), `missing reviewed release: ${id}@${version}`).toBe(true);
  }
});

test("annotation selection uses exact module, identity and release without range inference", () => {
  // Independent tuple-key model, rather than the production find predicate.
  // New reviewed releases expand the matrix without changing a fixed total.
  const subjects = new Set(catalogueCompatibility.map(note => subjectKey(note.module, note.id, note.version)));
  const ids = [...new Set([...catalogueCompatibility.map(note => note.id), "warehouse/unknown-connector", "warehouse/bigquery-mcp-query-runner-extra"])];
  const versions = [...new Set([...catalogueCompatibility.map(note => note.version), "0.3.0", "v0.2.0", "latest", "unreviewed-version"])];
  for (const module of modules) {
    for (const id of ids) {
      for (const version of versions) {
        const result = findCatalogueCompatibility(module, id, version);
        const subject = subjectKey(module, id, version);
        if (subjects.has(subject)) {
          expect(result && subjectKey(result.module, result.id, result.version), subject).toBe(subject);
        } else {
          expect(result, subject).toBeUndefined();
        }
      }
    }
  }
});

test("registry decoration preserves every package declaration and selects only its latest release note", async () => {
  const notes = new Map(catalogueCompatibility.map(note => [subjectKey(note.module, note.id, note.version), note]));
  for (const module of modules) {
    const registryPath = path.resolve(`../../registry/${module}-index.json`);
    const original = JSON.parse(await fs.readFile(registryPath, "utf8")) as RegistryIndex;
    const decorated = await loadRegistry(module);
    const declarations = decorated.skills.map(({ catalogueCompatibility: note, ...entry }) => {
      expect(note, `${module}/${entry.id}@${entry.latest}`).toEqual(notes.get(subjectKey(module, entry.id, entry.latest)));
      return entry;
    });
    expect({ ...decorated, skills: declarations }).toEqual(original);
  }
});

test("tool page assertions select only the latest review while retaining historical notes", () => {
  const oldNote = findCatalogueCompatibility("tools", "analytics/ga4-mcp-connector", "0.2.0")!;
  const newNote = { ...oldNote, version: "0.3.0" };
  const otherModuleNote = { ...newNote, module: "skills" as const };
  for (const scenario of [
    { latest: "0.2.0", notes: [oldNote], expected: [oldNote] },
    { latest: "0.3.0", notes: [oldNote], expected: [] },
    { latest: "0.3.0", notes: [oldNote, newNote], expected: [newNote] },
    { latest: "0.3.0", notes: [oldNote, otherModuleNote], expected: [] }
  ]) {
    const retained = structuredClone(scenario.notes);
    expect(latestToolPageNotes([{ id: oldNote.id, latest: scenario.latest }], scenario.notes)).toEqual(scenario.expected);
    expect(scenario.notes).toEqual(retained);
    expect(scenario.notes).toContain(oldNote);
  }
});

test("a newer unreviewed release shows unknown compatibility while its old note stays retained", async ({ page }) => {
  const id = "analytics/ga4-mcp-connector";
  const historical = findCatalogueCompatibility("tools", id, "0.2.0")!;
  const retained = JSON.parse(await fs.readFile(path.resolve("../../releases/tools-mcp", id, "0.2.0/registry-entry.json"), "utf8")) as RegistryEntry;
  // Simulate 0.3.0 today. If that release later acquires a note, pick the next
  // unreviewed minor without forcing real releases or annotations to change.
  const reviewedVersions = new Set(catalogueCompatibility.filter(note => note.module === "tools" && note.id === id).map(note => note.version));
  let minor = 3;
  while (reviewedVersions.has(`0.${minor}.0`)) minor++;
  const latest = `0.${minor}.0`;
  const fixture: RegistryIndex = {
    registry_version: "1.3", generated_at: "2026-10-31T00:00:00Z",
    skills: [{
      ...retained, latest, catalogueCompatibility: historical,
      versions: ["0.2.0", latest].map(version => ({
        version, released_at: "2026-10-31T00:00:00Z",
        manifest_url: `https://example.com/fixtures/ga4/${version}.yaml`,
        artifact_url: `https://example.com/fixtures/ga4/${version}.tar.gz`, sha256: "0".repeat(64)
      }))
    }]
  };
  const entry = withCatalogueCompatibility(fixture, "tools").skills[0];
  expect(entry.catalogueCompatibility).toBeUndefined();
  expect(latestToolPageNotes([entry], catalogueCompatibility)).toEqual([]);
  expect(fixture.skills[0].catalogueCompatibility).toBe(historical);
  expect(findCatalogueCompatibility("tools", id, "0.2.0")).toBe(historical);

  await page.setContent(renderToStaticMarkup(React.createElement(UsabilityPanel, {
    usability: entry.usability, compatibility: entry.compatibility, catalogueCompatibility: entry.catalogueCompatibility
  })));
  const panel = page.locator("#usability");
  await expect(panel.getByText("Compatibility unknown: versions and capabilities have not been declared.", { exact: true })).toBeVisible();
  await expect(panel.getByText(/Maintainer-reviewed information/)).toHaveCount(0);
  await expect(panel.getByRole("link", { name: historical.source.label, exact: true })).toHaveCount(0);
});

test("compatibility display distinguishes unknown, declared and reviewed information", async ({ page }) => {
  const usability = { availability: "not-verified", execution: "local-tool", source: "declared" } as const;
  const compatibility = [{ kind: "api", name: "Fixture Warehouse API", versions: ["v2"] }] as const;
  const note: CatalogueCompatibility = {
    module: "tools", id: "fixtures/local-tool", version: "1.0.0", reviewed: "2000-01-01",
    details: ["Reviewed fixture API target: v1"], limitations: ["No live provider check was performed."],
    source: { label: "Fixture review source", url: "https://example.com/review" }
  };
  for (const state of ["unknown", "declared", "reviewed"] as const) {
    await page.setContent(renderToStaticMarkup(React.createElement(UsabilityPanel, {
      usability,
      compatibility: state === "unknown" ? undefined : compatibility.map(item => ({ ...item, versions: [...item.versions] })),
      catalogueCompatibility: state === "reviewed" ? note : undefined
    })));
    const panel = page.locator("#usability");
    await expect(panel.getByText("Not verified", { exact: true })).toBeVisible();
    if (state === "unknown") {
      await expect(panel.getByText("Compatibility unknown: versions and capabilities have not been declared.", { exact: true })).toBeVisible();
    } else if (state === "declared") {
      await expect(panel.getByText("Fixture Warehouse API", { exact: true })).toBeVisible();
      await expect(panel.getByText(/Maintainer support declarations/)).toBeVisible();
      await expect(panel.getByText(/Maintainer-reviewed information/)).toHaveCount(0);
    } else {
      // Old review dates remain historical context, with no invented expiry.
      await expect(panel.getByText(/Reviewed 2000-01-01\. This is not a live provider check\./)).toBeVisible();
      await expect(panel.getByText("Reviewed fixture API target: v1", { exact: true })).toBeVisible();
      await expect(panel.getByText("No live provider check was performed.", { exact: true })).toBeVisible();
      await expect(panel.getByRole("link", { name: "Fixture review source" })).toHaveAttribute("href", note.source.url);
      await expect(panel.getByText("Fixture Warehouse API", { exact: true })).toHaveCount(0);
    }
  }
});
