#!/usr/bin/env node

import { readFile } from "node:fs/promises";
import { pathToFileURL } from "node:url";

import { contextFromEnv, decideApproval, installPolicy, publicError, signIdentity } from "./control_plane_core.mjs";

async function readStdin() {
  const chunks = [];
  let size = 0;
  for await (const chunk of process.stdin) {
    size += chunk.length;
    if (size > 1024 * 1024) throw new Error("administrator input exceeds the safe size limit");
    chunks.push(chunk);
  }
  return JSON.parse(Buffer.concat(chunks).toString("utf8"));
}

export async function runAdmin(command, input, options = {}) {
  const env = options.env ?? process.env;
  if (command === "mint-identity") return signIdentity(input, env.CONTROL_PLANE_IDENTITY_PRIVATE_KEY);
  const ctx = options.context ?? contextFromEnv(env, { admin: true, now: options.now, fault: options.fault });
  if (command === "install-policy") return installPolicy(ctx, input);
  if (command === "decide-approval") return decideApproval(ctx, input);
  throw new Error("unsupported administrator command");
}

async function main() {
  const command = process.argv[2];
  const fileIndex = process.argv.indexOf("--file");
  const input = fileIndex >= 0 ? JSON.parse(await readFile(process.argv[fileIndex + 1], "utf8")) : await readStdin();
  process.stdout.write(`${JSON.stringify(await runAdmin(command, input))}\n`);
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) {
  main().catch((error) => {
    process.stderr.write(`${JSON.stringify(publicError(error))}\n`);
    process.exitCode = 1;
  });
}
