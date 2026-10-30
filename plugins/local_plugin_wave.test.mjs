import assert from "node:assert/strict";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const pluginsRoot = path.dirname(fileURLToPath(import.meta.url));
const scenarios = [
  ["agentops/harness-governance-plugin", "effect_authorized", false],
  ["engineering/code-maintenance-plugin", "merge_authorized", false],
  ["marketing/competitive-intelligence-plugin", "publish_authorized", false],
  ["marketing/creative-operating-system-plugin", "launch_authorized", false],
  ["security/runtime-safety-plugin", "remediation_authorized", false]
];

for (const [plugin, authorityField, authorityValue] of scenarios) {
  test(`${plugin} has a runnable, bounded first-use scenario`, () => {
    const root = path.join(pluginsRoot, plugin);
    const result = spawnSync(process.execPath, [
      path.join(root, "scripts", "first_use.mjs"),
      path.join(root, "examples", "first-use-input.json")
    ], { encoding: "utf8", timeout: 5000 });
    assert.equal(result.status, 0, result.stderr);
    const output = JSON.parse(result.stdout);
    assert.equal(output.plugin, plugin);
    assert.equal(output[authorityField], authorityValue);
  });
}

test("content repurposing creates the documented multi-format deliverable without dependencies", async () => {
  const root = path.join(pluginsRoot, "marketing", "content-repurposing-plugin");
  const output = await mkdtemp(path.join(tmpdir(), "content-repurposing-first-use-"));
  try {
    const result = spawnSync(process.execPath, [
      path.join(root, "scripts", "first_use.mjs"),
      path.join(root, "examples", "first-use-input.json"),
      output
    ], { encoding: "utf8", timeout: 5000 });
    assert.equal(result.status, 0, result.stderr);
    const manifest = JSON.parse(await readFile(path.join(output, "manifest.json"), "utf8"));
    assert.deepEqual(manifest.files, ["article.md", "social-post.txt", "email-brief.json"]);
    assert.equal(manifest.publish_authorized, false);
    assert.match(await readFile(path.join(output, "article.md"), "utf8"), /governed AI workflows/);
    assert.match(await readFile(path.join(output, "social-post.txt"), "utf8"), /explicit approval boundaries/);
    assert.equal(JSON.parse(await readFile(path.join(output, "email-brief.json"), "utf8")).review_status, "human-review-required");
  } finally {
    await rm(output, { recursive: true, force: true });
  }
});

test("first-use scenarios reject oversized input", async () => {
  const root = path.join(pluginsRoot, "security", "runtime-safety-plugin");
  const directory = await mkdtemp(path.join(tmpdir(), "plugin-first-use-bounds-"));
  const input = path.join(directory, "oversized.json");
  try {
    await writeFile(input, JSON.stringify({ findings: [], padding: "x".repeat(65 * 1024) }));
    const result = spawnSync(process.execPath, [path.join(root, "scripts", "first_use.mjs"), input], {
      encoding: "utf8",
      timeout: 5000
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /exceeds 64 KiB/);
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});
