import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { gzipSync } from "node:zlib";
import { pack } from "tar-stream";
import { expect, test } from "@playwright/test";
import { buildModuleInstallSnippet, loadRegistry, type ModuleKey } from "../lib/registry";
import { loadInstallationGuidance } from "../lib/installationGuidance";
import { readReleaseFiles } from "../lib/releaseFiles";
import { firstUseNotes, loadResourceFirstUse } from "../lib/resourceFirstUse";

async function writeArchive(filename: string, files: Map<string, string | Buffer>) {
  const archive = pack();
  const chunks: Buffer[] = [];
  const collected = (async () => {
    for await (const chunk of archive) chunks.push(Buffer.from(chunk));
  })();
  for (const [name, content] of files) archive.entry({ name }, content);
  archive.finalize();
  await collected;
  await fs.mkdir(path.dirname(filename), { recursive: true });
  await fs.writeFile(filename, gzipSync(Buffer.concat(chunks)));
}

const unpreviewableExamples = [
  { name: "large valid JSON", content: Buffer.from(JSON.stringify({ padding: "x".repeat(1024 * 1024) })) },
  { name: "empty file", content: Buffer.alloc(0) },
  { name: "invalid UTF-8", content: Buffer.from([0xff]) },
];

test("unavailable bundled requirements keep previews readable without guessing install commands", async () => {
  const registry = await loadRegistry("agents");
  const entry = registry.skills.find((item) => item.id === "marketing/weekly-performance-supervisor")!;
  const root = await fs.mkdtemp(path.join(os.tmpdir(), "closure-preview-"));
  const previousRoot = process.env.RELEASE_STORE_ROOT;
  const filename = "bundled/tools-mcp/analytics/ga4-mcp-connector/tool.yaml";
  try {
    for (const example of [...unpreviewableExamples, { name: "invalid YAML", content: Buffer.from("id: [") }]) {
      await writeArchive(path.join(root, "agents", entry.id, entry.latest, "package.tar.gz"), new Map([[filename, example.content]]));
      process.env.RELEASE_STORE_ROOT = root;
      const guidance = await loadInstallationGuidance("agents", entry);
      expect(guidance.unavailable).toContain(filename);
      for (const runtime of ["codex", "claude", "generic"] as const) {
        expect(buildModuleInstallSnippet("agents", entry, runtime, guidance)).toBe("");
      }
    }
    const largeClosure = new Map<string, string>();
    for (let index = 0; index < 6; index++) {
      largeClosure.set(`bundled/tools-mcp/fixtures/tool-${index}/tool.yaml`, `schema_version: "2.1"\nid: fixtures/tool-${index}\nversion: "0.1.0"\nruntimes: [codex, claude, generic]\nexecution:\n  kind: script\n  supported_platforms: [linux, macos]\n  supported_runtimes: [node22]\n# ${"x".repeat(800 * 1024)}`);
    }
    await writeArchive(path.join(root, "agents", entry.id, entry.latest, "package.tar.gz"), largeClosure);
    const bounded = await loadInstallationGuidance("agents", entry);
    expect(bounded.unavailable).toContain("bundled/tools-mcp/fixtures/tool-5/tool.yaml");
    expect(buildModuleInstallSnippet("agents", entry, "generic", bounded)).toBe("");
  } finally {
    if (previousRoot === undefined) delete process.env.RELEASE_STORE_ROOT;
    else process.env.RELEASE_STORE_ROOT = previousRoot;
    await fs.rm(root, { recursive: true, force: true });
  }
});

