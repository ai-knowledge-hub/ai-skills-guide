#!/usr/bin/env node

import { pathToFileURL } from "node:url";

import {
  parsePolicy,
  publicError,
  verifyMetaAuthority,
} from "./meta_ads_mcp_server.mjs";

const REQUIRED_SCOPES = Object.freeze(["ads_read", "business_management"]);

async function readInput() {
  const chunks = [];
  let size = 0;
  for await (const chunk of process.stdin) {
    size += chunk.length;
    if (size > 64 * 1024) throw new Error("request exceeds the safe size limit");
    chunks.push(chunk);
  }
  const text = Buffer.concat(chunks).toString("utf8");
  return text ? JSON.parse(text) : {};
}

export async function runDriver(method, operation, request = {}, options = {}) {
  if (method !== "bearer-token") throw new Error("unsupported authentication method");
  if (operation === "credential") throw new Error("credential is resolved by runtime-owned environment bindings");
  const env = options.env ?? process.env;
  const now = options.now ?? (() => new Date());
  const policy = parsePolicy(env.META_ADS_POLICY);
  const verified = await verifyMetaAuthority({ ...options, env, now });
  if (operation === "bootstrap") {
    if (request.method !== method || request.flow !== "credential-binding" ||
        !Array.isArray(request.requested_scopes) || request.requested_scopes.length !== REQUIRED_SCOPES.length || REQUIRED_SCOPES.some((scope) => !request.requested_scopes.includes(scope)) ||
        !Array.isArray(request.credential_bindings) || request.credential_bindings.length !== 2 || !request.credential_bindings.includes("META_ADS_CREDENTIAL") || !request.credential_bindings.includes("META_ADS_POLICY")) {
      throw new Error("bootstrap request does not match the packaged authentication contract");
    }
    if (request.expected_account && request.expected_account !== verified.accountID) throw new Error("selected Meta ad account does not match expected account");
    return { completed: true, account: verified.accountID, scopes: verified.scopes };
  }
  if (operation === "status") {
    const observedAt = now();
    const localMaximum = new Date(observedAt.valueOf() + 5 * 60 * 1000);
    const providerExpiry = verified.providerExpiresAt === null ? localMaximum : new Date(verified.providerExpiresAt);
    if (Number.isNaN(providerExpiry.valueOf()) || providerExpiry <= observedAt) throw new Error("provider authentication expired during verification");
    const expiresAt = new Date(Math.min(providerExpiry.valueOf(), localMaximum.valueOf()));
    return {
      authenticated: true,
      revoked: false,
      principal: verified.principal,
      account: verified.accountID,
      scopes: verified.scopes,
      expires_at: expiresAt.toISOString(),
      provider: "meta-marketing-api",
      tier: policy.environmentTier,
      endpoint: "graph.facebook.com",
      region: "global",
      api_version: policy.graphVersion,
      attestation_reference: verified.attestationReference,
      attestation_expires_at: expiresAt.toISOString(),
    };
  }
  throw new Error("unsupported authentication driver operation");
}

async function main() {
  const [method, operation] = process.argv.slice(2);
  const request = await readInput();
  const response = await runDriver(method, operation, request);
  process.stdout.write(`${JSON.stringify(response)}\n`);
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) {
  main().catch((error) => {
    process.stderr.write(`${JSON.stringify(publicError(error))}\n`);
    process.exitCode = 1;
  });
}
