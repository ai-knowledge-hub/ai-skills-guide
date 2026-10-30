#!/usr/bin/env node

import { createHash, createHmac } from "node:crypto";
import { createInterface } from "node:readline";
import { pathToFileURL } from "node:url";

const DEFAULT_GRAPH_ORIGIN = "https://graph.facebook.com";
const DEFAULT_GRAPH_VERSION = "v24.0";
const REQUIRED_SCOPES = Object.freeze(["ads_read", "business_management"]);
const MAX_UPSTREAM_BYTES = 2 * 1024 * 1024;
const MAX_MCP_BYTES = 2 * 1024 * 1024;
const RESULT_RESERVE_BYTES = 96 * 1024;
const MAX_PAGE_SIZE = 100;
const MAX_ROWS = 1_000;
const MAX_PAGES = 20;
const MAX_DATE_DAYS = 366;

export const ALLOWED_FIELDS = Object.freeze([
  "account_id", "account_name", "campaign_id", "campaign_name", "adset_id", "adset_name",
  "ad_id", "ad_name", "date_start", "date_stop", "impressions", "reach", "clicks",
  "spend", "actions", "action_values", "attribution_setting",
]);

export const ALLOWED_BREAKDOWNS = Object.freeze([
  "age", "country", "device_platform", "gender", "impression_device", "platform_position",
  "publisher_platform", "region",
]);

const LEVELS = Object.freeze(["account", "campaign", "adset", "ad"]);

export class MetaAdsError extends Error {
  constructor(code, message, { retryable = false, status = 0, retryAfterSeconds = 0 } = {}) {
    super(message);
    this.name = "MetaAdsError";
    this.code = code;
    this.retryable = retryable;
    this.status = status;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

function assertPlainObject(value, label) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new MetaAdsError("INVALID_ARGUMENT", `${label} must be an object`);
  }
  return value;
}

function exactKeys(value, allowed, label) {
  for (const key of Object.keys(value)) {
    if (!allowed.includes(key)) throw new MetaAdsError("INVALID_ARGUMENT", `${label} contains an unsupported field`);
  }
}

function parsePositiveInteger(value, fallback, maximum, label) {
  if (value === undefined) return fallback;
  if (!Number.isSafeInteger(value) || value < 1 || value > maximum) {
    throw new MetaAdsError("INVALID_ARGUMENT", `${label} must be an integer from 1 to ${maximum}`);
  }
  return value;
}

function parseDate(value, label) {
  if (typeof value !== "string" || !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    throw new MetaAdsError("INVALID_ARGUMENT", `${label} must use YYYY-MM-DD`);
  }
  const date = new Date(`${value}T00:00:00.000Z`);
  if (Number.isNaN(date.valueOf()) || date.toISOString().slice(0, 10) !== value) {
    throw new MetaAdsError("INVALID_ARGUMENT", `${label} is not a calendar date`);
  }
  return date;
}

function uniqueStringList(value, allowed, label, { required = false, maximum = 20 } = {}) {
  if (!Array.isArray(value) || (required && value.length === 0) || value.length > maximum) {
    throw new MetaAdsError("INVALID_ARGUMENT", `${label} has an invalid number of values`);
  }
  const seen = new Set();
  for (const item of value) {
    if (typeof item !== "string" || !allowed.includes(item) || seen.has(item)) {
      throw new MetaAdsError("INVALID_ARGUMENT", `${label} contains an unsupported or duplicate value`);
    }
    seen.add(item);
  }
  return [...value];
}

function parseIdentifierList(value, pattern, label, maximum = 100) {
  if (!Array.isArray(value) || value.length < 1 || value.length > maximum) {
    throw new MetaAdsError("CONFIGURATION_ERROR", `${label} must contain 1 to ${maximum} identifiers`);
  }
  const normalized = value.map((item) => String(item ?? "").trim());
  if (!normalized.every((item) => pattern.test(item)) || new Set(normalized).size !== normalized.length) {
    throw new MetaAdsError("CONFIGURATION_ERROR", `${label} contains an invalid or duplicate identifier`);
  }
  return normalized;
}

