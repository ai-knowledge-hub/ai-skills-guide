#!/usr/bin/env node
import { readFile } from "node:fs/promises";

const inputPath = process.argv[2];
if (!inputPath) throw new Error("usage: first_use.mjs <input.json>");
const raw = await readFile(inputPath);
if (raw.length > 64 * 1024) throw new Error("first-use input exceeds 64 KiB");
const input = JSON.parse(raw);
for (const field of ["brand", "objective", "approval_owner"]) {
  if (typeof input[field] !== "string" || input[field].trim() === "") throw new Error(`missing ${field}`);
}
process.stdout.write(`${JSON.stringify({
  schema_version: "local-plugin.first-use/v1",
  plugin: "marketing/creative-operating-system-plugin",
  brand: input.brand,
  objective: input.objective,
  workflow: ["complete brand memory", "audit operating system", "triage signals", "draft concept", "run launch gate"],
  approval_owner: input.approval_owner,
  launch_authorized: false
}, null, 2)}\n`);
