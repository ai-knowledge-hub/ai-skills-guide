#!/usr/bin/env node

import { createSign, createHash } from "node:crypto";
import { createInterface } from "node:readline";
import { pathToFileURL } from "node:url";

const DATA_SCOPE = "https://www.googleapis.com/auth/analytics.readonly";
const DEFAULT_API_BASE = "https://analyticsdata.googleapis.com";
const DEFAULT_TOKEN_URL = "https://oauth2.googleapis.com/token";
const DEFAULT_TOKENINFO_URL = "https://oauth2.googleapis.com/tokeninfo";
const MAX_PAGE_SIZE = 250;
const MAX_ROWS = 1_000;
const MAX_PAGES = 10;
const MAX_RESPONSE_BYTES = 2 * 1024 * 1024;
const RESPONSE_ENVELOPE_RESERVE_BYTES = 64 * 1024;
const MAX_DATE_DAYS = 366;

export const ALLOWED_DIMENSIONS = Object.freeze([
  "city", "country", "date", "deviceCategory", "eventName", "firstUserDefaultChannelGroup",
  "firstUserSourceMedium", "landingPagePlusQueryString", "newVsReturning", "pagePath",
  "sessionDefaultChannelGroup", "sessionSourceMedium",
]);

export const ALLOWED_METRICS = Object.freeze([
  "activeUsers", "advertiserAdCost", "conversions", "eventCount", "keyEvents", "newUsers",
  "purchaseRevenue", "screenPageViews", "sessions", "totalRevenue", "totalUsers",
]);

export class GA4Error extends Error {
  constructor(code, message, { retryable = false, status = 0, details = undefined } = {}) {
    super(message);
    this.name = "GA4Error";
    this.code = code;
    this.retryable = retryable;
    this.status = status;
    this.details = details;
  }
}

function assertPlainObject(value, label) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new GA4Error("INVALID_ARGUMENT", `${label} must be an object`);
  }
  return value;
}

function exactKeys(value, allowed, label) {
  for (const key of Object.keys(value)) {
    if (!allowed.includes(key)) {
      throw new GA4Error("INVALID_ARGUMENT", `${label} contains unsupported field ${key}`);
    }
  }
}

function parsePositiveInteger(value, fallback, maximum, label) {
  if (value === undefined) return fallback;
  if (!Number.isSafeInteger(value) || value < 1 || value > maximum) {
    throw new GA4Error("INVALID_ARGUMENT", `${label} must be an integer from 1 to ${maximum}`);
  }
  return value;
}

function parseDate(value, label) {
  if (typeof value !== "string" || !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    throw new GA4Error("INVALID_ARGUMENT", `${label} must use YYYY-MM-DD`);
  }
  const date = new Date(`${value}T00:00:00.000Z`);
  if (Number.isNaN(date.valueOf()) || date.toISOString().slice(0, 10) !== value) {
    throw new GA4Error("INVALID_ARGUMENT", `${label} is not a calendar date`);
  }
  return date;
}

function parseStringList(value, allowed, label, { required = true, maximum = 10 } = {}) {
  if (!Array.isArray(value) || (required && value.length === 0) || value.length > maximum) {
    throw new GA4Error("INVALID_ARGUMENT", `${label} must contain ${required ? "1" : "0"} to ${maximum} values`);
  }
  const seen = new Set();
  for (const item of value) {
    if (typeof item !== "string" || !allowed.includes(item) || seen.has(item)) {
      throw new GA4Error("INVALID_ARGUMENT", `${label} contains an unsupported or duplicate value`);
    }
    seen.add(item);
  }
  return [...value];
}