export function parsePolicy(raw) {
  if (typeof raw !== "string" || raw.length < 2 || raw.length > 64 * 1024) {
    throw new MetaAdsError("CONFIGURATION_ERROR", "META_ADS_POLICY is unavailable or exceeds the safe size limit");
  }
  let policy;
  try { policy = JSON.parse(raw); } catch { throw new MetaAdsError("CONFIGURATION_ERROR", "META_ADS_POLICY must be valid JSON"); }
  assertPlainObject(policy, "META_ADS_POLICY");
  exactKeys(policy, ["allowed_ad_account_ids", "allowed_business_ids", "allowed_fields", "max_date_days", "max_rows", "max_pages", "max_timeout_ms", "graph_api_version", "environment_tier"], "META_ADS_POLICY");
  const graphVersion = String(policy.graph_api_version ?? DEFAULT_GRAPH_VERSION).trim();
  if (graphVersion !== DEFAULT_GRAPH_VERSION) throw new MetaAdsError("CONFIGURATION_ERROR", `graph_api_version must be ${DEFAULT_GRAPH_VERSION}`);
  if (!['sandbox', 'live'].includes(policy.environment_tier)) throw new MetaAdsError("CONFIGURATION_ERROR", "environment_tier must be sandbox or live");
  return {
    allowedAccountIDs: parseIdentifierList(policy.allowed_ad_account_ids, /^act_[1-9][0-9]{4,30}$/, "allowed_ad_account_ids", 1),
    allowedBusinessIDs: parseIdentifierList(policy.allowed_business_ids, /^[1-9][0-9]{4,30}$/, "allowed_business_ids", 1),
    allowedFields: uniqueStringList(policy.allowed_fields ?? ALLOWED_FIELDS, ALLOWED_FIELDS, "allowed_fields", { required: true, maximum: ALLOWED_FIELDS.length }),
    maxDateDays: parsePositiveInteger(policy.max_date_days, 90, MAX_DATE_DAYS, "max_date_days"),
    maxRows: parsePositiveInteger(policy.max_rows, 500, MAX_ROWS, "max_rows"),
    maxPages: parsePositiveInteger(policy.max_pages, 10, MAX_PAGES, "max_pages"),
    maxTimeoutMs: parsePositiveInteger(policy.max_timeout_ms, 30_000, 60_000, "max_timeout_ms"),
    graphVersion,
    environmentTier: policy.environment_tier,
  };
}

export function parseCredentialEnvelope(raw) {
  if (typeof raw !== "string" || raw.length < 2 || raw.length > 64 * 1024) {
    throw new MetaAdsError("AUTH_MISSING", "META_ADS_CREDENTIAL is unavailable or exceeds the safe size limit");
  }
  let credential;
  try { credential = JSON.parse(raw); } catch { throw new MetaAdsError("AUTH_INVALID", "META_ADS_CREDENTIAL must be valid JSON"); }
  assertPlainObject(credential, "META_ADS_CREDENTIAL");
  exactKeys(credential, ["type", "access_token", "app_id", "app_secret"], "META_ADS_CREDENTIAL");
  if (credential.type !== "access_token" || typeof credential.access_token !== "string" || credential.access_token.length < 20 ||
      typeof credential.app_id !== "string" || !/^[1-9][0-9]{4,30}$/.test(credential.app_id) ||
      typeof credential.app_secret !== "string" || credential.app_secret.length < 16) {
    throw new MetaAdsError("AUTH_INVALID", "Meta access-token credential is incomplete");
  }
  return credential;
}

export function credentialGeneration(raw) {
  return createHash("sha256").update(raw).digest("hex").slice(0, 24);
}

