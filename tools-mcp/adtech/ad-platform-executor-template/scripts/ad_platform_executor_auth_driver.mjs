#!/usr/bin/env node

import { createInterface } from "node:readline";
import { pathToFileURL } from "node:url";

import { contextFromEnv, publicError, verifyExecutorAuthority } from "./ad_platform_executor_core.mjs";

const BINDINGS = Object.freeze([
  "AD_PLATFORM_PROVIDER_CREDENTIAL", "AD_PLATFORM_EXECUTOR_POLICY", "AD_PLATFORM_EXECUTOR_STORE", "AD_PLATFORM_EXECUTOR_AUDIT_KEY",
  "CONTROL_PLANE_PACKAGE_ROOT", "CONTROL_PLANE_PACKAGE_INTEGRITY", "CONTROL_PLANE_STORE", "CONTROL_PLANE_IDENTITY", "CONTROL_PLANE_IDENTITY_PUBLIC_KEY", "CONTROL_PLANE_GRANT_ROLE_KEY", "CONTROL_PLANE_AUDIT_KEY",
]);

export async function runDriver(method, operation, request, options = {}) {
  if (method !== "custom") throw new Error("unsupported authentication method");
  if (operation === "credential") throw new Error("credentials are resolved by runtime-owned environment bindings");
  if (operation === "bootstrap") {
    if (request?.method !== "custom" || request?.flow !== "custom" || !Array.isArray(request.credential_bindings) || request.credential_bindings.length !== BINDINGS.length || !BINDINGS.every((name) => request.credential_bindings.includes(name))) throw new Error("executor bootstrap request is incomplete");
  } else if (operation !== "status") throw new Error("unsupported authentication operation");
  const authority = await verifyExecutorAuthority(contextFromEnv(options.env, options));
  if (operation === "bootstrap") return { completed: true, account: authority.account, scopes: authority.scopes };
  const observedAt = (options.now ?? (() => new Date()))();
  const expiresAt = new Date(Math.min(observedAt.valueOf() + 5 * 60_000, new Date(authority.identityExpiresAt).valueOf())).toISOString();
  return {
    authenticated: true,
    principal: authority.principal,
    account: authority.account,
    scopes: authority.scopes,
    expires_at: expiresAt,
    credential_generation: authority.credentialGeneration,
    tier: authority.tier,
    attestation_reference: `control-plane-provider:${authority.targetFingerprint.slice(7)}`,
    attestation_expires_at: expiresAt,
  };
}

async function main() {
  const method = process.argv[2];
  const operation = process.argv[3];
  let request = {};
  const lines = createInterface({ input: process.stdin, crlfDelay: Infinity });
  for await (const line of lines) { if (line.trim()) request = JSON.parse(line); break; }
  process.stdout.write(`${JSON.stringify(await runDriver(method, operation, request))}\n`);
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) main().catch((error) => { process.stderr.write(`${JSON.stringify(publicError(error))}\n`); process.exitCode = 1; });
