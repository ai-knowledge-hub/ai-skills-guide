#!/usr/bin/env node

import { createHash, createSign, randomUUID } from "node:crypto";
import { realpathSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const BIGQUERY_SCOPE = "https://www.googleapis.com/auth/bigquery";
const CLOUD_SCOPE = "https://www.googleapis.com/auth/cloud-platform";
const DEFAULT_API_BASE = "https://bigquery.googleapis.com";
const DEFAULT_TOKEN_URL = "https://oauth2.googleapis.com/token";
const DEFAULT_TOKENINFO_URL = "https://oauth2.googleapis.com/tokeninfo";
const DEFAULT_STS_URL = "https://sts.googleapis.com/v1/token";
const DEFAULT_METADATA_TOKEN_URL = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token";
const MAX_UPSTREAM_BYTES = 2 * 1024 * 1024;
const MAX_MCP_BYTES = 2 * 1024 * 1024;
const MAX_MCP_REQUEST_BYTES = 256 * 1024;
const MAX_NORMALIZED_ROW_BYTES = Math.floor((MAX_MCP_BYTES - 128 * 1024) / 2);
const MAX_SQL_BYTES = 128 * 1024;
const MAX_ROWS = 1_000;
const MAX_PAGE_SIZE = 1_000;
const MAX_PAGES = 20;
const MAX_TIMEOUT_MS = 120_000;
const MAX_PARAMETERS = 100;
const PARAMETER_TYPES = new Set(["STRING", "BOOL", "INT64", "FLOAT64", "NUMERIC", "BIGNUMERIC", "BYTES", "DATE", "DATETIME", "TIME", "TIMESTAMP", "GEOGRAPHY"]);
const FORBIDDEN_SQL = new Set([
  "ALTER", "ASSERT", "BEGIN", "CALL", "COMMIT", "CREATE", "DELETE", "DROP", "EXECUTE",
  "EXPORT", "GRANT", "INSERT", "LOAD", "MERGE", "REVOKE", "ROLLBACK", "TRUNCATE", "UPDATE",
]);
const FORBIDDEN_FUNCTIONS = new Set(["EXTERNAL_QUERY", "ML.PREDICT", "REMOTE_FUNCTION"]);

export class BigQueryError extends Error {
  constructor(code, message, { retryable = false, status = 0, details = undefined } = {}) {
    super(message);
    this.name = "BigQueryError";
    this.code = code;
    this.retryable = retryable;
    this.status = status;
    this.details = details;
  }
}

function plainObject(value, label) {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new BigQueryError("INVALID_ARGUMENT", `${label} must be an object`);
  return value;
}

function exactKeys(value, allowed, label) {
  for (const key of Object.keys(value)) if (!allowed.includes(key)) throw new BigQueryError("INVALID_ARGUMENT", `${label} contains unsupported field ${key}`);
}

function positiveInteger(value, fallback, maximum, label) {
  const parsed = value === undefined ? fallback : Number(value);
  if (!Number.isSafeInteger(parsed) || parsed < 1 || parsed > maximum) throw new BigQueryError("INVALID_ARGUMENT", `${label} must be an integer from 1 to ${maximum}`);
  return parsed;
}

function requiredEnvList(raw, pattern, label) {
  const values = String(raw ?? "").split(",").map((value) => value.trim()).filter(Boolean);
  if (values.length === 0 || values.length > 100 || values.some((value) => !pattern.test(value)) || new Set(values).size !== values.length) {
    throw new BigQueryError("CONFIGURATION_ERROR", `${label} must contain a unique, comma-separated allowlist`);
  }
  return values;
}

export function parseRuntimePolicy(raw) {
  if (typeof raw !== "string" || raw.length < 2 || raw.length > 64 * 1024) throw new BigQueryError("CONFIGURATION_ERROR", "BQ_POLICY is unavailable or exceeds the safe size limit");
  let policy;
  try { policy = JSON.parse(raw); } catch { throw new BigQueryError("CONFIGURATION_ERROR", "BQ_POLICY must be a JSON policy envelope"); }
  plainObject(policy, "BQ_POLICY");
  exactKeys(policy, ["allowed_project_ids", "billing_project_id", "allowed_datasets", "allowed_locations", "max_bytes_billed", "max_rows", "max_timeout_ms", "allow_user_oauth", "environment_tier"], "BQ_POLICY");
  const list = (value, pattern, label) => {
    if (!Array.isArray(value) || value.length < 1 || value.length > 100 || value.some((item) => typeof item !== "string" || !pattern.test(item)) || new Set(value).size !== value.length) throw new BigQueryError("CONFIGURATION_ERROR", `${label} must contain a unique allowlist`);
    return [...value];
  };
  const allowedProjectIDs = list(policy.allowed_project_ids, /^[a-z][a-z0-9-]{4,61}[a-z0-9]$/, "allowed_project_ids");
  const billingProjectID = String(policy.billing_project_id ?? "");
  if (!allowedProjectIDs.includes(billingProjectID)) throw new BigQueryError("CONFIGURATION_ERROR", "billing_project_id must be in allowed_project_ids");
  if (policy.environment_tier !== "sandbox" && policy.environment_tier !== "live") throw new BigQueryError("CONFIGURATION_ERROR", "environment_tier must be sandbox or live");
  return {
    allowedProjectIDs,
    billingProjectID,
    allowedDatasets: list(policy.allowed_datasets, /^[a-z][a-z0-9-]{4,61}[a-z0-9]\.[A-Za-z_][A-Za-z0-9_]{0,1023}$/, "allowed_datasets"),
    allowedLocations: list(policy.allowed_locations, /^[A-Za-z][A-Za-z0-9-]{0,62}$/, "allowed_locations"),
    maxBytesBilled: positiveInteger(policy.max_bytes_billed, undefined, Number.MAX_SAFE_INTEGER, "max_bytes_billed"),
    maxRows: positiveInteger(policy.max_rows, MAX_ROWS, MAX_ROWS, "max_rows"),
    maxTimeoutMs: positiveInteger(policy.max_timeout_ms, 30_000, MAX_TIMEOUT_MS, "max_timeout_ms"),
    allowUserOAuth: policy.allow_user_oauth === true,
    environmentTier: policy.environment_tier,
  };
}

function trustedEndpoint(raw, fallback, label, env, { metadata = false } = {}) {
  let endpoint;
  try { endpoint = new URL(raw || fallback); } catch { throw new BigQueryError("CONFIGURATION_ERROR", `${label} is invalid`); }
  if (endpoint.username || endpoint.password || endpoint.hash) throw new BigQueryError("CONFIGURATION_ERROR", `${label} must not include credentials or a fragment`);
  const canonical = new URL(fallback);
  const loopback = ["localhost", "127.0.0.1", "::1"].includes(endpoint.hostname);
  const allowedLoopback = Boolean(raw) && env.BQ_ALLOW_LOOPBACK_TEST_ENDPOINTS === "1" && loopback && endpoint.protocol === "http:";
  const canonicalMatch = endpoint.origin === canonical.origin && endpoint.pathname === canonical.pathname && endpoint.search === "";
  if (!allowedLoopback && (!canonicalMatch || (!metadata && endpoint.protocol !== "https:"))) {
    throw new BigQueryError("CONFIGURATION_ERROR", `${label} must use the canonical Google endpoint`);
  }
  return endpoint;
}

async function boundedFetch(url, options, { timeoutMs, fetchImpl }) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try { return await fetchImpl(url, { ...options, signal: controller.signal }); }
  catch (error) {
    if (error?.name === "AbortError") throw new BigQueryError("UPSTREAM_TIMEOUT", "BigQuery request exceeded the configured timeout", { retryable: true });
    throw new BigQueryError("UPSTREAM_UNAVAILABLE", "BigQuery request could not be completed", { retryable: true });
  } finally { clearTimeout(timer); }
}

