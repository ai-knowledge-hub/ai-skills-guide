#!/usr/bin/env node

import { pathToFileURL } from "node:url";

import {
  credentialGeneration,
  getAccessToken,
  introspectAccessToken,
  parseCredentialEnvelope,
  publicError,
  runReport,
} from "./ga4_mcp_server.mjs";

const READ_SCOPE = "https://www.googleapis.com/auth/analytics.readonly";

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

function selectedAccount(credential) {
  const account = String(credential.account ?? "").trim();
  if (!/^\d{4,20}$/.test(account)) throw new Error("credential envelope must declare the numeric GA4 property account");
  return account;
}

function requireMethodCredential(method, credential) {
  if (method === "bearer-token" && credential.type !== "access_token" && credential.type !== "authorized_user") {
    throw new Error("bearer-token flow requires an access_token or authorized_user envelope");
  }
  if (method === "service-account" && credential.type !== "service_account") {
    throw new Error("service-account flow requires a service_account envelope");
  }
}

async function verify(method, env, fetchImpl, now) {
  const raw = env.GA4_CREDENTIAL;
  const credential = parseCredentialEnvelope(raw);
  requireMethodCredential(method, credential);
  const account = selectedAccount(credential);
  const auth = await getAccessToken(env, fetchImpl);
  const attestation = await introspectAccessToken(auth.token, credential.type, env, fetchImpl);
  const report = await runReport({
    property_id: account,
    start_date: new Date(now().valueOf() - 2 * 86_400_000).toISOString().slice(0, 10),
    end_date: new Date(now().valueOf() - 86_400_000).toISOString().slice(0, 10),
    metrics: ["sessions"],
    page_size: 1,
    max_rows: 1,
  }, { env: { ...env, GA4_ALLOWED_PROPERTY_IDS: account }, fetchImpl, now, resolvedAuth: auth });
  return { account, ...attestation, generation: credentialGeneration(raw), observedAt: report.source.retrieved_at };
}

export async function runDriver(method, operation, request = {}, { env = process.env, fetchImpl = fetch, now = () => new Date() } = {}) {
  if (!['bearer-token', 'service-account'].includes(method)) throw new Error("unsupported authentication method");
  if (operation === "credential") {
    throw new Error("credential is resolved by the runtime-owned external environment binding");
  }
  const verified = await verify(method, env, fetchImpl, now);
  if (operation === "bootstrap") {
    const expectedFlow = method === "service-account" ? "non-interactive-service" : "credential-binding";
    if (request.method !== method || request.flow !== expectedFlow ||
        !Array.isArray(request.requested_scopes) || request.requested_scopes.length !== 1 || request.requested_scopes[0] !== READ_SCOPE ||
        !Array.isArray(request.credential_bindings) || request.credential_bindings.length !== 1 || request.credential_bindings[0] !== "GA4_CREDENTIAL") {
      throw new Error("bootstrap request does not match the packaged authentication contract");
    }
    if (request.expected_account && request.expected_account !== verified.account) throw new Error("selected GA4 property does not match expected account");
    return { completed: true, account: verified.account, scopes: verified.scopes };
  }
  if (operation === "status") {
    const observedAt = now();
    const evidenceLifetimeSeconds = Math.min(verified.expiresIn, 5 * 60);
    const explicitExpiry = new Date(observedAt.valueOf() + evidenceLifetimeSeconds * 1000);
    return {
      authenticated: true,
      revoked: false,
      principal: verified.principal,
      account: verified.account,
      scopes: verified.scopes,
      expires_at: explicitExpiry.toISOString(),
      provider: "google-analytics-data-api",
      tier: "standard",
      endpoint: "analyticsdata.googleapis.com",
      region: "global",
      api_version: "v1beta",
      attestation_reference: `ga4-live-${verified.generation}`,
      attestation_expires_at: explicitExpiry.toISOString(),
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