function graphOrigin(env) {
  const raw = env.META_ADS_GRAPH_ORIGIN;
  let endpoint;
  try { endpoint = new URL(raw || DEFAULT_GRAPH_ORIGIN); } catch { throw new MetaAdsError("CONFIGURATION_ERROR", "Meta Graph endpoint is invalid"); }
  const canonical = endpoint.origin === DEFAULT_GRAPH_ORIGIN && endpoint.pathname === "/" && !endpoint.search && !endpoint.hash;
  const loopback = ["localhost", "127.0.0.1", "::1"].includes(endpoint.hostname);
  const allowedTest = raw && env.META_ADS_ALLOW_LOOPBACK_TEST_ENDPOINTS === "1" && loopback && endpoint.protocol === "http:" && endpoint.pathname === "/" && !endpoint.search && !endpoint.hash;
  if (endpoint.username || endpoint.password || (!canonical && !allowedTest)) throw new MetaAdsError("CONFIGURATION_ERROR", "Meta Graph endpoint is not trusted");
  return endpoint.origin;
}

function graphURL(pathname, env) {
  return new URL(pathname, `${graphOrigin(env)}/`);
}

function remaining(deadline, clock) {
  const value = deadline - clock();
  if (value <= 0) throw new MetaAdsError("UPSTREAM_TIMEOUT", "Meta Ads request exceeded the configured timeout", { retryable: true });
  return value;
}

function abortError() {
  const error = new Error("request aborted");
  error.name = "AbortError";
  return error;
}

async function readChunk(reader, signal) {
  if (signal.aborted) throw abortError();
  return new Promise((resolve, reject) => {
    const onAbort = () => {
      void reader.cancel().catch(() => {});
      reject(abortError());
    };
    signal.addEventListener("abort", onAbort, { once: true });
    reader.read().then(resolve, reject).finally(() => signal.removeEventListener("abort", onAbort));
  });
}

async function boundedFetchJSON(url, options, { deadline, clock, fetchImpl }) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), remaining(deadline, clock));
  try {
    const response = await fetchImpl(url, { ...options, redirect: "error", signal: controller.signal });
    remaining(deadline, clock);
    const payload = await readBoundedJSON(response, { deadline, clock, signal: controller.signal });
    remaining(deadline, clock);
    return { response, payload };
  } catch (error) {
    if (error instanceof MetaAdsError) throw error;
    if (error?.name === "AbortError") throw new MetaAdsError("UPSTREAM_TIMEOUT", "Meta Ads request exceeded the configured timeout", { retryable: true });
    throw new MetaAdsError("UPSTREAM_UNAVAILABLE", "Meta Ads request could not be completed", { retryable: true });
  } finally { clearTimeout(timer); }
}

async function readBoundedJSON(response, { deadline, clock, signal }) {
  const declared = Number(response.headers.get("content-length") ?? 0);
  if (Number.isFinite(declared) && declared > MAX_UPSTREAM_BYTES) throw new MetaAdsError("UPSTREAM_RESPONSE_TOO_LARGE", "Meta returned an oversized response");
  const chunks = [];
  let size = 0;
  if (response.body) {
    const reader = response.body.getReader();
    while (true) {
      remaining(deadline, clock);
      const { done, value } = await readChunk(reader, signal);
      if (done) break;
      remaining(deadline, clock);
      size += value.byteLength;
      if (size > MAX_UPSTREAM_BYTES) {
        await reader.cancel();
        throw new MetaAdsError("UPSTREAM_RESPONSE_TOO_LARGE", "Meta returned an oversized response");
      }
      chunks.push(Buffer.from(value));
    }
  }
  try { return JSON.parse(Buffer.concat(chunks, size).toString("utf8")); }
  catch { throw new MetaAdsError("UPSTREAM_MALFORMED", "Meta returned malformed JSON"); }
}

function retryAfter(response) {
  const value = Number(response.headers.get("retry-after") ?? 0);
  return Number.isFinite(value) && value > 0 && value <= 86_400 ? Math.ceil(value) : 0;
}

