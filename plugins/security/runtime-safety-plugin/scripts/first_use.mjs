#!/usr/bin/env node
import { readFile } from "node:fs/promises";

const inputPath = process.argv[2];
if (!inputPath) throw new Error("usage: first_use.mjs <input.json>");
const raw = await readFile(inputPath);
if (raw.length > 64 * 1024) throw new Error("first-use input exceeds 64 KiB");
const input = JSON.parse(raw);
if (!Array.isArray(input.findings) || input.findings.length === 0 || input.findings.length > 50) throw new Error("findings must contain 1-50 entries");
const severity = new Set(["low", "medium", "high", "critical"]);
for (const [index, finding] of input.findings.entries()) {
  if (!severity.has(finding.severity) || typeof finding.summary !== "string" || finding.summary.trim() === "") throw new Error(`invalid findings[${index}]`);
}
const halt = input.findings.some((finding) => finding.severity === "high" || finding.severity === "critical");
process.stdout.write(`${JSON.stringify({
  schema_version: "local-plugin.first-use/v1",
  plugin: "security/runtime-safety-plugin",
  decision: halt ? "halt-and-escalate" : "review-before-proceeding",
  findings: input.findings,
  remediation_authorized: false
}, null, 2)}\n`);