export function validateRunReportInput(raw, env = process.env, now = () => new Date()) {
  const input = assertPlainObject(raw, "arguments");
  exactKeys(input, ["property_id", "start_date", "end_date", "dimensions", "metrics", "page_size", "max_rows"], "arguments");
  const propertyID = String(input.property_id ?? "").trim();
  if (!/^\d{4,20}$/.test(propertyID)) {
    throw new GA4Error("INVALID_ARGUMENT", "property_id must be a numeric GA4 property identifier");
  }
  const allowedProperties = String(env.GA4_ALLOWED_PROPERTY_IDS ?? "")
    .split(",").map((value) => value.trim()).filter(Boolean);
  if (allowedProperties.length === 0 || !allowedProperties.every((value) => /^\d{4,20}$/.test(value))) {
    throw new GA4Error("CONFIGURATION_ERROR", "GA4_ALLOWED_PROPERTY_IDS must declare numeric property identifiers");
  }
  if (!allowedProperties.includes(propertyID)) {
    throw new GA4Error("PROPERTY_NOT_ALLOWED", "the requested property is outside the configured allowlist");
  }
  const start = parseDate(input.start_date, "start_date");
  const end = parseDate(input.end_date, "end_date");
  const days = Math.floor((end - start) / 86_400_000) + 1;
  if (days < 1 || days > MAX_DATE_DAYS) {
    throw new GA4Error("DATE_RANGE_NOT_ALLOWED", `date range must span 1 to ${MAX_DATE_DAYS} days`);
  }
  const today = now();
  today.setUTCHours(0, 0, 0, 0);
  if (end > today) {
    throw new GA4Error("DATE_RANGE_NOT_ALLOWED", "end_date cannot be in the future");
  }
  return {
    propertyID,
    startDate: input.start_date,
    endDate: input.end_date,
    dimensions: parseStringList(input.dimensions ?? [], ALLOWED_DIMENSIONS, "dimensions", { required: false }),
    metrics: parseStringList(input.metrics, ALLOWED_METRICS, "metrics"),
    pageSize: parsePositiveInteger(input.page_size, 100, MAX_PAGE_SIZE, "page_size"),
    maxRows: parsePositiveInteger(input.max_rows, 500, MAX_ROWS, "max_rows"),
  };
}

function base64url(value) {
  return Buffer.from(value).toString("base64url");
}

export function credentialGeneration(value) {
  return createHash("sha256").update(value).digest("hex").slice(0, 24);
}

export function parseCredentialEnvelope(raw) {
  if (typeof raw !== "string" || raw.length < 1 || raw.length > 64 * 1024) {
    throw new GA4Error("AUTH_MISSING", "GA4_CREDENTIAL is unavailable or exceeds the safe size limit");
  }
  let credential;
  try {
    credential = JSON.parse(raw);
  } catch {
    throw new GA4Error("AUTH_INVALID", "GA4_CREDENTIAL must be a JSON credential envelope");
  }
  assertPlainObject(credential, "GA4_CREDENTIAL");
  if (credential.type === "access_token") {
    exactKeys(credential, ["type", "access_token", "expires_at", "account"], "access-token credential");
    if (typeof credential.access_token !== "string" || credential.access_token.length < 12) {
      throw new GA4Error("AUTH_INVALID", "access-token credential is incomplete");
    }
  } else if (credential.type === "authorized_user") {
    exactKeys(credential, ["type", "client_id", "client_secret", "refresh_token", "token_uri", "account"], "authorized-user credential");
    for (const field of ["client_id", "client_secret", "refresh_token"]) {
      if (typeof credential[field] !== "string" || credential[field].length < 8) {
        throw new GA4Error("AUTH_INVALID", "authorized-user credential is incomplete");
      }
    }
  } else if (credential.type === "service_account") {
    exactKeys(credential, ["type", "client_email", "private_key", "private_key_id", "token_uri", "project_id", "account"], "service-account credential");
    if (typeof credential.client_email !== "string" || !credential.client_email.includes("@") ||
        typeof credential.private_key !== "string" || !credential.private_key.includes("BEGIN PRIVATE KEY")) {
      throw new GA4Error("AUTH_INVALID", "service-account credential is incomplete");
    }
  } else {
    throw new GA4Error("AUTH_INVALID", "credential type must be access_token, authorized_user, or service_account");
  }
  return credential;
}

