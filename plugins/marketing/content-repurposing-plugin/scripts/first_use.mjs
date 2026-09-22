#!/usr/bin/env node
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const [inputPath, outputArgument] = process.argv.slice(2);
if (!inputPath || !outputArgument) throw new Error("usage: first_use.mjs <input.json> <output-directory>");
const raw = await readFile(inputPath);
if (raw.length > 64 * 1024) throw new Error("first-use input exceeds 64 KiB");
const input = JSON.parse(raw);
for (const field of ["title", "audience", "source_summary", "call_to_action"]) {
  if (typeof input[field] !== "string" || input[field].trim() === "" || input[field].length > 4000) throw new Error(`invalid ${field}`);
}

const outputDirectory = path.resolve(outputArgument);
await mkdir(outputDirectory, { recursive: true });
const deliverables = {
  "article.md": `# ${input.title}\n\nAudience: ${input.audience}\n\n${input.source_summary}\n\n## Next step\n\n${input.call_to_action}\n`,
  "social-post.txt": `${input.title}\n\n${input.source_summary}\n\n${input.call_to_action}\n`,
  "email-brief.json": `${JSON.stringify({ subject: input.title, audience: input.audience, body: input.source_summary, call_to_action: input.call_to_action, review_status: "human-review-required" }, null, 2)}\n`
};
for (const [name, contents] of Object.entries(deliverables)) await writeFile(path.join(outputDirectory, name), contents, { flag: "wx" });
const manifest = {
  schema_version: "content-repurposing.deliverable/v1",
  source_title: input.title,
  files: Object.keys(deliverables),
  publish_authorized: false
};
await writeFile(path.join(outputDirectory, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`, { flag: "wx" });
process.stdout.write(`${JSON.stringify(manifest)}\n`);