async function readBoundedJSON(response) {
  const declared = Number(response.headers.get("content-length") ?? 0);
  if (Number.isFinite(declared) && declared > MAX_UPSTREAM_BYTES) throw new BigQueryError("UPSTREAM_RESPONSE_TOO_LARGE", "BigQuery response exceeded the size limit");
  const chunks = [];
  let size = 0;
  if (response.body) {
    const reader = response.body.getReader();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > MAX_UPSTREAM_BYTES) { await reader.cancel(); throw new BigQueryError("UPSTREAM_RESPONSE_TOO_LARGE", "BigQuery response exceeded the size limit"); }
      chunks.push(Buffer.from(value));
    }
  }
  try { return JSON.parse(Buffer.concat(chunks, size).toString("utf8")); }
  catch { throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery returned malformed JSON"); }
}

function normalizeProviderFailure(response, payload, fallback = "BigQuery rejected the query request") {
  const reason = payload?.error?.errors?.[0]?.reason ?? payload?.error?.status ?? "";
  if (response.status === 401 || reason === "UNAUTHENTICATED") return new BigQueryError("AUTH_REVOKED", "Google rejected the configured credential", { status: response.status });
  if (response.status === 403 || reason === "accessDenied") return new BigQueryError("ACCESS_DENIED", "credential cannot access the requested BigQuery resources", { status: response.status });
  if (response.status === 429 || ["rateLimitExceeded", "quotaExceeded", "RESOURCE_EXHAUSTED"].includes(reason)) return new BigQueryError("QUOTA_EXHAUSTED", "BigQuery quota is exhausted", { status: response.status, retryable: true });
  if (reason === "billingTierLimitExceeded") return new BigQueryError("BYTE_BUDGET_EXCEEDED", "BigQuery rejected the configured byte budget", { status: response.status });
  return new BigQueryError("UPSTREAM_ERROR", fallback, { status: response.status, retryable: response.status >= 500 });
}

function base64url(value) { return Buffer.from(value).toString("base64url"); }
export function credentialGeneration(value) { return createHash("sha256").update(value).digest("hex").slice(0, 24); }

export function parseCredentialEnvelope(raw) {
  if (typeof raw !== "string" || raw.length < 1 || raw.length > 64 * 1024) throw new BigQueryError("AUTH_MISSING", "BQ_CREDENTIAL is unavailable or exceeds the safe size limit");
  let credential;
  try { credential = JSON.parse(raw); } catch { throw new BigQueryError("AUTH_INVALID", "BQ_CREDENTIAL must be a JSON credential envelope"); }
  plainObject(credential, "BQ_CREDENTIAL");
  if (credential.type === "access_token") {
    exactKeys(credential, ["type", "access_token", "expires_at", "account"], "access-token credential");
    if (typeof credential.access_token !== "string" || credential.access_token.length < 12) throw new BigQueryError("AUTH_INVALID", "access-token credential is incomplete");
  } else if (credential.type === "authorized_user") {
    exactKeys(credential, ["type", "client_id", "client_secret", "refresh_token", "token_uri", "account"], "authorized-user credential");
    for (const field of ["client_id", "client_secret", "refresh_token"]) if (typeof credential[field] !== "string" || credential[field].length < 8) throw new BigQueryError("AUTH_INVALID", "authorized-user credential is incomplete");
  } else if (credential.type === "service_account") {
    exactKeys(credential, ["type", "client_email", "private_key", "private_key_id", "token_uri", "project_id", "account"], "service-account credential");
    if (typeof credential.client_email !== "string" || !credential.client_email.endsWith(".gserviceaccount.com") || typeof credential.private_key !== "string" || !credential.private_key.includes("BEGIN PRIVATE KEY")) throw new BigQueryError("AUTH_INVALID", "service-account credential is incomplete");
  } else if (credential.type === "external_account") {
    exactKeys(credential, ["type", "audience", "subject_token_type", "subject_token", "token_url", "service_account_impersonation_url", "account"], "external-account credential");
    if (typeof credential.audience !== "string" || !credential.audience.startsWith("//iam.googleapis.com/projects/") || !["urn:ietf:params:oauth:token-type:jwt", "urn:ietf:params:oauth:token-type:id_token", "urn:ietf:params:oauth:token-type:saml2"].includes(credential.subject_token_type) || typeof credential.subject_token !== "string" || credential.subject_token.length < 8 || credential.subject_token.length > 64 * 1024) throw new BigQueryError("AUTH_INVALID", "external-account credential is incomplete");
    let impersonation;
    try { impersonation = new URL(credential.service_account_impersonation_url); } catch { throw new BigQueryError("AUTH_INVALID", "external-account impersonation URL is invalid"); }
    if (impersonation.origin !== "https://iamcredentials.googleapis.com" || !/^\/v1\/projects\/-\/serviceAccounts\/[A-Za-z0-9._%+@-]+:generateAccessToken$/.test(impersonation.pathname) || impersonation.search || impersonation.hash || impersonation.username || impersonation.password) throw new BigQueryError("AUTH_INVALID", "external-account impersonation URL must use the canonical Google endpoint");
  } else if (credential.type === "application_default") {
    exactKeys(credential, ["type", "source", "account"], "application-default credential");
    if (credential.source !== "metadata") throw new BigQueryError("AUTH_INVALID", "application-default source must be metadata");
  } else throw new BigQueryError("AUTH_INVALID", "unsupported BigQuery credential type");
  if (credential.account !== undefined && (typeof credential.account !== "string" || !/^[A-Za-z0-9._:@/-]{1,256}$/.test(credential.account))) throw new BigQueryError("AUTH_INVALID", "credential account is invalid");
  return credential;
}