function trustedEndpoint(raw, fallback, label, env) {
  let endpoint;
  try {
    endpoint = new URL(raw || fallback);
  } catch {
    throw new GA4Error("CONFIGURATION_ERROR", `${label} is invalid`);
  }
  if (endpoint.username || endpoint.password || endpoint.hash) {
    throw new GA4Error("CONFIGURATION_ERROR", `${label} must not include user information or a fragment`);
  }
  const canonical = new URL(fallback);
  const loopback = endpoint.hostname === "localhost" || endpoint.hostname === "127.0.0.1" || endpoint.hostname === "::1";
  const override = Boolean(raw);
  const allowedLoopback = override && env.GA4_ALLOW_LOOPBACK_TEST_ENDPOINTS === "1" && loopback && endpoint.protocol === "http:";
  const canonicalEndpoint = endpoint.origin === canonical.origin && endpoint.pathname === canonical.pathname && endpoint.search === "";
  if (!allowedLoopback && (!canonicalEndpoint || endpoint.protocol !== "https:")) {
    throw new GA4Error("CONFIGURATION_ERROR", `${label} must use the canonical Google endpoint`);
  }
  return endpoint;
}

async function boundedFetch(url, options, { timeoutMs, fetchImpl }) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetchImpl(url, { ...options, signal: controller.signal });
  } catch (error) {
    if (error?.name === "AbortError") {
      throw new GA4Error("UPSTREAM_TIMEOUT", "GA4 request exceeded the configured timeout", { retryable: true });
    }
    throw new GA4Error("UPSTREAM_UNAVAILABLE", "GA4 request could not be completed", { retryable: true });
  } finally {
    clearTimeout(timer);
  }
}

async function readBoundedJSON(response) {
  const declared = Number(response.headers.get("content-length") ?? 0);
  if (Number.isFinite(declared) && declared > MAX_RESPONSE_BYTES) {
    throw new GA4Error("UPSTREAM_RESPONSE_TOO_LARGE", "GA4 response exceeded the size limit");
  }
  const chunks = [];
  let size = 0;
  if (response.body) {
    const reader = response.body.getReader();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > MAX_RESPONSE_BYTES) {
        await reader.cancel();
        throw new GA4Error("UPSTREAM_RESPONSE_TOO_LARGE", "GA4 response exceeded the size limit");
      }
      chunks.push(Buffer.from(value));
    }
  }
  const bytes = Buffer.concat(chunks, size);
  try {
    return JSON.parse(bytes.toString("utf8"));
  } catch {
    throw new GA4Error("UPSTREAM_MALFORMED", "GA4 returned malformed JSON");
  }
}

function normalizeProviderFailure(response, payload) {
  const providerStatus = typeof payload?.error?.status === "string" ? payload.error.status : "";
  if (response.status === 401 || providerStatus === "UNAUTHENTICATED") {
    return new GA4Error("AUTH_REVOKED", "GA4 rejected the configured credential", { status: response.status });
  }
  if (response.status === 403 || providerStatus === "PERMISSION_DENIED") {
    return new GA4Error("PROPERTY_ACCESS_DENIED", "credential cannot access the requested GA4 property", { status: response.status });
  }
  if (response.status === 429 || providerStatus === "RESOURCE_EXHAUSTED") {
    return new GA4Error("QUOTA_EXHAUSTED", "GA4 quota is exhausted", { status: response.status, retryable: true });
  }
  return new GA4Error("UPSTREAM_ERROR", "GA4 rejected the report request", { status: response.status, retryable: response.status >= 500 });
}