function providerError(response, payload, accountContext = false) {
  const code = Number(payload?.error?.code ?? 0);
  const subcode = Number(payload?.error?.error_subcode ?? 0);
  if (response.status === 401 || code === 190) {
    const expired = [463, 464, 467].includes(subcode);
    return new MetaAdsError(expired ? "AUTH_EXPIRED" : "AUTH_REVOKED", expired ? "Meta reports that the configured token is expired" : "Meta rejected the configured credential", { status: response.status });
  }
  if (response.status === 429 || [4, 17, 32, 613].includes(code)) {
    return new MetaAdsError("RATE_LIMITED", "Meta Ads rate limit is active", { status: response.status, retryable: true, retryAfterSeconds: retryAfter(response) });
  }
  if (response.status === 403 || [10, 200, 294].includes(code)) {
    return new MetaAdsError(accountContext ? "ACCOUNT_ACCESS_DENIED" : "AUTH_SCOPE_INVALID", accountContext ? "credential cannot access the selected Meta ad account" : "Meta did not grant the required read permissions", { status: response.status });
  }
  return new MetaAdsError("UPSTREAM_ERROR", "Meta rejected the read request", { status: response.status, retryable: response.status >= 500 });
}

function bearerHeaders(token) {
  return { accept: "application/json", authorization: `Bearer ${token}` };
}

function safeProviderID(value) {
  return typeof value === "string" && /^[1-9][0-9]{4,30}$/.test(value);
}

export async function verifyMetaAuthority({ env = process.env, fetchImpl = fetch, now = () => new Date(), clock = Date.now, policy: suppliedPolicy, deadline: suppliedDeadline } = {}) {
  const credentialRaw = env.META_ADS_CREDENTIAL;
  const credential = parseCredentialEnvelope(credentialRaw);
  const policy = suppliedPolicy ?? parsePolicy(env.META_ADS_POLICY);
  const deadline = suppliedDeadline ?? clock() + policy.maxTimeoutMs;
  const debugURL = graphURL(`/${policy.graphVersion}/debug_token`, env);
  debugURL.searchParams.set("input_token", credential.access_token);
  debugURL.searchParams.set("access_token", `${credential.app_id}|${credential.app_secret}`);
  const { response: debugResponse, payload: debugPayload } = await boundedFetchJSON(debugURL, { method: "GET", headers: { accept: "application/json" } }, { deadline, clock, fetchImpl });
  if (!debugResponse.ok) throw providerError(debugResponse, debugPayload);
  const data = debugPayload?.data;
  if (!data || data.is_valid !== true) throw new MetaAdsError("AUTH_REVOKED", "Meta reports that the configured token is invalid");
  if (String(data.app_id ?? "") !== credential.app_id) throw new MetaAdsError("AUTH_APP_MISMATCH", "Meta token is bound to a different application");
  if (!safeProviderID(String(data.user_id ?? ""))) throw new MetaAdsError("AUTH_IDENTITY_UNVERIFIED", "Meta did not attest a usable token principal");
  const scopes = Array.isArray(data.scopes) ? [...new Set(data.scopes.filter((value) => typeof value === "string"))] : [];
  if (REQUIRED_SCOPES.some((scope) => !scopes.includes(scope))) throw new MetaAdsError("AUTH_SCOPE_INVALID", "Meta did not attest every required read permission");
  const expiryFields = ["expires_at", "data_access_expires_at"].filter((field) => Object.hasOwn(data, field));
  if (expiryFields.length === 0) throw new MetaAdsError("AUTH_IDENTITY_UNVERIFIED", "Meta did not attest token lifetime metadata");
  const declaredExpiries = expiryFields.map((field) => Number(data[field]));
  if (!declaredExpiries.every((value) => Number.isSafeInteger(value) && value >= 0)) throw new MetaAdsError("AUTH_IDENTITY_UNVERIFIED", "Meta returned invalid token lifetime metadata");
  const nowSeconds = Math.floor(now().valueOf() / 1000);
  if (declaredExpiries.some((value) => value > 0 && value <= nowSeconds)) throw new MetaAdsError("AUTH_EXPIRED", "Meta reports that the configured token is expired");
  const finiteExpiries = declaredExpiries.filter((value) => value > 0);
  const providerExpirySeconds = finiteExpiries.length ? Math.min(...finiteExpiries) : null;

  const accountID = policy.allowedAccountIDs[0];
  const appSecretProof = createHmac("sha256", credential.app_secret).update(credential.access_token).digest("hex");
  const accountURL = graphURL(`/${policy.graphVersion}/${accountID}`, env);
  accountURL.searchParams.set("fields", "id,account_id,name,account_status,business{id,name},currency,timezone_name,timezone_offset_hours_utc,attribution_spec");
  accountURL.searchParams.set("appsecret_proof", appSecretProof);
  const { response: accountResponse, payload: account } = await boundedFetchJSON(accountURL, { method: "GET", headers: bearerHeaders(credential.access_token) }, { deadline, clock, fetchImpl });
  if (!accountResponse.ok) throw providerError(accountResponse, account, true);
  const normalizedAccount = String(account.id ?? "").startsWith("act_") ? String(account.id) : `act_${String(account.account_id ?? "")}`;
  if (normalizedAccount !== accountID || !policy.allowedAccountIDs.includes(normalizedAccount)) throw new MetaAdsError("ACCOUNT_MISMATCH", "Meta returned a different ad account");
  if (Number(account.account_status) !== 1) throw new MetaAdsError("ACCOUNT_INACTIVE", "Meta reports that the selected ad account is not active");
  const businessID = String(account.business?.id ?? "");
  if (!safeProviderID(businessID) || !policy.allowedBusinessIDs.includes(businessID)) throw new MetaAdsError("BUSINESS_MISMATCH", "Meta ad account is not owned by an allowed business");
  const completedAtSeconds = Math.floor(now().valueOf() / 1000);
  if (providerExpirySeconds !== null && providerExpirySeconds <= completedAtSeconds) throw new MetaAdsError("AUTH_EXPIRED", "Meta reports that the configured token expired during verification");
  const generation = credentialGeneration(credentialRaw);
  const providerExpiresAt = providerExpirySeconds === null ? null : new Date(providerExpirySeconds * 1000).toISOString();
  const attestationMaterial = JSON.stringify({ app_id: data.app_id, user_id: data.user_id, scopes: [...scopes].sort(), expires_at: providerExpiresAt, account_id: normalizedAccount, business_id: businessID });
  return {
    token: credential.access_token, appSecretProof, policy, account,
    accountID: normalizedAccount, businessID, principal: `meta-user:${data.user_id}`,
    scopes, providerExpiresAt, generation,
    attestationReference: `meta-live:${createHash("sha256").update(attestationMaterial).digest("hex")}`,
  };
}