for (const subject of [
  { module: "agents", id: "marketing/weekly-performance-supervisor" },
  { module: "plugins", id: "engineering/code-maintenance-plugin" },
] as const) {
  test(`${subject.module} withhold commands for missing or mismatched expected closure members`, async () => {
    const registry = await loadRegistry(subject.module);
    const entry = registry.skills.find((item) => item.id === subject.id)!;
    const lockPath = entry.artifact?.dependency_lock;
    const original = await readReleaseFiles(subject.module, entry, (name) => name === lockPath || /^bundled\/.+\/(skill|agent|tool)\.yaml$/.test(name));
    const missing = subject.module === "agents"
      ? ["bundled/tools-mcp/analytics/ga4-mcp-connector/tool.yaml", "bundled/tools-mcp/warehouse/bigquery-mcp-query-runner/tool.yaml"]
      : [JSON.parse(original!.get(lockPath!)!).components[0].manifest_path as string];
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "missing-closure-"));
    const previousRoot = process.env.RELEASE_STORE_ROOT;
    try {
      // Model partial stores without changing any real retained release.
      for (const mutation of ["absent", "misnamed", "wrong-version", ...(lockPath ? ["missing-lock", "unlocked-extra"] : [])]) {
        const files = new Map(original!);
        if (mutation === "missing-lock") files.delete(lockPath!);
        else if (mutation === "unlocked-extra") {
          files.set("bundled/skills/fixtures/extra/skill.yaml", 'schema_version: "1.1"\nid: fixtures/extra\nversion: "0.1.0"\nruntimes: [codex, claude, generic]\n');
        } else {
          for (const name of missing) {
            const source = files.get(name)!;
            files.delete(name);
            if (mutation === "misnamed") files.set(name.replace(/yaml$/, "yml"), source);
            if (mutation === "wrong-version") files.set(name, source.replace(/^version:.*$/m, 'version: "9.9.9"'));
          }
        }
        await writeArchive(path.join(root, subject.module, entry.id, entry.latest, "package.tar.gz"), files);
        process.env.RELEASE_STORE_ROOT = root;
        const guidance = await loadInstallationGuidance(subject.module, entry);
        expect(guidance.unavailable.length, mutation).toBeGreaterThan(0);
        if (["absent", "misnamed", "wrong-version"].includes(mutation)) {
          for (const name of missing) expect(guidance.unavailable, mutation).toContain(name);
        }
        for (const runtime of ["codex", "claude", "generic"] as const) expect(buildModuleInstallSnippet(subject.module, entry, runtime, guidance)).toBe("");
      }
    } finally {
      if (previousRoot === undefined) delete process.env.RELEASE_STORE_ROOT;
      else process.env.RELEASE_STORE_ROOT = previousRoot;
      await fs.rm(root, { recursive: true, force: true });
    }
  });
}

const readmeSubjects: { module: ModuleKey; id: string; document: string; manifest: string }[] = [
  { module: "skills", id: "marketing/meta-google-weekly-performance-review", document: "SKILL.md", manifest: "skill.yaml" },
  { module: "agents", id: "marketing/weekly-performance-supervisor", document: "AGENT.md", manifest: "agent.yaml" },
  { module: "tools", id: "analytics/ga4-mcp-connector", document: "TOOL.md", manifest: "tool.yaml" },
  { module: "tools", id: "adtech/conversion-event-reconciler", document: "TOOL.md", manifest: "tool.yaml" },
  { module: "plugins", id: "marketing/performance-reporting-plugin", document: "README.md", manifest: "plugin.yaml" },
];