export async function getAccessToken(env = process.env, fetchImpl = fetch) {
  const raw = env.GA4_CREDENTIAL;
  const credential = parseCredentialEnvelope(raw);
  if (credential.type === "access_token") {
    if (credential.expires_at && new Date(credential.expires_at) <= new Date()) {
      throw new GA4Error("AUTH_EXPIRED", "configured access token is expired");
    }
    return { token: credential.access_token, credential, generation: credentialGeneration(raw) };
  }
  const tokenEndpoint = trustedEndpoint(env.GA4_TOKEN_URL || credential.token_uri, DEFAULT_TOKEN_URL, "token endpoint", env);
  const body = new URLSearchParams();
  if (credential.type === "authorized_user") {
    body.set("grant_type", "refresh_token");
    body.set("client_id", credential.client_id);
    body.set("client_secret", credential.client_secret);
    body.set("refresh_token", credential.refresh_token);
  } else {
    const now = Math.floor(Date.now() / 1000);
    const header = { alg: "RS256", typ: "JWT" };
    if (credential.private_key_id) header.kid = credential.private_key_id;
    const claim = { iss: credential.client_email, scope: DATA_SCOPE, aud: tokenEndpoint.toString(), iat: now, exp: now + 3600 };
    const unsigned = `${base64url(JSON.stringify(header))}.${base64url(JSON.stringify(claim))}`;
    let signature;
    try {
      const signer = createSign("RSA-SHA256");
      signer.update(unsigned);
      signer.end();
      signature = signer.sign(credential.private_key).toString("base64url");
    } catch {
      throw new GA4Error("AUTH_INVALID", "service-account private key is invalid");
    }
    body.set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer");
    body.set("assertion", `${unsigned}.${signature}`);
  }
  const response = await boundedFetch(tokenEndpoint, {
    method: "POST", headers: { "content-type": "application/x-www-form-urlencoded" }, body,
  }, { timeoutMs: 10_000, fetchImpl });
  const payload = await readBoundedJSON(response);
  if (!response.ok || typeof payload.access_token !== "string" || payload.access_token.length < 12) {
    if (response.status === 400 || response.status === 401) {
      throw new GA4Error("AUTH_REVOKED", "Google rejected the configured credential", { status: response.status });
    }
    throw new GA4Error("AUTH_UNAVAILABLE", "Google token exchange did not complete", { status: response.status, retryable: response.status >= 500 });
  }
  return { token: payload.access_token, credential, generation: credentialGeneration(raw), expiresIn: payload.expires_in };
}

function safeIdentityValue(value) {
  return typeof value === "string" && value.length >= 1 && value.length <= 256 && !/[\r\n\0]/.test(value);
}

function safeProviderIdentity(payload, credentialType) {
  const candidate = payload.email ?? payload.sub ?? payload.user_id;
  if (safeIdentityValue(candidate)) {
    return payload.email ? candidate : `google-sub:${candidate}`;
  }
  if (credentialType === "service_account") {
    const authorizedParty = payload.azp;
    const audience = payload.aud;
    if (authorizedParty !== undefined && !safeIdentityValue(authorizedParty)) {
      throw new GA4Error("AUTH_IDENTITY_UNVERIFIED", "Google did not attest a usable service-account principal");
    }
    if (audience !== undefined && !safeIdentityValue(audience)) {
      throw new GA4Error("AUTH_IDENTITY_UNVERIFIED", "Google did not attest a usable service-account principal");
    }
    if (authorizedParty && audience && authorizedParty !== audience) {
      throw new GA4Error("AUTH_IDENTITY_UNVERIFIED", "Google returned ambiguous service-account identity claims");
    }
    const serviceAccountID = authorizedParty ?? audience;
    if (serviceAccountID) return `google-service-account:${serviceAccountID}`;
  }
  throw new GA4Error("AUTH_IDENTITY_UNVERIFIED", "Google did not attest a usable credential principal");
}