export function validateInsightsInput(raw, policy, now = () => new Date()) {
  const input = assertPlainObject(raw, "arguments");
  exactKeys(input, ["ad_account_id", "level", "start_date", "end_date", "fields", "breakdowns", "page_size", "max_rows", "timeout_ms"], "arguments");
  const accountID = String(input.ad_account_id ?? "").trim();
  if (!policy.allowedAccountIDs.includes(accountID)) throw new MetaAdsError("ACCOUNT_NOT_ALLOWED", "the requested ad account is outside the configured allowlist");
  if (!LEVELS.includes(input.level)) throw new MetaAdsError("INVALID_ARGUMENT", "level must be account, campaign, adset, or ad");
  const start = parseDate(input.start_date, "start_date");
  const end = parseDate(input.end_date, "end_date");
  const days = Math.floor((end - start) / 86_400_000) + 1;
  if (days < 1 || days > policy.maxDateDays) throw new MetaAdsError("DATE_RANGE_NOT_ALLOWED", `date range must span 1 to ${policy.maxDateDays} days`);
  const today = now();
  today.setUTCHours(0, 0, 0, 0);
  if (end > today) throw new MetaAdsError("DATE_RANGE_NOT_ALLOWED", "end_date cannot be in the future");
  const defaults = ["account_id", "campaign_id", "campaign_name", "date_start", "date_stop", "impressions", "clicks", "spend", "actions", "action_values", "attribution_setting"];
  const requestedFields = uniqueStringList(input.fields ?? defaults, policy.allowedFields, "fields", { required: true, maximum: 20 });
  const fields = [...new Set([...requestedFields, "account_id", "date_start", "date_stop", "attribution_setting"])];
  return {
    accountID, level: input.level, startDate: input.start_date, endDate: input.end_date, fields,
    breakdowns: uniqueStringList(input.breakdowns ?? [], ALLOWED_BREAKDOWNS, "breakdowns", { maximum: 4 }),
    pageSize: parsePositiveInteger(input.page_size, 50, Math.min(MAX_PAGE_SIZE, policy.maxRows), "page_size"),
    maxRows: parsePositiveInteger(input.max_rows, policy.maxRows, policy.maxRows, "max_rows"),
    timeoutMs: parsePositiveInteger(input.timeout_ms, policy.maxTimeoutMs, policy.maxTimeoutMs, "timeout_ms"),
  };
}