export async function getAccessToken(env = process.env, fetchImpl = fetch) {
  const raw = env.BQ_CREDENTIAL;
  const credential = parseCredentialEnvelope(raw);
  const policy = parseRuntimePolicy(env.BQ_POLICY);
  if (["access_token", "authorized_user"].includes(credential.type) && !policy.allowUserOAuth) throw new BigQueryError("AUTH_METHOD_NOT_ALLOWED", "user OAuth requires explicit BQ_POLICY approval");
  if (credential.type === "access_token") {
    if (credential.expires_at && new Date(credential.expires_at) <= new Date()) throw new BigQueryError("AUTH_EXPIRED", "configured access token is expired");
    return { token: credential.access_token, credential, generation: credentialGeneration(raw) };
  }
  if (credential.type === "application_default") {
    const endpoint = trustedEndpoint(env.BQ_METADATA_TOKEN_URL, DEFAULT_METADATA_TOKEN_URL, "metadata token endpoint", env, { metadata: true });
    const response = await boundedFetch(endpoint, { method: "GET", headers: { "metadata-flavor": "Google", accept: "application/json" } }, { timeoutMs: 5_000, fetchImpl });
    const payload = await readBoundedJSON(response);
    if (!response.ok || typeof payload.access_token !== "string") throw normalizeProviderFailure(response, payload, "Google metadata authentication did not complete");
    return { token: payload.access_token, credential, generation: credentialGeneration(raw), expiresIn: payload.expires_in };
  }
  if (credential.type === "external_account") {
    const stsEndpoint = trustedEndpoint(env.BQ_STS_URL || credential.token_url, DEFAULT_STS_URL, "STS endpoint", env);
    const exchange = new URLSearchParams({
      grant_type: "urn:ietf:params:oauth:grant-type:token-exchange",
      requested_token_type: "urn:ietf:params:oauth:token-type:access_token",
      audience: credential.audience,
      scope: CLOUD_SCOPE,
      subject_token_type: credential.subject_token_type,
      subject_token: credential.subject_token,
    });
    const stsResponse = await boundedFetch(stsEndpoint, { method: "POST", headers: { "content-type": "application/x-www-form-urlencoded" }, body: exchange }, { timeoutMs: 10_000, fetchImpl });
    const stsPayload = await readBoundedJSON(stsResponse);
    if (!stsResponse.ok || typeof stsPayload.access_token !== "string") throw normalizeProviderFailure(stsResponse, stsPayload, "Google workload token exchange did not complete");
    const impersonation = new URL(credential.service_account_impersonation_url);
    const impResponse = await boundedFetch(impersonation, { method: "POST", headers: { authorization: `Bearer ${stsPayload.access_token}`, "content-type": "application/json" }, body: JSON.stringify({ scope: [BIGQUERY_SCOPE], lifetime: "3600s" }) }, { timeoutMs: 10_000, fetchImpl });
    const impPayload = await readBoundedJSON(impResponse);
    if (!impResponse.ok || typeof impPayload.accessToken !== "string") throw normalizeProviderFailure(impResponse, impPayload, "Google service-account impersonation did not complete");
    return { token: impPayload.accessToken, credential, generation: credentialGeneration(raw), expiresAt: impPayload.expireTime };
  }
  const tokenEndpoint = trustedEndpoint(env.BQ_TOKEN_URL || credential.token_uri, DEFAULT_TOKEN_URL, "token endpoint", env);
  const body = new URLSearchParams();
  if (credential.type === "authorized_user") {
    body.set("grant_type", "refresh_token"); body.set("client_id", credential.client_id); body.set("client_secret", credential.client_secret); body.set("refresh_token", credential.refresh_token);
  } else {
    const now = Math.floor(Date.now() / 1000);
    const header = { alg: "RS256", typ: "JWT", ...(credential.private_key_id ? { kid: credential.private_key_id } : {}) };
    const claim = { iss: credential.client_email, scope: BIGQUERY_SCOPE, aud: tokenEndpoint.toString(), iat: now, exp: now + 3600 };
    const unsigned = `${base64url(JSON.stringify(header))}.${base64url(JSON.stringify(claim))}`;
    let signature;
    try { const signer = createSign("RSA-SHA256"); signer.update(unsigned); signer.end(); signature = signer.sign(credential.private_key).toString("base64url"); }
    catch { throw new BigQueryError("AUTH_INVALID", "service-account private key is invalid"); }
    body.set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer"); body.set("assertion", `${unsigned}.${signature}`);
  }
  const response = await boundedFetch(tokenEndpoint, { method: "POST", headers: { "content-type": "application/x-www-form-urlencoded" }, body }, { timeoutMs: 10_000, fetchImpl });
  const payload = await readBoundedJSON(response);
  if (!response.ok || typeof payload.access_token !== "string") throw normalizeProviderFailure(response, payload, "Google token exchange did not complete");
  return { token: payload.access_token, credential, generation: credentialGeneration(raw), expiresIn: payload.expires_in };
}

function lexSQL(sql) {
  if (typeof sql !== "string" || Buffer.byteLength(sql, "utf8") < 1 || Buffer.byteLength(sql, "utf8") > MAX_SQL_BYTES) throw new BigQueryError("INVALID_ARGUMENT", "query must contain 1 to 131072 UTF-8 bytes");
  const code = [];
  const identifiers = [];
  let i = 0;
  while (i < sql.length) {
    const char = sql[i];
    if (char === "'" || char === '"') {
      const quote = char; code.push(" "); i += 1; let closed = false;
      while (i < sql.length) { if (sql[i] === "\\") { i += 2; continue; } if (sql[i] === quote) { if (sql[i + 1] === quote) { i += 2; continue; } i += 1; closed = true; break; } i += 1; }
      if (!closed) throw new BigQueryError("QUERY_NOT_ALLOWED", "query contains an unterminated string literal");
      continue;
    }
    if (char === "`") {
      i += 1; let identifier = ""; let closed = false;
      while (i < sql.length) { if (sql[i] === "`") { i += 1; closed = true; break; } identifier += sql[i]; i += 1; }
      if (!closed) throw new BigQueryError("QUERY_NOT_ALLOWED", "query contains an unterminated quoted identifier");
      identifiers.push(identifier); code.push(` __BQ_IDENTIFIER_${identifiers.length - 1}__ `); continue;
    }
    if (char === "-" && sql[i + 1] === "-") { while (i < sql.length && sql[i] !== "\n") i += 1; code.push(" "); continue; }
    if (char === "#") { while (i < sql.length && sql[i] !== "\n") i += 1; code.push(" "); continue; }
    if (char === "/" && sql[i + 1] === "*") { const end = sql.indexOf("*/", i + 2); if (end < 0) throw new BigQueryError("QUERY_NOT_ALLOWED", "query contains an unterminated comment"); i = end + 2; code.push(" "); continue; }
    code.push(char); i += 1;
  }
  const normalized = code.join("");
  if (normalized.includes(";")) throw new BigQueryError("QUERY_NOT_ALLOWED", "multiple statements and statement terminators are not allowed");
  const tokens = normalized.match(/__BQ_IDENTIFIER_\d+__|@[A-Za-z_][A-Za-z0-9_]*|[A-Za-z_][A-Za-z0-9_.$]*|\S/g) ?? [];
  return { normalized, tokens, identifiers };
}