export async function introspectAccessToken(token, credentialType, env = process.env, fetchImpl = fetch) {
  if (typeof token !== "string" || token.length < 12) {
    throw new GA4Error("AUTH_INVALID", "access token is unavailable");
  }
  if (!["access_token", "authorized_user", "service_account"].includes(credentialType)) {
    throw new GA4Error("AUTH_INVALID", "credential type is unavailable for token introspection");
  }
  const endpoint = trustedEndpoint(env.GA4_TOKENINFO_URL, DEFAULT_TOKENINFO_URL, "token-info endpoint", env);
  endpoint.searchParams.set("access_token", token);
  const response = await boundedFetch(endpoint, { method: "GET", headers: { accept: "application/json" } }, {
    timeoutMs: 10_000,
    fetchImpl,
  });
  const payload = await readBoundedJSON(response);
  if (!response.ok) {
    if (response.status === 400 || response.status === 401) {
      throw new GA4Error("AUTH_REVOKED", "Google rejected the configured credential", { status: response.status });
    }
    throw new GA4Error("AUTH_UNAVAILABLE", "Google token introspection did not complete", {
      status: response.status,
      retryable: response.status >= 500,
    });
  }
  if (typeof payload.scope !== "string") {
    throw new GA4Error("AUTH_SCOPE_UNVERIFIED", "Google did not attest credential scopes");
  }
  const scopes = [...new Set(payload.scope.split(/\s+/).filter(Boolean))];
  if (!scopes.includes(DATA_SCOPE)) {
    throw new GA4Error("AUTH_SCOPE_INVALID", "Google did not attest the required GA4 read-only scope");
  }
  const expiresIn = Number(payload.expires_in);
  if (!Number.isFinite(expiresIn) || expiresIn <= 0) {
    throw new GA4Error("AUTH_EXPIRED", "Google reports that the configured credential is expired");
  }
  return { principal: safeProviderIdentity(payload, credentialType), scopes, expiresIn };
}

function normalizeRows(payload, dimensions, metrics) {
  if (payload.rows !== undefined && !Array.isArray(payload.rows)) {
    throw new GA4Error("UPSTREAM_MALFORMED", "GA4 response rows are invalid");
  }
  return (payload.rows ?? []).map((row) => {
    if (!Array.isArray(row?.dimensionValues) || !Array.isArray(row?.metricValues) ||
        row.dimensionValues.length !== dimensions.length || row.metricValues.length !== metrics.length) {
      throw new GA4Error("UPSTREAM_MALFORMED", "GA4 response row shape does not match the request");
    }
    const normalized = {};
    dimensions.forEach((name, index) => { normalized[name] = String(row.dimensionValues[index]?.value ?? ""); });
    metrics.forEach((name, index) => { normalized[name] = String(row.metricValues[index]?.value ?? ""); });
    return normalized;
  });
}