function normalizeActionList(value, label) {
  if (value === undefined) return undefined;
  if (!Array.isArray(value) || value.length > 500) throw new MetaAdsError("UPSTREAM_MALFORMED", `Meta ${label} field is invalid`);
  return value.map((item) => {
    if (!item || typeof item !== "object" || typeof item.action_type !== "string" || item.action_type.length > 256 || !["string", "number"].includes(typeof item.value)) {
      throw new MetaAdsError("UPSTREAM_MALFORMED", `Meta ${label} item is invalid`);
    }
    return { action_type: item.action_type, value: String(item.value) };
  });
}

function normalizeRow(row, fields, accountID) {
  if (!row || typeof row !== "object" || Array.isArray(row) || String(row.account_id ?? "") !== accountID.slice(4)) {
    throw new MetaAdsError("UPSTREAM_MALFORMED", "Meta returned a row for an unexpected ad account");
  }
  const normalized = {};
  for (const field of fields) {
    const value = row[field];
    if (field === "actions" || field === "action_values") {
      const actions = normalizeActionList(value, field);
      if (actions !== undefined) normalized[field] = actions;
    } else if (value !== undefined) {
      if (!["string", "number", "boolean"].includes(typeof value)) throw new MetaAdsError("UPSTREAM_MALFORMED", "Meta returned an unsupported row value");
      normalized[field] = String(value);
    }
  }
  return normalized;
}

function safeCursor(payload) {
  const cursor = payload?.paging?.cursors?.after;
  if (cursor === undefined) return "";
  if (typeof cursor !== "string" || cursor.length < 1 || cursor.length > 2_048 || /[\r\n\0]/.test(cursor)) {
    throw new MetaAdsError("UPSTREAM_MALFORMED", "Meta returned an invalid pagination cursor");
  }
  return cursor;
}