for (const subject of readmeSubjects) {
  test(`${subject.id} keeps summaries available when any release preview is unavailable`, async () => {
    const registry = await loadRegistry(subject.module);
    const entry = registry.skills.find((item) => item.id === subject.id)!;
    const original = structuredClone(entry);
    const directory = subject.module === "tools" ? "tools-mcp" : subject.module;
    const note = firstUseNotes.find((item) => item.module === subject.module && item.id === entry.id && item.version === entry.latest);
    const names = [...new Set([
      subject.document, subject.manifest, "README.md", "examples/input.json", "examples/output.json", "examples/first-use-input.json",
      ...(note?.sources ?? []),
    ])];
    const released = await readReleaseFiles(directory, entry, names);
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "readme-preview-"));
    const previousRoot = process.env.RELEASE_STORE_ROOT;
    try {
      for (const name of names) {
        for (const example of unpreviewableExamples) {
          const files = new Map<string, string | Buffer>(released!);
          // Keep oversized Markdown valid and retain YAML semantics by adding
          // only a comment to the original manifest.
          const body = example.name === "large valid JSON" && !name.endsWith(".json")
            ? Buffer.from(`${released!.get(name) ?? ""}\n# ${"x".repeat(1024 * 1024)}`)
            : example.content;
          files.set(name, body);
          await writeArchive(path.join(root, directory, entry.id, entry.latest, "package.tar.gz"), files);
          process.env.RELEASE_STORE_ROOT = root;

          const content = await loadResourceFirstUse(subject.module, entry);
          expect(content.files?.has(name), `${name}: ${example.name}`).toBe(false);
          expect(content.unavailablePreviews).toEqual([name]);
          expect(content.sources).not.toContain(name);
          for (const source of content.sources) expect(content.files?.has(source)).toBe(true);
          for (const [filename, source] of released!) {
            if (filename !== name) expect(content.files?.get(filename)).toBe(source);
          }
          expect(content.inputs).not.toBe("");
          expect(content.outputs).not.toBe("");
          expect(content.requirements.length).toBeGreaterThan(0);
          expect(entry).toEqual(original);
        }
      }
      await writeArchive(path.join(root, directory, entry.id, entry.latest, "package.tar.gz"), new Map(names.map((name) => [name, Buffer.alloc(0)])));
      const metadataOnly = await loadResourceFirstUse(subject.module, entry);
      expect(metadataOnly.sources).toEqual([]);
      expect(metadataOnly.unavailablePreviews).toEqual(names);
      expect(metadataOnly.sample).toBeUndefined();
      expect(metadataOnly.inputs).toContain("could not be summarized");
      const outputs = entry.operational?.outputs;
      expect(metadataOnly.outputs).toBe(outputs?.length
        ? outputs.map((item) => `- ${item}`).join("\n")
        : "Expected outputs could not be summarized from the available previews. Inspect the release package before use.");
    } finally {
      if (previousRoot === undefined) delete process.env.RELEASE_STORE_ROOT;
      else process.env.RELEASE_STORE_ROOT = previousRoot;
      await fs.rm(root, { recursive: true, force: true });
    }
  });
}

for (const example of unpreviewableExamples) {
  test(`optional examples with ${example.name} do not hide released instructions or metadata`, async () => {
    const registry = await loadRegistry("skills");
    const entry = registry.skills.find((item) => item.id === "marketing/meta-google-weekly-performance-review")!;
    const original = structuredClone(entry);
    const released = await readReleaseFiles("skills", entry, ["SKILL.md", "README.md", "skill.yaml"]);
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "release-preview-"));
    const previousRoot = process.env.RELEASE_STORE_ROOT;
    try {
      const files = new Map<string, string | Buffer>(released!);
      files.set("examples/input.json", example.content);
      files.set("examples/output.json", example.content);
      files.set("examples/first-use-input.json", JSON.stringify({ supplied_export: "metrics.csv" }));
      await writeArchive(path.join(root, "skills", entry.id, entry.latest, "package.tar.gz"), files);
      process.env.RELEASE_STORE_ROOT = root;

      const content = await loadResourceFirstUse("skills", entry);
      expect(content.files?.get("SKILL.md")).toBe(released!.get("SKILL.md"));
      expect(content.files?.get("skill.yaml")).toBe(released!.get("skill.yaml"));
      expect(content.files?.has("examples/input.json")).toBe(false);
      expect(content.exampleOutput).toBeUndefined();
      expect(content.exampleInput).toBe(files.get("examples/first-use-input.json"));
      expect(content.inputs).toContain("Meta and Google Ads account IDs");
      expect(content.outputs).toContain("Executive summary and KPI table");
      expect(content.sources).not.toContain("examples/output.json");
      expect(content.unavailablePreviews).toEqual(["examples/input.json", "examples/output.json"]);
      expect(entry).toEqual(original);

      // Consumers that require a document (rather than a best-effort summary)
      // retain the reader's strict default.
      files.set("SKILL.md", example.content);
      await writeArchive(path.join(root, "skills", entry.id, entry.latest, "package.tar.gz"), files);
      await expect(readReleaseFiles("skills", entry, ["SKILL.md"])).rejects.toThrow();
    } finally {
      if (previousRoot === undefined) delete process.env.RELEASE_STORE_ROOT;
      else process.env.RELEASE_STORE_ROOT = previousRoot;
      await fs.rm(root, { recursive: true, force: true });
    }
  });
}