function parseParameters(raw, referenced) {
  const parameters = raw ?? [];
  if (!Array.isArray(parameters) || parameters.length > MAX_PARAMETERS) throw new BigQueryError("INVALID_ARGUMENT", `parameters must contain at most ${MAX_PARAMETERS} entries`);
  const seen = new Set();
  const queryParameters = parameters.map((item) => {
    plainObject(item, "parameter"); exactKeys(item, ["name", "type", "value"], "parameter");
    if (typeof item.name !== "string" || !/^[A-Za-z_][A-Za-z0-9_]{0,127}$/.test(item.name) || seen.has(item.name)) throw new BigQueryError("INVALID_ARGUMENT", "parameter name is invalid or duplicated");
    if (typeof item.type !== "string" || !PARAMETER_TYPES.has(item.type)) throw new BigQueryError("INVALID_ARGUMENT", "parameter type is unsupported");
    if (item.value === null || !["string", "number", "boolean"].includes(typeof item.value)) throw new BigQueryError("INVALID_ARGUMENT", "parameter value must be a non-null scalar");
    if (typeof item.value === "string" && Buffer.byteLength(item.value, "utf8") > 16 * 1024) throw new BigQueryError("INVALID_ARGUMENT", "parameter value exceeds the safe size limit");
    if (item.type === "BOOL" && typeof item.value !== "boolean") throw new BigQueryError("INVALID_ARGUMENT", "BOOL parameter value must be boolean");
    if (item.type === "FLOAT64" && (typeof item.value !== "number" || !Number.isFinite(item.value))) throw new BigQueryError("INVALID_ARGUMENT", "FLOAT64 parameter value must be finite");
    if (item.type === "INT64" && !((typeof item.value === "number" && Number.isSafeInteger(item.value)) || (typeof item.value === "string" && /^-?(?:0|[1-9][0-9]*)$/.test(item.value)))) throw new BigQueryError("INVALID_ARGUMENT", "INT64 parameter value must be a safe integer or canonical integer string");
    if (["NUMERIC", "BIGNUMERIC"].includes(item.type) && (typeof item.value !== "string" || !/^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$/.test(item.value))) throw new BigQueryError("INVALID_ARGUMENT", `${item.type} parameter value must be a canonical decimal string`);
    if (!["BOOL", "FLOAT64", "INT64", "NUMERIC", "BIGNUMERIC"].includes(item.type) && typeof item.value !== "string") throw new BigQueryError("INVALID_ARGUMENT", `${item.type} parameter value must be a string`);
    seen.add(item.name);
    return { name: item.name, parameterType: { type: item.type }, parameterValue: { value: String(item.value) } };
  });
  if (seen.size !== referenced.size || [...seen].some((name) => !referenced.has(name))) throw new BigQueryError("PARAMETER_MISMATCH", "declared parameters must exactly match named query placeholders");
  return queryParameters;
}

function validateQuerySources(tokens) {
  const upper = tokens.map((token) => token.toUpperCase());
  const ctes = new Set();
  if (upper[0] === "WITH") {
    let cursor = upper[1] === "RECURSIVE" ? 2 : 1;
    while (cursor < tokens.length) {
      const name = tokens[cursor];
      if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name) || upper[cursor + 1] !== "AS" || tokens[cursor + 2] !== "(") break;
      ctes.add(name.toUpperCase());
      cursor += 3;
      let depth = 1;
      while (cursor < tokens.length && depth > 0) { if (tokens[cursor] === "(") depth += 1; if (tokens[cursor] === ")") depth -= 1; cursor += 1; }
      if (depth !== 0 || tokens[cursor] !== ",") break;
      cursor += 1;
    }
  }
  const validSource = (index) => {
    const token = tokens[index];
    const canonical = upper[index];
    return Boolean(token) && (token.startsWith("__BQ_IDENTIFIER_") || token === "(" || canonical === "UNNEST" || ctes.has(canonical));
  };
  const terminators = new Set(["WHERE", "GROUP", "HAVING", "QUALIFY", "WINDOW", "ORDER", "LIMIT", "UNION", "EXCEPT", "INTERSECT"]);
  for (let index = 0; index < tokens.length; index += 1) {
    if (upper[index] !== "FROM" && upper[index] !== "JOIN") continue;
    if (!validSource(index + 1)) throw new BigQueryError("DATASET_NOT_ALLOWED", "every query source must be a qualified allowed table, CTE, subquery, or UNNEST expression");
    if (upper[index] === "JOIN") continue;
    let depth = 0;
    for (let cursor = index + 1; cursor < tokens.length; cursor += 1) {
      if (tokens[cursor] === "(") depth += 1;
      if (tokens[cursor] === ")") { if (depth === 0) break; depth -= 1; }
      if (depth === 0 && terminators.has(upper[cursor])) break;
      if (depth === 0 && tokens[cursor] === "," && !validSource(cursor + 1)) throw new BigQueryError("DATASET_NOT_ALLOWED", "comma-joined sources must be qualified allowed tables, CTEs, subqueries, or UNNEST expressions");
    }
  }
}