export async function runInsights(raw, { env = process.env, fetchImpl = fetch, now = () => new Date(), clock = Date.now, resolvedAuth } = {}) {
  const policy = resolvedAuth?.policy ?? parsePolicy(env.META_ADS_POLICY);
  const input = validateInsightsInput(raw, policy, now);
  const deadline = clock() + input.timeoutMs;
  const auth = resolvedAuth ?? await verifyMetaAuthority({ env, fetchImpl, now, clock, policy, deadline });
  if (input.accountID !== auth.accountID) throw new MetaAdsError("ACCOUNT_MISMATCH", "selected account does not match the verified credential target");
  const rows = [];
  const providerResponseDates = [];
  let after = "";
  let requests = 0;
  let rowBytes = 0;
  while (rows.length < input.maxRows && requests < auth.policy.maxPages) {
    const pageLimit = Math.min(input.pageSize, input.maxRows - rows.length);
    const endpoint = graphURL(`/${auth.policy.graphVersion}/${input.accountID}/insights`, env);
    endpoint.searchParams.set("fields", input.fields.join(","));
    endpoint.searchParams.set("level", input.level);
    endpoint.searchParams.set("time_range", JSON.stringify({ since: input.startDate, until: input.endDate }));
    endpoint.searchParams.set("use_account_attribution_setting", "true");
    endpoint.searchParams.set("appsecret_proof", auth.appSecretProof);
    endpoint.searchParams.set("limit", String(pageLimit));
    if (input.breakdowns.length) endpoint.searchParams.set("breakdowns", input.breakdowns.join(","));
    if (after) endpoint.searchParams.set("after", after);
    const { response, payload } = await boundedFetchJSON(endpoint, { method: "GET", headers: bearerHeaders(auth.token) }, { deadline, clock, fetchImpl });
    if (!response.ok) throw providerError(response, payload, true);
    if (!Array.isArray(payload.data) || payload.data.length > pageLimit) throw new MetaAdsError("UPSTREAM_MALFORMED", "Meta returned an invalid insights page");
    const responseDate = response.headers.get("date");
    if (responseDate && !Number.isNaN(Date.parse(responseDate))) providerResponseDates.push(new Date(responseDate).toISOString());
    for (const item of payload.data) {
      const row = normalizeRow(item, input.fields, input.accountID);
      const bytes = Buffer.byteLength(JSON.stringify(row), "utf8") + (rows.length ? 1 : 0);
      if (rowBytes + bytes > MAX_MCP_BYTES / 2 - RESULT_RESERVE_BYTES) throw new MetaAdsError("RESULT_TOO_LARGE", "normalized Meta Ads report exceeded the response limit");
      rowBytes += bytes;
      rows.push(row);
    }
    requests += 1;
    const next = safeCursor(payload);
    if (!next || payload.data.length === 0) { after = ""; break; }
    if (next === after) throw new MetaAdsError("UPSTREAM_MALFORMED", "Meta repeated a pagination cursor");
    after = next;
  }
  if (after && rows.length < input.maxRows && requests >= auth.policy.maxPages) throw new MetaAdsError("QUERY_BUDGET_EXCEEDED", "report requires more than the allowed page budget");
  const result = {
    schema_version: "skills-hub.meta-ads-insights/v1",
    ad_account_id: input.accountID,
    business_id: auth.businessID,
    level: input.level,
    date_range: { start_date: input.startDate, end_date: input.endDate },
    fields: input.fields,
    breakdowns: input.breakdowns,
    rows,
    pagination: { returned_rows: rows.length, truncated: Boolean(after), requests },
    attribution: { mode: "account-setting", account_spec: auth.account.attribution_spec ?? null },
    source: {
      provider: "Meta Marketing API", api_version: auth.policy.graphVersion,
      retrieved_at: now().toISOString(), provider_response_dates: [...new Set(providerResponseDates)],
      account_timezone: String(auth.account.timezone_name ?? ""), account_currency: String(auth.account.currency ?? ""),
    },
  };
  if (Buffer.byteLength(JSON.stringify(result), "utf8") > MAX_MCP_BYTES / 2) throw new MetaAdsError("RESULT_TOO_LARGE", "normalized Meta Ads report exceeded the response limit");
  return result;
}

export function toolDefinition() {
  return {
    name: "meta_ads_read_insights",
    description: "Read a bounded Meta Ads insights report for a policy-allowed ad account.",
    inputSchema: {
      type: "object", additionalProperties: false,
      required: ["ad_account_id", "level", "start_date", "end_date"],
      properties: {
        ad_account_id: { type: "string", pattern: "^act_[1-9][0-9]{4,30}$" },
        level: { enum: LEVELS }, start_date: { type: "string", format: "date" }, end_date: { type: "string", format: "date" },
        fields: { type: "array", minItems: 1, maxItems: 20, uniqueItems: true, items: { enum: ALLOWED_FIELDS } },
        breakdowns: { type: "array", maxItems: 4, uniqueItems: true, items: { enum: ALLOWED_BREAKDOWNS } },
        page_size: { type: "integer", minimum: 1, maximum: MAX_PAGE_SIZE },
        max_rows: { type: "integer", minimum: 1, maximum: MAX_ROWS },
        timeout_ms: { type: "integer", minimum: 1, maximum: 60_000 },
      },
    },
  };
}

export function publicError(error) {
  const normalized = error instanceof MetaAdsError ? error : new MetaAdsError("INTERNAL_ERROR", "Meta Ads connector failed");
  return {
    code: normalized.code, message: normalized.message, retryable: normalized.retryable,
    ...(normalized.status ? { upstream_status: normalized.status } : {}),
    ...(normalized.retryAfterSeconds ? { retry_after_seconds: normalized.retryAfterSeconds } : {}),
  };
}