export async function runReport(raw, { env = process.env, fetchImpl = fetch, now = () => new Date(), resolvedAuth } = {}) {
  const input = validateRunReportInput(raw, env, now);
  const auth = resolvedAuth ?? await getAccessToken(env, fetchImpl);
  const apiBase = trustedEndpoint(env.GA4_API_BASE_URL, DEFAULT_API_BASE, "GA4 API endpoint", env);
  const timeoutMs = parsePositiveInteger(env.GA4_REQUEST_TIMEOUT_MS ? Number(env.GA4_REQUEST_TIMEOUT_MS) : undefined, 15_000, 30_000, "GA4_REQUEST_TIMEOUT_MS");
  const rows = [];
  let offset = 0;
  let rowCount = 0;
  let requests = 0;
  let normalizedRowBytes = 0;
  while (rows.length < input.maxRows && requests < MAX_PAGES) {
    const limit = Math.min(input.pageSize, input.maxRows - rows.length);
    const endpoint = new URL(`/v1beta/properties/${input.propertyID}:runReport`, apiBase);
    const headers = { authorization: `Bearer ${auth.token}`, "content-type": "application/json" };
    if (env.GA4_QUOTA_PROJECT) headers["x-goog-user-project"] = env.GA4_QUOTA_PROJECT;
    const response = await boundedFetch(endpoint, {
      method: "POST",
      headers,
      body: JSON.stringify({
        dateRanges: [{ startDate: input.startDate, endDate: input.endDate }],
        dimensions: input.dimensions.map((name) => ({ name })),
        metrics: input.metrics.map((name) => ({ name })),
        limit: String(limit), offset: String(offset), keepEmptyRows: false, returnPropertyQuota: true,
      }),
    }, { timeoutMs, fetchImpl });
    const payload = await readBoundedJSON(response);
    if (!response.ok) throw normalizeProviderFailure(response, payload);
    if (!Number.isSafeInteger(payload.rowCount) || payload.rowCount < 0) {
      throw new GA4Error("UPSTREAM_MALFORMED", "GA4 response rowCount is invalid");
    }
    rowCount = payload.rowCount;
    const pageRows = normalizeRows(payload, input.dimensions, input.metrics);
    if (pageRows.length > limit) {
      throw new GA4Error("UPSTREAM_MALFORMED", "GA4 response exceeded the requested page size");
    }
    for (const row of pageRows) {
      const rowBytes = Buffer.byteLength(JSON.stringify(row), "utf8") + (rows.length === 0 ? 0 : 1);
      if (normalizedRowBytes + rowBytes > MAX_RESPONSE_BYTES - RESPONSE_ENVELOPE_RESERVE_BYTES) {
        throw new GA4Error("RESULT_TOO_LARGE", "normalized GA4 report exceeded the aggregate response size limit");
      }
      normalizedRowBytes += rowBytes;
      rows.push(row);
    }
    requests += 1;
    offset += pageRows.length;
    if (pageRows.length === 0 || offset >= rowCount) break;
  }
  if (offset < rowCount && requests >= MAX_PAGES) {
    throw new GA4Error("QUERY_BUDGET_EXCEEDED", "report requires more than the allowed page budget");
  }
  if (offset < Math.min(rowCount, input.maxRows)) {
    throw new GA4Error("UPSTREAM_MALFORMED", "GA4 response ended before its declared row count");
  }
  const observedAt = now().toISOString();
  const result = {
    schema_version: "skills-hub.ga4-report/v1",
    property_id: input.propertyID,
    date_range: { start_date: input.startDate, end_date: input.endDate },
    dimensions: input.dimensions,
    metrics: input.metrics,
    rows: rows.slice(0, input.maxRows),
    pagination: { returned_rows: Math.min(rows.length, input.maxRows), total_rows: rowCount, truncated: rowCount > input.maxRows, requests },
    source: { provider: "Google Analytics Data API", api_version: "v1beta", retrieved_at: observedAt, freshness: "retrieved-live" },
  };
  if (Buffer.byteLength(JSON.stringify(result), "utf8") > MAX_RESPONSE_BYTES) {
    throw new GA4Error("RESULT_TOO_LARGE", "normalized GA4 report exceeded the aggregate response size limit");
  }
  return result;
}

export function toolDefinition() {
  return {
    name: "ga4_run_report",
    description: "Run a bounded, read-only GA4 report for an explicitly allowed property.",
    inputSchema: {
      type: "object", additionalProperties: false,
      required: ["property_id", "start_date", "end_date", "metrics"],
      properties: {
        property_id: { type: "string", pattern: "^[0-9]{4,20}$" },
        start_date: { type: "string", format: "date" }, end_date: { type: "string", format: "date" },
        dimensions: { type: "array", maxItems: 10, uniqueItems: true, items: { enum: ALLOWED_DIMENSIONS } },
        metrics: { type: "array", minItems: 1, maxItems: 10, uniqueItems: true, items: { enum: ALLOWED_METRICS } },
        page_size: { type: "integer", minimum: 1, maximum: MAX_PAGE_SIZE },
        max_rows: { type: "integer", minimum: 1, maximum: MAX_ROWS },
      },
    },
  };
}

export function publicError(error) {
  const normalized = error instanceof GA4Error ? error : new GA4Error("INTERNAL_ERROR", "GA4 connector failed");
  return { code: normalized.code, message: normalized.message, retryable: normalized.retryable, ...(normalized.status ? { upstream_status: normalized.status } : {}) };
}