export function validateQueryInput(raw, env = process.env) {
  const input = plainObject(raw, "arguments");
  exactKeys(input, ["query", "parameters", "project_id", "billing_project_id", "location", "max_rows", "page_size", "timeout_ms"], "arguments");
  const policy = parseRuntimePolicy(env.BQ_POLICY);
  const projects = policy.allowedProjectIDs;
  const datasets = policy.allowedDatasets;
  const locations = policy.allowedLocations;
  const projectID = String(input.project_id ?? "");
  const billingProjectID = String(input.billing_project_id ?? "");
  const location = String(input.location ?? "");
  if (!projects.includes(projectID)) throw new BigQueryError("PROJECT_NOT_ALLOWED", "requested project is outside the configured allowlist");
  if (billingProjectID !== policy.billingProjectID || !projects.includes(billingProjectID)) throw new BigQueryError("BILLING_PROJECT_NOT_ALLOWED", "billing project does not match the configured authority");
  if (!locations.some((allowed) => allowed.toLowerCase() === location.toLowerCase())) throw new BigQueryError("LOCATION_NOT_ALLOWED", "requested location is outside the configured allowlist");
  const maximumBytesBilled = policy.maxBytesBilled;
  const configuredMaxRows = policy.maxRows;
  const configuredTimeout = policy.maxTimeoutMs;
  const lexed = lexSQL(input.query);
  if (lexed.tokens.includes("?") || lexed.normalized.includes("@@")) throw new BigQueryError("PARAMETER_MISMATCH", "only declared named parameters are allowed");
  const words = lexed.tokens.filter((token) => /^[A-Za-z_]/.test(token) && !token.startsWith("__BQ_IDENTIFIER_")).map((token) => token.toUpperCase());
  if (!words.length || !["SELECT", "WITH"].includes(words[0])) throw new BigQueryError("QUERY_NOT_ALLOWED", "only SELECT and WITH queries are allowed");
  const forbidden = words.find((word) => FORBIDDEN_SQL.has(word) || FORBIDDEN_FUNCTIONS.has(word));
  if (forbidden) throw new BigQueryError("QUERY_NOT_ALLOWED", "query contains a prohibited operation");
  for (let index = 0; index < lexed.tokens.length - 1; index += 1) {
    const token = lexed.tokens[index];
    if (lexed.tokens[index + 1] === "(" && (token.startsWith("__BQ_IDENTIFIER_") || token.includes("."))) {
      throw new BigQueryError("QUERY_NOT_ALLOWED", "qualified routines and table functions are not allowed");
    }
  }
  validateQuerySources(lexed.tokens);
  const allowedDatasetSet = new Set(datasets.map((value) => value.toLowerCase()));
  const tableReferences = lexed.identifiers.map((identifier) => {
    const segments = identifier.split(".");
    if (segments.length !== 2 && segments.length !== 3) throw new BigQueryError("DATASET_NOT_ALLOWED", "table references must be qualified as dataset.table or project.dataset.table");
    const [tableProject, dataset, table] = segments.length === 3 ? segments : [projectID, ...segments];
    if (!/^[a-z][a-z0-9-]{4,61}[a-z0-9]$/.test(tableProject) || !/^[A-Za-z_][A-Za-z0-9_]{0,1023}$/.test(dataset) || !/^[A-Za-z0-9_$*.-]{1,1024}$/.test(table)) throw new BigQueryError("DATASET_NOT_ALLOWED", "table reference is invalid");
    if (!projects.includes(tableProject) || !allowedDatasetSet.has(`${tableProject}.${dataset}`.toLowerCase())) throw new BigQueryError("DATASET_NOT_ALLOWED", "query references a dataset outside the configured allowlist");
    return `${tableProject}.${dataset}.${table}`;
  });
  if (/\b(?:FROM|JOIN)\b/i.test(lexed.normalized) && tableReferences.length === 0 && !/\bUNNEST\s*\(/i.test(lexed.normalized)) throw new BigQueryError("DATASET_NOT_ALLOWED", "external table references must use qualified backtick identifiers");
  const referenced = new Set(lexed.tokens.filter((token) => token.startsWith("@")).map((token) => token.slice(1)));
  const queryParameters = parseParameters(input.parameters, referenced);
  const maxRows = positiveInteger(input.max_rows, Math.min(100, configuredMaxRows), configuredMaxRows, "max_rows");
  const pageSize = positiveInteger(input.page_size, Math.min(100, maxRows), Math.min(MAX_PAGE_SIZE, configuredMaxRows), "page_size");
  return {
    query: input.query, queryParameters, projectID, billingProjectID, location,
    maximumBytesBilled, maxRows, pageSize: Math.min(pageSize, maxRows),
    timeoutMs: positiveInteger(input.timeout_ms, Math.min(30_000, configuredTimeout), configuredTimeout, "timeout_ms"),
    tableReferences,
  };
}

function queryBody(input, dryRun) {
  return {
    query: input.query, useLegacySql: false, parameterMode: "NAMED", queryParameters: input.queryParameters,
    location: input.location, dryRun, maximumBytesBilled: String(input.maximumBytesBilled),
    timeoutMs: Math.min(input.timeoutMs, 10_000), jobTimeoutMs: String(input.timeoutMs),
    maxResults: input.pageSize, requestId: randomUUID(),
  };
}

function executionJobBody(input, jobReference) {
  return {
    jobReference,
    configuration: {
      jobTimeoutMs: String(input.timeoutMs),
      query: {
        query: input.query,
        useLegacySql: false,
        parameterMode: "NAMED",
        queryParameters: input.queryParameters,
        maximumBytesBilled: String(input.maximumBytesBilled),
        useQueryCache: true,
        priority: "INTERACTIVE",
      },
    },
  };
}

function decodeValue(field, cell) {
  const raw = cell?.v;
  if (raw === null || raw === undefined) return null;
  if (field.mode === "REPEATED") {
    if (!Array.isArray(raw)) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery repeated field is invalid");
    return raw.map((entry) => decodeValue({ ...field, mode: "NULLABLE" }, entry));
  }
  if (field.type === "RECORD" || field.type === "STRUCT") {
    if (!raw || !Array.isArray(raw.f) || !Array.isArray(field.fields) || raw.f.length !== field.fields.length) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery record field is invalid");
    return Object.fromEntries(field.fields.map((nested, index) => [nested.name, decodeValue(nested, raw.f[index])]));
  }
  if (field.type === "BOOL" || field.type === "BOOLEAN") {
    if (raw !== "true" && raw !== "false" && typeof raw !== "boolean") throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery boolean field is invalid");
    return raw === true || raw === "true";
  }
  if (["FLOAT", "FLOAT64"].includes(field.type)) { const value = Number(raw); if (!Number.isFinite(value)) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery floating field is invalid"); return value; }
  return String(raw);
}

function normalizePage(payload, expectedSchema) {
  const fields = payload?.schema?.fields ?? expectedSchema;
  if (!Array.isArray(fields) || fields.some((field) => typeof field?.name !== "string" || typeof field?.type !== "string")) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery result schema is invalid");
  if (payload.rows !== undefined && !Array.isArray(payload.rows)) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery result rows are invalid");
  const rows = (payload.rows ?? []).map((row) => {
    if (!Array.isArray(row?.f) || row.f.length !== fields.length) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery row does not match its schema");
    return Object.fromEntries(fields.map((field, index) => [field.name, decodeValue(field, row.f[index])]));
  });
  return { fields, rows };
}

function publicSchemaField(field) {
  return {
    name: field.name,
    type: field.type,
    mode: field.mode ?? "NULLABLE",
    ...(Array.isArray(field.fields) ? { fields: field.fields.map(publicSchemaField) } : {}),
  };
}

async function apiJSON(endpoint, options, context) {
  const response = await boundedFetch(endpoint, options, context);
  const payload = await readBoundedJSON(response);
  if (!response.ok) throw normalizeProviderFailure(response, payload);
  return payload;
}

function providerWarningReasons(...payloads) {
  const reasons = [];
  for (const payload of payloads) {
    for (const warnings of [payload?.errors, payload?.status?.errors]) {
      if (!Array.isArray(warnings)) continue;
      for (const warning of warnings) {
        const reason = warning?.reason;
        if (typeof reason === "string" && /^[A-Za-z][A-Za-z0-9_-]{0,63}$/.test(reason) && !reasons.includes(reason)) reasons.push(reason);
      }
    }
  }
  return reasons;
}

function terminalJobFailure(job) {
  return job?.status?.errorResult && typeof job.status.errorResult === "object";
}

async function cancelJob(input, auth, jobReference, apiBase, fetchImpl) {
  if (!jobReference?.jobId) return false;
  const endpoint = new URL(`/bigquery/v2/projects/${encodeURIComponent(input.billingProjectID)}/jobs/${encodeURIComponent(jobReference.jobId)}/cancel`, apiBase);
  endpoint.searchParams.set("location", input.location);
  try {
    const payload = await apiJSON(endpoint, { method: "POST", headers: { authorization: `Bearer ${auth.token}`, "content-type": "application/json" }, body: "{}" }, { timeoutMs: 5_000, fetchImpl });
    return Boolean(payload.job);
  } catch { return false; }
}

export async function runQuery(raw, { env = process.env, fetchImpl = fetch, now = () => new Date(), resolvedAuth, sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms)), jobIDFactory = randomUUID } = {}) {
  const input = validateQueryInput(raw, env);
  const auth = resolvedAuth ?? await getAccessToken(env, fetchImpl);
  const apiBase = trustedEndpoint(env.BQ_API_BASE_URL, DEFAULT_API_BASE, "BigQuery API endpoint", env);
  const queryEndpoint = new URL(`/bigquery/v2/projects/${encodeURIComponent(input.billingProjectID)}/queries`, apiBase);
  const headers = { authorization: `Bearer ${auth.token}`, "content-type": "application/json" };
  const dryPayload = await apiJSON(queryEndpoint, { method: "POST", headers, body: JSON.stringify(queryBody(input, true)) }, { timeoutMs: Math.min(input.timeoutMs, 15_000), fetchImpl });
  const estimatedBytes = Number(dryPayload.totalBytesProcessed);
  if (!Number.isSafeInteger(estimatedBytes) || estimatedBytes < 0) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery dry run did not return a valid byte estimate");
  if (estimatedBytes > input.maximumBytesBilled) throw new BigQueryError("BYTE_BUDGET_EXCEEDED", "dry-run estimate exceeds the configured byte budget");
  const generatedID = String(jobIDFactory());
  const jobID = `skills_hub_${generatedID}`;
  if (!/^skills_hub_[A-Za-z0-9_-]{1,1000}$/.test(jobID)) throw new BigQueryError("INTERNAL_ERROR", "BigQuery job identity could not be generated");
  const jobReference = { projectId: input.billingProjectID, jobId: jobID, location: input.location };
  const insertEndpoint = new URL(`/bigquery/v2/projects/${encodeURIComponent(input.billingProjectID)}/jobs`, apiBase);
  const started = now().valueOf();
  const deadline = started + input.timeoutMs;
  const queryTimeout = async () => {
    const cancelled = await cancelJob(input, auth, jobReference, apiBase, fetchImpl);
    throw new BigQueryError("QUERY_TIMEOUT", "BigQuery query exceeded the configured timeout", { retryable: true, details: { job_id: jobID, cancellation_requested: cancelled, provider_timeout_ms: input.timeoutMs } });
  };
  const remainingTime = async () => {
    const remaining = deadline - now().valueOf();
    if (remaining <= 0) await queryTimeout();
    return remaining;
  };
  const lifecycleJSON = async (endpoint, options, maximumRequestMs) => {
    const remaining = await remainingTime();
    let result;
    try {
      result = await apiJSON(endpoint, options, { timeoutMs: Math.min(maximumRequestMs, remaining), fetchImpl });
    } catch (error) {
      if (deadline - now().valueOf() <= 0) await queryTimeout();
      throw error;
    }
    await remainingTime();
    return result;
  };
  let submittedJob;
  const submissionTimeout = Math.min(await remainingTime(), 15_000);
  try {
    submittedJob = await apiJSON(insertEndpoint, { method: "POST", headers, body: JSON.stringify(executionJobBody(input, jobReference)) }, { timeoutMs: submissionTimeout, fetchImpl });
  } catch (error) {
    if (error?.status > 0 && error.status < 500) throw error;
    const cancelled = await cancelJob(input, auth, jobReference, apiBase, fetchImpl);
    throw new BigQueryError("QUERY_SUBMISSION_AMBIGUOUS", "BigQuery query submission did not return a durable receipt", { retryable: false, details: { job_id: jobID, cancellation_requested: cancelled, provider_timeout_ms: input.timeoutMs } });
  }
  await remainingTime();
  if (submittedJob?.jobReference?.jobId !== jobID || submittedJob?.jobReference?.projectId !== input.billingProjectID) {
    const cancelled = await cancelJob(input, auth, jobReference, apiBase, fetchImpl);
    throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery returned a mismatched query job identity", { details: { job_id: jobID, cancellation_requested: cancelled } });
  }
  if (terminalJobFailure(submittedJob)) throw new BigQueryError("QUERY_FAILED", "BigQuery reported a terminal query failure", { details: { job_id: jobID } });
  let payload;
  let pages = 1;
  const lifecyclePayloads = [submittedJob];
  while (true) {
    const remaining = await remainingTime();
    const poll = new URL(`/bigquery/v2/projects/${encodeURIComponent(input.billingProjectID)}/queries/${encodeURIComponent(jobReference.jobId)}`, apiBase);
    poll.searchParams.set("location", input.location); poll.searchParams.set("maxResults", String(input.pageSize)); poll.searchParams.set("timeoutMs", String(Math.min(1_000, remaining)));
    try {
      payload = await apiJSON(poll, { method: "GET", headers: { authorization: `Bearer ${auth.token}`, accept: "application/json" } }, { timeoutMs: Math.min(5_000, remaining), fetchImpl });
    } catch (error) {
      if (deadline - now().valueOf() <= 0) await queryTimeout();
      const cancelled = await cancelJob(input, auth, jobReference, apiBase, fetchImpl);
      throw new BigQueryError(error.code ?? "UPSTREAM_ERROR", error.message ?? "BigQuery result polling failed", { retryable: error.retryable, status: error.status, details: { job_id: jobID, cancellation_requested: cancelled, provider_timeout_ms: input.timeoutMs } });
    }
    await remainingTime();
    lifecyclePayloads.push(payload);
    if (payload.jobReference?.jobId && payload.jobReference.jobId !== jobID) {
      const cancelled = await cancelJob(input, auth, jobReference, apiBase, fetchImpl);
      throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery changed the query job identity", { details: { job_id: jobID, cancellation_requested: cancelled } });
    }
    if (payload.jobComplete === true) break;
    await sleep(10);
  }
  const jobEndpoint = new URL(`/bigquery/v2/projects/${encodeURIComponent(input.billingProjectID)}/jobs/${encodeURIComponent(jobID)}`, apiBase);
  jobEndpoint.searchParams.set("location", input.location);
  const finalJob = await lifecycleJSON(jobEndpoint, { method: "GET", headers: { authorization: `Bearer ${auth.token}`, accept: "application/json" } }, 5_000);
  lifecyclePayloads.push(finalJob);
  if (finalJob?.jobReference?.jobId !== jobID || finalJob?.status?.state !== "DONE") throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery returned an invalid terminal job receipt");
  if (terminalJobFailure(finalJob)) throw new BigQueryError("QUERY_FAILED", "BigQuery reported a terminal query failure", { details: { job_id: jobID } });
  const normalized = normalizePage(payload);
  const schema = normalized.fields;
  const rows = [...normalized.rows];
  let normalizedRowBytes = rows.reduce((total, row) => total + Buffer.byteLength(JSON.stringify(row), "utf8") + 1, 0);
  if (normalizedRowBytes > MAX_NORMALIZED_ROW_BYTES) throw new BigQueryError("RESULT_TOO_LARGE", "normalized BigQuery rows exceeded the aggregate MCP response budget");
  let pageToken = payload.pageToken;
  while (pageToken && rows.length < input.maxRows && pages < MAX_PAGES) {
    if (!jobReference?.jobId) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery pagination lacks a job reference");
    const next = new URL(`/bigquery/v2/projects/${encodeURIComponent(input.billingProjectID)}/queries/${encodeURIComponent(jobReference.jobId)}`, apiBase);
    next.searchParams.set("location", input.location); next.searchParams.set("pageToken", pageToken); next.searchParams.set("maxResults", String(Math.min(input.pageSize, input.maxRows - rows.length)));
    const nextPayload = await lifecycleJSON(next, { method: "GET", headers: { authorization: `Bearer ${auth.token}`, accept: "application/json" } }, 10_000);
    lifecyclePayloads.push(nextPayload);
    const page = normalizePage(nextPayload, schema);
    if (JSON.stringify(page.fields) !== JSON.stringify(schema)) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery changed the result schema between pages");
    for (const row of page.rows) {
      normalizedRowBytes += Buffer.byteLength(JSON.stringify(row), "utf8") + 1;
      if (normalizedRowBytes > MAX_NORMALIZED_ROW_BYTES) throw new BigQueryError("RESULT_TOO_LARGE", "normalized BigQuery rows exceeded the aggregate MCP response budget");
      rows.push(row);
    }
    pageToken = nextPayload.pageToken; pages += 1;
  }
  if (pageToken && pages >= MAX_PAGES) throw new BigQueryError("QUERY_BUDGET_EXCEEDED", "query results exceed the allowed page budget");
  const completedAt = now();
  if (deadline - completedAt.valueOf() <= 0) await queryTimeout();
  const totalRows = Number(payload.totalRows ?? rows.length);
  if (!Number.isSafeInteger(totalRows) || totalRows < 0) throw new BigQueryError("UPSTREAM_MALFORMED", "BigQuery total row count is invalid");
  const result = {
    schema_version: "skills-hub.bigquery-result/v1",
    project_id: input.projectID,
    billing_project_id: input.billingProjectID,
    location: input.location,
    schema: schema.map(publicSchemaField),
    rows: rows.slice(0, input.maxRows),
    pagination: { returned_rows: Math.min(rows.length, input.maxRows), total_rows: totalRows, truncated: Boolean(pageToken) || totalRows > input.maxRows, pages },
    diagnostics: {
      job_id: jobReference?.jobId ?? null,
      dry_run_bytes_processed: String(estimatedBytes),
      bytes_processed: String(finalJob?.statistics?.query?.totalBytesProcessed ?? payload.totalBytesProcessed ?? estimatedBytes),
      bytes_billed: String(finalJob?.statistics?.query?.totalBytesBilled ?? "0"),
      slot_millis: String(finalJob?.statistics?.query?.totalSlotMs ?? "0"),
      cache_hit: Boolean(finalJob?.statistics?.query?.cacheHit ?? payload.cacheHit),
      timeout_ms: input.timeoutMs,
      provider_job_timeout_ms: input.timeoutMs,
      maximum_bytes_billed: String(input.maximumBytesBilled),
      table_references: input.tableReferences,
      warning_reasons: providerWarningReasons(...lifecyclePayloads),
    },
    source: { provider: "Google BigQuery", api_version: "v2", retrieved_at: completedAt.toISOString(), freshness: "retrieved-live" },
  };
  return result;
}