function assertMCPWireBudget(id, result) {
  if (Buffer.byteLength(`${JSON.stringify({ jsonrpc: "2.0", id, result })}\n`, "utf8") > MAX_MCP_BYTES) {
    throw new MetaAdsError("RESULT_TOO_LARGE", "MCP response exceeded the aggregate response limit");
  }
}

function safeRequestID(value) {
  return Number.isSafeInteger(value) || (typeof value === "string" && Buffer.byteLength(value, "utf8") <= 256);
}

export async function handleMCPRequest(message, options = {}) {
  if (!message || message.jsonrpc !== "2.0" || !Object.hasOwn(message, "id") || !safeRequestID(message.id) || typeof message.method !== "string") {
    throw new MetaAdsError("INVALID_REQUEST", "request must be JSON-RPC 2.0 with an id and method");
  }
  if (message.method === "initialize") {
    const versions = new Set(["2024-11-05", "2025-03-26", "2025-06-18"]);
    const requested = message.params?.protocolVersion;
    return { protocolVersion: versions.has(requested) ? requested : "2025-06-18", capabilities: { tools: { listChanged: false } }, serverInfo: { name: "meta-ads-mcp-connector", version: "0.2.0" } };
  }
  if (message.method === "ping") return {};
  if (message.method === "tools/list") return { tools: [toolDefinition()] };
  if (message.method === "tools/call") {
    if (message.params?.name !== "meta_ads_read_insights") throw new MetaAdsError("METHOD_NOT_FOUND", "unknown tool name");
    try {
      const report = await runInsights(message.params?.arguments ?? {}, options);
      const result = { content: [{ type: "text", text: JSON.stringify(report) }], structuredContent: report, isError: false };
      assertMCPWireBudget(message.id, result);
      return result;
    } catch (error) {
      const failure = publicError(error);
      return { content: [{ type: "text", text: JSON.stringify(failure) }], structuredContent: { error: failure }, isError: true };
    }
  }
  throw new MetaAdsError("METHOD_NOT_FOUND", "unsupported MCP method");
}

async function serve() {
  const lines = createInterface({ input: process.stdin, crlfDelay: Infinity });
  for await (const line of lines) {
    if (!line.trim()) continue;
    let message;
    try {
      if (Buffer.byteLength(line, "utf8") > 64 * 1024) throw new MetaAdsError("INVALID_REQUEST", "request exceeds the safe size limit");
      message = JSON.parse(line);
      if (!Object.hasOwn(message, "id") && typeof message.method === "string" && message.method.startsWith("notifications/")) continue;
      const result = await handleMCPRequest(message);
      process.stdout.write(`${JSON.stringify({ jsonrpc: "2.0", id: message.id, result })}\n`);
    } catch (error) {
      const failure = publicError(error);
      const id = safeRequestID(message?.id) ? message.id : null;
      process.stdout.write(`${JSON.stringify({ jsonrpc: "2.0", id, error: { code: -32600, message: failure.message, data: failure } })}\n`);
    }
  }
}

async function main() {
  if (process.argv.includes("--healthcheck")) {
    process.stdout.write(`${JSON.stringify({ status: "ok", server: "meta-ads-mcp-connector", version: "0.2.0", transport: "stdio" })}\n`);
    return;
  }
  if (process.argv.includes("--smoke")) {
    const policy = parsePolicy(process.env.META_ADS_POLICY);
    const end = new Date(Date.now() - 86_400_000).toISOString().slice(0, 10);
    const report = await runInsights({ ad_account_id: policy.allowedAccountIDs[0], level: "account", start_date: end, end_date: end, fields: ["account_id", "impressions"], page_size: 1, max_rows: 1 });
    process.stdout.write(`${JSON.stringify({ status: "ok", ad_account_id: report.ad_account_id, retrieved_at: report.source.retrieved_at })}\n`);
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
