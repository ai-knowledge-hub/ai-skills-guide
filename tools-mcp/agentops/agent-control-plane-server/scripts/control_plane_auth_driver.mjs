#!/usr/bin/env node

import { pathToFileURL } from "node:url";

import { contextFromEnv, publicError, verifyAuditChain } from "./control_plane_core.mjs";

async function readInput() {
  const chunks = [];
  let size = 0;
  for await (const chunk of process.stdin) {
    size += chunk.length;
    if (size > 64 * 1024) throw new Error("request exceeds the safe size limit");
    chunks.push(chunk);
  }
  return JSON.parse(Buffer.concat(chunks).toString("utf8") || "{}");
}

export async function runDriver(method, operation, request = {}, options = {}) {
  if (method !== "custom") throw new Error("unsupported authentication method");
  if (operation === "credential") throw new Error("identity is resolved by runtime-owned environment bindings");
  const now = options.now ?? (() => new Date());
  const ctx = options.context ?? contextFromEnv(options.env ?? process.env, { now });
  if (operation === "bootstrap") {
    if (request.method !== "custom" || request.flow !== "custom" || !Array.isArray(request.credential_bindings) || request.credential_bindings.length !== 5 ||
        !["CONTROL_PLANE_IDENTITY", "CONTROL_PLANE_IDENTITY_PUBLIC_KEY", "CONTROL_PLANE_GRANT_ROLE_KEY", "CONTROL_PLANE_AUDIT_KEY", "CONTROL_PLANE_STORE"].every((binding) => request.credential_bindings.includes(binding))) {
      throw new Error("bootstrap request does not match the packaged authentication contract");
    }
    return { completed: true, account: `${ctx.identity.tenant_id}:${ctx.identity.actor_id}`, scopes: ctx.identity.capabilities };
  }
  if (operation === "status") {
    const audit = await verifyAuditChain(ctx);
    if (!audit.valid) throw new Error("tenant audit chain verification failed");
    const identityExpiry = new Date(ctx.identity.expires_at);
    const localMaximum = new Date(now().valueOf() + 5 * 60 * 1000);
    const expiresAt = new Date(Math.min(identityExpiry.valueOf(), localMaximum.valueOf()));
    return {
      authenticated: true,
      revoked: false,
      principal: ctx.identity.actor_id,
      account: ctx.identity.tenant_id,
      scopes: ctx.identity.capabilities,
      expires_at: expiresAt.toISOString(),
      provider: "local-agent-control-plane",
      tier: "local",
      endpoint: "stdio",
      region: "local",
      api_version: "control-plane/v1",
      attestation_reference: `audit-head:${audit.head_hash ?? "empty"}`,
      attestation_expires_at: expiresAt.toISOString()
    };
  }
  throw new Error("unsupported authentication driver operation");
}

async function main() {
  const [method, operation] = process.argv.slice(2);
  process.stdout.write(`${JSON.stringify(await runDriver(method, operation, await readInput()))}\n`);
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) {
  main().catch((error) => {
    process.stderr.write(`${JSON.stringify(publicError(error))}\n`);
    process.exitCode = 1;
  });
}