function assertMCPWireBudget(id, result) {
  const wireResponse = { jsonrpc: "2.0", id, result };
  if (Buffer.byteLength(`${JSON.stringify(wireResponse)}\n`, "utf8") > MAX_RESPONSE_BYTES) {
    throw new GA4Error("RESULT_TOO_LARGE", "MCP response exceeded the aggregate response size limit");
  }
}

function isSafeMCPRequestID(value) {
  return Number.isSafeInteger(value) || (typeof value === "string" && Buffer.byteLength(value, "utf8") <= 256);
}

export async function handleMCPRequest(message, options = {}) {
  if (!message || message.jsonrpc !== "2.0" || !Object.hasOwn(message, "id") || !isSafeMCPRequestID(message.id) || typeof message.method !== "string") {
    throw new GA4Error("INVALID_REQUEST", "request must be JSON-RPC 2.0 with an id and method");
  }
  if (message.method === "initialize") {
    const supported = new Set(["2024-11-05", "2025-03-26", "2025-06-18"]);
    const requested = message.params?.protocolVersion;
    return { protocolVersion: supported.has(requested) ? requested : "2025-06-18", capabilities: { tools: { listChanged: false } }, serverInfo: { name: "ga4-mcp-connector", version: "0.2.0" } };
  }
  if (message.method === "ping") return {};
  if (message.method === "tools/list") return { tools: [toolDefinition()] };
  if (message.method === "tools/call") {
    if (message.params?.name !== "ga4_run_report") throw new GA4Error("METHOD_NOT_FOUND", "unknown tool name");
    try {
      const report = await runReport(message.params?.arguments ?? {}, options);
      const result = { content: [{ type: "text", text: JSON.stringify(report) }], structuredContent: report, isError: false };
      assertMCPWireBudget(message.id, result);
      return result;
    } catch (error) {
      const failure = publicError(error);
      return { content: [{ type: "text", text: JSON.stringify(failure) }], structuredContent: { error: failure }, isError: true };
    }
  }
  throw new GA4Error("METHOD_NOT_FOUND", "unsupported MCP method");
}

async function serve() {
  const lines = createInterface({ input: process.stdin, crlfDelay: Infinity });
  for await (const line of lines) {
    if (!line.trim()) continue;
    let message;
    try {
      message = JSON.parse(line);
      if (!Object.hasOwn(message, "id") && typeof message.method === "string" && message.method.startsWith("notifications/")) {
        continue;
      }
      const result = await handleMCPRequest(message);
      process.stdout.write(`${JSON.stringify({ jsonrpc: "2.0", id: message.id, result })}\n`);
    } catch (error) {
      const failure = publicError(error);
      const responseID = isSafeMCPRequestID(message?.id) ? message.id : null;
      process.stdout.write(`${JSON.stringify({ jsonrpc: "2.0", id: responseID, error: { code: -32600, message: failure.message, data: failure } })}\n`);
    }
  }
}

async function main() {
  if (process.argv.includes("--healthcheck")) {
    process.stdout.write(`${JSON.stringify({ status: "ok", server: "ga4-mcp-connector", version: "0.2.0", transport: "stdio" })}\n`);
    return;
  }
  if (process.argv.includes("--smoke")) {
    const propertyID = String(process.env.GA4_ALLOWED_PROPERTY_IDS ?? "").split(",")[0]?.trim();
    if (!propertyID) throw new GA4Error("CONFIGURATION_ERROR", "smoke requires GA4_ALLOWED_PROPERTY_IDS");
    const end = new Date(Date.now() - 86_400_000).toISOString().slice(0, 10);
    const report = await runReport({ property_id: propertyID, start_date: end, end_date: end, metrics: ["sessions"], page_size: 1, max_rows: 1 });
    process.stdout.write(`${JSON.stringify({ status: "ok", property_id: propertyID, retrieved_at: report.source.retrieved_at })}\n`);
    return;
  }
  await serve();
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) {
  main().catch((error) => {
    process.stderr.write(`${JSON.stringify(publicError(error))}\n`);
    process.exitCode = 1;
  });
}
