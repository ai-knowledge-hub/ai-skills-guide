#!/usr/bin/env node
import { readFile } from "node:fs/promises";

const inputPath = process.argv[2];
if (!inputPath) throw new Error("usage: first_use.mjs <input.json>");
const raw = await readFile(inputPath);
if (raw.length > 64 * 1024) throw new Error("first-use input exceeds 64 KiB");
const input = JSON.parse(raw);
if (!Array.isArray(input.signals) || input.signals.length === 0 || input.signals.length > 20) throw new Error("signals must contain 1-20 entries");
const signals = input.signals.map((signal, index) => {
  for (const field of ["claim", "source", "observed_at"]) {
    if (typeof signal[field] !== "string" || signal[field].trim() === "") throw new Error(`signals[${index}] missing ${field}`);
  }
  return { ...signal, trust: "untrusted-external", requires_human_review: true };
});
process.stdout.write(`${JSON.stringify({
  schema_version: "local-plugin.first-use/v1",
  plugin: "marketing/competitive-intelligence-plugin",
  subject: input.subject,
  signals,
  publish_authorized: false
}, null, 2)}\n`);
