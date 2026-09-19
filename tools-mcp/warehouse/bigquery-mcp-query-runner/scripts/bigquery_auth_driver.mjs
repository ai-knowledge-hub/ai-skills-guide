#!/usr/bin/env node

import { createHash } from "node:crypto";
import { realpathSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
  BIGQUERY_SCOPE,
  BigQueryError,
  credentialGeneration,
  getAccessToken,
  parseCredentialEnvelope,
  parseRuntimePolicy,
  publicError,
  runQuery,
} from "./bigquery_mcp_server.mjs";

const TOKENINFO_URL = "https://oauth2.googleapis.com/tokeninfo";
const CLOUD_SCOPE = "https://www.googleapis.com/auth/cloud-platform";

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

function methodMatches(method, credential, env) {
  if (method === "service-account" && credential.type !== "service_account") throw new Error("service-account flow requires a service_account envelope");
  if (method === "workload-identity" && !["external_account", "application_default"].includes(credential.type)) throw new Error("workload-identity flow requires an external_account or metadata application_default envelope");
  if (method === "bearer-token" && !["access_token", "authorized_user"].includes(credential.type)) throw new Error("bearer-token flow requires an access_token or authorized_user envelope");
  if (method === "bearer-token" && !parseRuntimePolicy(env.BQ_POLICY).allowUserOAuth) throw new Error("bearer-token flow requires explicit BQ_POLICY approval");
}

function safeIdentity(value) { return typeof value === "string" && value.length >= 1 && value.length <= 256 && !/[\r\n\0]/.test(value); }
function privatePrincipal(kind, value) { return `${kind}-sha256:${createHash("sha256").update(value).digest("hex")}`; }

function principalFromTokenInfo(payload, credentialType) {
  const direct = payload.email ?? payload.sub ?? payload.user_id;
  if (safeIdentity(direct)) return privatePrincipal(payload.email ? "google-user" : "google-sub", direct);
  if (["service_account", "external_account", "application_default"].includes(credentialType)) {
    if (payload.aud !== undefined && !safeIdentity(payload.aud)) throw new BigQueryError("AUTH_IDENTITY_UNVERIFIED", "Google did not attest a usable workload principal");
    if (payload.azp !== undefined && !safeIdentity(payload.azp)) throw new BigQueryError("AUTH_IDENTITY_UNVERIFIED", "Google did not attest a usable workload principal");
    if (payload.aud && payload.azp && payload.aud !== payload.azp) throw new BigQueryError("AUTH_IDENTITY_UNVERIFIED", "Google returned ambiguous workload identity claims");
    const workload = payload.azp ?? payload.aud;
    if (workload) return privatePrincipal("google-workload", workload);
  }
  throw new BigQueryError("AUTH_IDENTITY_UNVERIFIED", "Google did not attest a usable credential principal");
}

function tokenInfoEndpoint(env) {
  const raw = env.BQ_TOKENINFO_URL || TOKENINFO_URL;
  let endpoint;
  try { endpoint = new URL(raw); } catch { throw new BigQueryError("CONFIGURATION_ERROR", "token-info endpoint is invalid"); }
  const loopback = ["localhost", "127.0.0.1", "::1"].includes(endpoint.hostname);
  const testEndpoint = env.BQ_ALLOW_LOOPBACK_TEST_ENDPOINTS === "1" && loopback && endpoint.protocol === "http:";
  if (!testEndpoint && (endpoint.origin !== "https://oauth2.googleapis.com" || endpoint.pathname !== "/tokeninfo" || endpoint.search || endpoint.hash || endpoint.username || endpoint.password)) throw new BigQueryError("CONFIGURATION_ERROR", "token-info endpoint must use the canonical Google endpoint");
  return endpoint;
}

async function introspect(token, credentialType, env, fetchImpl) {
  const endpoint = tokenInfoEndpoint(env);
  endpoint.searchParams.set("access_token", token);
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 10_000);
  let response;
  try { response = await fetchImpl(endpoint, { method: "GET", headers: { accept: "application/json" }, signal: controller.signal }); }
  catch { throw new BigQueryError("AUTH_UNAVAILABLE", "Google token introspection did not complete", { retryable: true }); }
  finally { clearTimeout(timer); }
  const declared = Number(response.headers.get("content-length") ?? 0);
  if (Number.isFinite(declared) && declared > 64 * 1024) throw new BigQueryError("AUTH_UNAVAILABLE", "Google token introspection response exceeded the size limit");
  const chunks = [];
  let size = 0;
  if (response.body) {
    const reader = response.body.getReader();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > 64 * 1024) { await reader.cancel(); throw new BigQueryError("AUTH_UNAVAILABLE", "Google token introspection response exceeded the size limit"); }
      chunks.push(Buffer.from(value));
    }
  }
  let payload;
  try { payload = JSON.parse(Buffer.concat(chunks, size).toString("utf8")); } catch { throw new BigQueryError("AUTH_UNAVAILABLE", "Google token introspection returned malformed JSON"); }
  if (!response.ok) throw new BigQueryError(response.status === 400 || response.status === 401 ? "AUTH_REVOKED" : "AUTH_UNAVAILABLE", "Google rejected token introspection", { status: response.status, retryable: response.status >= 500 });
  if (typeof payload.scope !== "string") throw new BigQueryError("AUTH_SCOPE_UNVERIFIED", "Google did not attest credential scopes");
  const scopes = [...new Set(payload.scope.split(/\s+/).filter(Boolean))];
  if (!scopes.includes(BIGQUERY_SCOPE) && !scopes.includes(CLOUD_SCOPE)) throw new BigQueryError("AUTH_SCOPE_INVALID", "Google did not attest an accepted BigQuery scope");
  // Google's cloud-platform scope authorizes BigQuery and is the normal scope
  // of metadata-server ADC. Expose the narrower effective capability as well
  // so the readiness contract can compare one portable package scope.
  if (scopes.includes(CLOUD_SCOPE) && !scopes.includes(BIGQUERY_SCOPE)) scopes.push(BIGQUERY_SCOPE);
  const expiresIn = Number(payload.expires_in);
  if (!Number.isFinite(expiresIn) || expiresIn <= 0) throw new BigQueryError("AUTH_EXPIRED", "Google reports that the configured credential is expired");
  return { principal: principalFromTokenInfo(payload, credentialType), scopes, expiresIn };
}