export function toolDefinition() {
  return {
    name: "bigquery_run_query",
    description: "Run one bounded, parameterized, read-only BigQuery query within configured project, dataset, location, cost, row, and timeout authority.",
    inputSchema: {
      type: "object", additionalProperties: false,
      required: ["query", "project_id", "billing_project_id", "location"],
      properties: {
        query: { type: "string", minLength: 1, maxLength: MAX_SQL_BYTES },
        parameters: { type: "array", maxItems: MAX_PARAMETERS, items: { type: "object", additionalProperties: false, required: ["name", "type", "value"], properties: { name: { type: "string" }, type: { enum: [...PARAMETER_TYPES] }, value: { type: ["string", "number", "boolean"] } } } },
        project_id: { type: "string" }, billing_project_id: { type: "string" }, location: { type: "string" },
        max_rows: { type: "integer", minimum: 1, maximum: MAX_ROWS }, page_size: { type: "integer", minimum: 1, maximum: MAX_PAGE_SIZE }, timeout_ms: { type: "integer", minimum: 1, maximum: MAX_TIMEOUT_MS },
      },
    },
  };
}

export function publicError(error) {
  const normalized = error instanceof BigQueryError ? error : new BigQueryError("INTERNAL_ERROR", "BigQuery connector failed");
  return { code: normalized.code, message: normalized.message, retryable: normalized.retryable, ...(normalized.status ? { upstream_status: normalized.status } : {}), ...(normalized.details ? { details: normalized.details } : {}) };
}

