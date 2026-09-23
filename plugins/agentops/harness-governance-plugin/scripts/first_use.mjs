#!/usr/bin/env node
import { readFile } from "node:fs/promises";

const inputPath = process.argv[2];
if (!inputPath) throw new Error("usage: first_use.mjs <input.json>");
const raw = await readFile(inputPath);
if (raw.length > 64 * 1024) throw new Error("first-use input exceeds 64 KiB");
const input = JSON.parse(raw);
for (const field of ["run_id", "observation", "proposed_change", "regression_result"]) {
  if (typeof input[field] !== "string" || input[field].trim() === "") throw new Error(`missing ${field}`);
}
const ready = input.regression_result === "pass";
process.stdout.write(`${JSON.stringify({
  schema_version: "local-plugin.first-use/v1",
  plugin: "agentops/harness-governance-plugin",
  run_id: input.run_id,
  decision: ready ? "awaiting-human-approval" : "reject",
  proposed_change: input.proposed_change,
  evidence: { observation: input.observation, regression_result: input.regression_result },
  effect_authorized: false
}, null, 2)}\n`);