async function verify(method, env, fetchImpl, now) {
  const raw = env.BQ_CREDENTIAL;
  const credential = parseCredentialEnvelope(raw);
  const policy = parseRuntimePolicy(env.BQ_POLICY);
  methodMatches(method, credential, env);
  const account = String(credential.account ?? policy.billingProjectID ?? "");
  if (!/^[a-z][a-z0-9-]{4,61}[a-z0-9]$/.test(account) || account !== policy.billingProjectID) throw new Error("credential account must match the policy billing project");
  const auth = await getAccessToken(env, fetchImpl);
  const attestation = await introspect(auth.token, credential.type, env, fetchImpl);
  const projectID = policy.allowedProjectIDs[0];
  const location = policy.allowedLocations[0];
  const report = await runQuery({
    query: "SELECT @probe AS probe",
    parameters: [{ name: "probe", type: "STRING", value: "readiness" }],
    project_id: projectID,
    billing_project_id: account,
    location,
    max_rows: 1,
    page_size: 1,
  }, { env, fetchImpl, now, resolvedAuth: auth });
  if (report.diagnostics.dry_run_bytes_processed !== "0" || report.diagnostics.bytes_processed !== "0" || report.diagnostics.bytes_billed !== "0") throw new BigQueryError("AUTH_PROBE_COST_NONZERO", "BigQuery authentication probe did not preserve the zero-byte cost boundary");
  return { account, ...attestation, generation: credentialGeneration(raw), jobID: report.diagnostics.job_id, observedAt: report.source.retrieved_at };
}

export async function runDriver(method, operation, request = {}, { env = process.env, fetchImpl = fetch, now = () => new Date() } = {}) {
  if (!["bearer-token", "service-account", "workload-identity"].includes(method)) throw new Error("unsupported authentication method");
  if (operation === "credential") throw new Error("credential is resolved by the runtime-owned external environment binding");
  const verified = await verify(method, env, fetchImpl, now);
  if (operation === "bootstrap") {
    const flow = method === "service-account" ? "non-interactive-service" : method === "workload-identity" ? "non-interactive-workload" : "credential-binding";
    if (request.method !== method || request.flow !== flow || !Array.isArray(request.requested_scopes) || request.requested_scopes.length !== 1 || request.requested_scopes[0] !== BIGQUERY_SCOPE || !Array.isArray(request.credential_bindings) || request.credential_bindings.length !== 2 || request.credential_bindings[0] !== "BQ_CREDENTIAL" || request.credential_bindings[1] !== "BQ_POLICY") throw new Error("bootstrap request does not match the packaged authentication contract");
    if (request.expected_account && request.expected_account !== verified.account) throw new Error("selected BigQuery billing project does not match expected account");
    return { completed: true, account: verified.account, scopes: verified.scopes };
  }
  if (operation === "status") {
    const observedAt = now();
    const policy = parseRuntimePolicy(env.BQ_POLICY);
    const expiresAt = new Date(observedAt.valueOf() + Math.min(verified.expiresIn, 300) * 1000).toISOString();
    return {
      authenticated: true, revoked: false, principal: verified.principal, account: verified.account,
      scopes: verified.scopes, expires_at: expiresAt, provider: "google-bigquery", tier: policy.environmentTier,
      endpoint: "https://bigquery.googleapis.com", region: policy.allowedLocations[0], api_version: "v2",
      attestation_reference: `bigquery-job:${verified.jobID}`, attestation_expires_at: expiresAt,
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

function isMainModule() {
  if (!process.argv[1]) return false;
  try { return realpathSync(fileURLToPath(import.meta.url)) === realpathSync(resolve(process.argv[1])); } catch { return false; }
}

if (isMainModule()) main().catch((error) => { process.stderr.write(`${JSON.stringify(publicError(error))}\n`); process.exitCode = 1; });