function safeRequestID(value) { return Number.isSafeInteger(value) || (typeof value === "string" && Buffer.byteLength(value, "utf8") <= 256); }
function assertWireBudget(id, result) { if (Buffer.byteLength(`${JSON.stringify({ jsonrpc: "2.0", id, result })}\n`, "utf8") > MAX_MCP_BYTES) throw new BigQueryError("RESULT_TOO_LARGE", "MCP response exceeded the aggregate response size limit"); }

export async function handleMCPRequest(message, options = {}) {
  if (!message || message.jsonrpc !== "2.0" || !Object.hasOwn(message, "id") || !safeRequestID(message.id) || typeof message.method !== "string") throw new BigQueryError("INVALID_REQUEST", "request must be JSON-RPC 2.0 with an id and method");
  if (message.method === "initialize") { const supported = new Set(["2024-11-05", "2025-03-26", "2025-06-18"]); const requested = message.params?.protocolVersion; return { protocolVersion: supported.has(requested) ? requested : "2025-06-18", capabilities: { tools: { listChanged: false } }, serverInfo: { name: "bigquery-mcp-query-runner", version: "0.2.0" } }; }
  if (message.method === "ping") return {};
  if (message.method === "tools/list") return { tools: [toolDefinition()] };
  if (message.method === "tools/call") {
    if (message.params?.name !== "bigquery_run_query") throw new BigQueryError("METHOD_NOT_FOUND", "unknown tool name");
    try { const query = await runQuery(message.params?.arguments ?? {}, options); const result = { content: [{ type: "text", text: JSON.stringify(query) }], structuredContent: query, isError: false }; assertWireBudget(message.id, result); return result; }
    catch (error) { const failure = publicError(error); return { content: [{ type: "text", text: JSON.stringify(failure) }], structuredContent: { error: failure }, isError: true }; }
  }
  throw new BigQueryError("METHOD_NOT_FOUND", "unsupported MCP method");
}

