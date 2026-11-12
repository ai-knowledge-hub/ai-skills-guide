import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { gzipSync } from "node:zlib";
import { pack } from "tar-stream";
import { expect, test } from "@playwright/test";
import { loadRegistry, type ModuleKey } from "../lib/registry";
import { readReleaseFiles } from "../lib/releaseFiles";
import { loadResourceFirstUse } from "../lib/resourceFirstUse";

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

const readmeSubjects: { module: ModuleKey; id: string; document: string; manifest: string }[] = [
  { module: "skills", id: "marketing/meta-google-weekly-performance-review", document: "SKILL.md", manifest: "skill.yaml" },
  { module: "agents", id: "marketing/weekly-performance-supervisor", document: "AGENT.md", manifest: "agent.yaml" },
  { module: "tools", id: "analytics/ga4-mcp-connector", document: "TOOL.md", manifest: "tool.yaml" },
  { module: "plugins", id: "engineering/code-maintenance-plugin", document: "README.md", manifest: "plugin.yaml" },
];

for (const subject of readmeSubjects) {
  test(`${subject.module} keep primary documents required when README previews are unavailable`, async () => {
    const registry = await loadRegistry(subject.module);
    const entry = registry.skills.find((item) => item.id === subject.id)!;
    const original = structuredClone(entry);
    const directory = subject.module === "tools" ? "tools-mcp" : subject.module;
    const released = await readReleaseFiles(directory, entry, [subject.document, subject.manifest, "README.md"]);
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "readme-preview-"));
    const previousRoot = process.env.RELEASE_STORE_ROOT;
    try {
      for (const example of unpreviewableExamples) {
        const files = new Map<string, string | Buffer>(released!);
        files.set("README.md", example.content);
        await writeArchive(path.join(root, directory, entry.id, entry.latest, "package.tar.gz"), files);
        process.env.RELEASE_STORE_ROOT = root;

        if (subject.document === "README.md") {
          // Plugins use README as their primary document, so its preview limit
          // must remain enforced rather than silently dropping the content.
          await expect(loadResourceFirstUse(subject.module, entry)).rejects.toThrow();
          continue;
        }

        const content = await loadResourceFirstUse(subject.module, entry);
        expect(content.files?.has("README.md"), example.name).toBe(false);
        expect(content.files?.get(subject.document)).toBe(released!.get(subject.document));
        expect(content.files?.get(subject.manifest)).toBe(released!.get(subject.manifest));
        expect(content.inputs).toContain(subject.module === "skills" ? "google_account_id" : "Inputs are not documented for this release.");
        const declaredOutputs = entry.operational?.outputs;
        expect(content.outputs).toBe(declaredOutputs?.length
          ? declaredOutputs.map((item) => `- ${item}`).join("\n")
          : "Expected outputs are not documented for this release. Inspect its guide before use.");
        expect(content.sources).not.toContain("README.md");
        expect(content.sources).toContain(subject.document);
        expect(content.sample).toBeUndefined();
        expect(entry).toEqual(original);
      }
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
      expect(entry).toEqual(original);

      // Required documents still fail preview validation rather than silently
      // disappearing when they contain the same unpreviewable content.
      files.set("SKILL.md", example.content);
      await writeArchive(path.join(root, "skills", entry.id, entry.latest, "package.tar.gz"), files);
      await expect(loadResourceFirstUse("skills", entry)).rejects.toThrow();
    } finally {
      if (previousRoot === undefined) delete process.env.RELEASE_STORE_ROOT;
      else process.env.RELEASE_STORE_ROOT = previousRoot;
      await fs.rm(root, { recursive: true, force: true });
    }
  });
}