async function* boundedRequestLines(input) {
  let parts = [];
  let size = 0;
  let oversized = false;
  for await (const chunk of input) {
    const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    let offset = 0;
    for (let index = 0; index < bytes.length; index += 1) {
      if (bytes[index] !== 0x0a) continue;
      const segment = bytes.subarray(offset, index);
      if (oversized || size + segment.length > MAX_MCP_REQUEST_BYTES) {
        yield { oversized: true };
      } else {
        parts.push(segment);
        yield { line: Buffer.concat(parts, size + segment.length).toString("utf8") };
      }
      parts = [];
      size = 0;
      oversized = false;
      offset = index + 1;
    }
    const tail = bytes.subarray(offset);
    if (oversized) continue;
    if (size + tail.length > MAX_MCP_REQUEST_BYTES) {
      parts = [];
      size = 0;
      oversized = true;
    } else if (tail.length > 0) {
      parts.push(tail);
      size += tail.length;
    }
  }
  if (oversized) yield { oversized: true };
  else if (size > 0) yield { line: Buffer.concat(parts, size).toString("utf8") };
}

export async function serveMCP(input = process.stdin, output = process.stdout) {
  for await (const request of boundedRequestLines(input)) {
    if (request.oversized) {
      const failure = publicError(new BigQueryError("REQUEST_TOO_LARGE", "MCP request exceeded the aggregate input size limit"));
      output.write(`${JSON.stringify({ jsonrpc: "2.0", id: null, error: { code: -32600, message: failure.message, data: failure } })}\n`);
      continue;
    }
    const line = request.line;
    if (!line.trim()) continue;
    let message;
    try { message = JSON.parse(line); if (!Object.hasOwn(message, "id") && typeof message.method === "string" && message.method.startsWith("notifications/")) continue; const result = await handleMCPRequest(message); output.write(`${JSON.stringify({ jsonrpc: "2.0", id: message.id, result })}\n`); }
    catch (error) { const failure = publicError(error); output.write(`${JSON.stringify({ jsonrpc: "2.0", id: safeRequestID(message?.id) ? message.id : null, error: { code: -32600, message: failure.message, data: failure } })}\n`); }
  }
}

async function main() {
  if (process.argv.includes("--healthcheck")) { process.stdout.write(`${JSON.stringify({ status: "ok", server: "bigquery-mcp-query-runner", version: "0.2.0", transport: "stdio" })}\n`); return; }
  if (process.argv.includes("--smoke")) {
    const policy = parseRuntimePolicy(process.env.BQ_POLICY);
    const result = await runQuery({ query: "SELECT @probe AS probe", parameters: [{ name: "probe", type: "STRING", value: "ok" }], project_id: policy.allowedProjectIDs[0], billing_project_id: policy.billingProjectID, location: policy.allowedLocations[0], max_rows: 1 });
    if (result.diagnostics.dry_run_bytes_processed !== "0" || result.diagnostics.bytes_processed !== "0" || result.diagnostics.bytes_billed !== "0") throw new BigQueryError("SMOKE_COST_NONZERO", "BigQuery readiness probe did not preserve the zero-byte cost boundary");
    process.stdout.write(`${JSON.stringify({ status: "ok", tier: policy.environmentTier, billing_project_id: policy.billingProjectID, location: policy.allowedLocations[0], job_id: result.diagnostics.job_id, dry_run_bytes_processed: result.diagnostics.dry_run_bytes_processed, bytes_processed: result.diagnostics.bytes_processed, bytes_billed: result.diagnostics.bytes_billed, retrieved_at: result.source.retrieved_at })}\n`); return;
  }
  await serveMCP();
}

function isMainModule() {
  if (!process.argv[1]) return false;
  try { return realpathSync(fileURLToPath(import.meta.url)) === realpathSync(resolve(process.argv[1])); } catch { return false; }
}

if (isMainModule()) main().catch((error) => { process.stderr.write(`${JSON.stringify(publicError(error))}\n`); process.exitCode = 1; });
