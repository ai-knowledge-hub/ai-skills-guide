import { createHash, createHmac, timingSafeEqual } from "node:crypto";
import { constants as fsConstants } from "node:fs";
import { access, lstat, mkdir, open, readFile, readdir, realpath, rename, rmdir, stat, unlink } from "node:fs/promises";
import path from "node:path";
import { spawn } from "node:child_process";

const SHA256 = /^sha256:[a-f0-9]{64}$/;
const RAW_SHA256 = /^[a-f0-9]{64}$/;
const ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{1,127}$/;
const NUMERIC_ID = /^[1-9][0-9]{3,29}$/;
const PLAN_VERSION = "ad-platform.change-plan/v2";
const ROLLBACK_VERSION = "ad-platform.rollback-request/v1";
const MAX_UPSTREAM_BYTES = 2 * 1024 * 1024;
const MAX_MCP_BYTES = 2 * 1024 * 1024;
const DEFAULT_TIMEOUT_MS = 30_000;
const MAX_TIMEOUT_MS = 120_000;

export class ExecutorError extends Error {
  constructor(code, message, { retryable = false, ambiguous = false, status = 0 } = {}) {
    super(message);
    this.name = "ExecutorError";
    this.code = code;
    this.retryable = retryable;
    this.ambiguous = ambiguous;
    this.status = status;
  }
}

function plain(value, label) {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new ExecutorError("INVALID_ARGUMENT", `${label} must be an object`);
  return value;
}

function exact(value, keys, label) {
  for (const key of Object.keys(value)) if (!keys.includes(key)) throw new ExecutorError("INVALID_ARGUMENT", `${label} contains an unsupported field`);
}

function identifier(value, label) {
  if (typeof value !== "string" || !ID.test(value)) throw new ExecutorError("INVALID_ARGUMENT", `${label} is invalid`);
  return value;
}

function numericID(value, label) {
  const normalized = String(value ?? "").replaceAll("-", "");
  if (!NUMERIC_ID.test(normalized)) throw new ExecutorError("INVALID_ARGUMENT", `${label} is invalid`);
  return normalized;
}

function timestamp(value, label) {
  if (typeof value !== "string") throw new ExecutorError("INVALID_ARGUMENT", `${label} is invalid`);
  const parsed = new Date(value);
  if (Number.isNaN(parsed.valueOf()) || parsed.toISOString() !== value) throw new ExecutorError("INVALID_ARGUMENT", `${label} is invalid`);
  return parsed;
}

export function canonicalJSON(value) {
  if (value === null || typeof value !== "object") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
  return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
}

export function digest(value) {
  const material = typeof value === "string" ? value : canonicalJSON(value);
  return `sha256:${createHash("sha256").update(material).digest("hex")}`;
}

function hmac(value, key) {
  return createHmac("sha256", key).update(canonicalJSON(value)).digest("hex");
}

function safeMAC(actual, expected) {
  const left = Buffer.from(String(actual), "hex");
  const right = Buffer.from(String(expected), "hex");
  return left.length === 32 && right.length === 32 && timingSafeEqual(left, right);
}

function parseJSON(raw, code, message) {
  try { return JSON.parse(raw); } catch { throw new ExecutorError(code, message); }
}

export function parsePolicy(raw) {
  if (typeof raw !== "string" || raw.length < 2 || raw.length > 128 * 1024) throw new ExecutorError("CONFIGURATION_ERROR", "AD_PLATFORM_EXECUTOR_POLICY is unavailable or too large");
  const value = plain(parseJSON(raw, "CONFIGURATION_ERROR", "AD_PLATFORM_EXECUTOR_POLICY must be valid JSON"), "executor policy");
  exact(value, ["environment_tier", "allow_live", "allowed_targets", "max_timeout_ms", "google_ads_api_version", "dv360_api_version"], "executor policy");
  if (!['sandbox', 'live'].includes(value.environment_tier) || (value.environment_tier === "live" && value.allow_live !== true)) throw new ExecutorError("CONFIGURATION_ERROR", "live execution requires an explicit allow_live policy");
  if (!Array.isArray(value.allowed_targets) || value.allowed_targets.length < 1 || value.allowed_targets.length > 100) throw new ExecutorError("CONFIGURATION_ERROR", "allowed_targets must contain 1 to 100 targets");
  const targets = value.allowed_targets.map((item) => {
    plain(item, "allowed target");
    exact(item, ["provider", "account_id", "resource_type", "resource_ids", "operations"], "allowed target");
    if (!['google_ads', 'dv360'].includes(item.provider)) throw new ExecutorError("CONFIGURATION_ERROR", "allowed target provider is unsupported");
    const accountID = numericID(item.account_id, "allowed target account_id");
    if (typeof item.resource_type !== "string" || !supportedResource(item.provider, item.resource_type)) throw new ExecutorError("CONFIGURATION_ERROR", "allowed target resource_type is unsupported");
    const resourceIDs = list(item.resource_ids, (entry) => numericID(entry, "allowed target resource_id"), 100, "resource_ids");
    const operations = list(item.operations, (entry) => supportedOperation(item.provider, item.resource_type, entry), 4, "operations");
    return { provider: item.provider, account_id: accountID, resource_type: item.resource_type, resource_ids: resourceIDs, operations };
  });
  const timeout = value.max_timeout_ms ?? DEFAULT_TIMEOUT_MS;
  if (!Number.isSafeInteger(timeout) || timeout < 1_000 || timeout > MAX_TIMEOUT_MS) throw new ExecutorError("CONFIGURATION_ERROR", "max_timeout_ms is invalid");
  if ((value.google_ads_api_version ?? "v25") !== "v25" || (value.dv360_api_version ?? "v4") !== "v4") throw new ExecutorError("CONFIGURATION_ERROR", "provider API version is unsupported");
  return { environmentTier: value.environment_tier, allowLive: value.allow_live === true, targets, maxTimeoutMs: timeout, googleAdsVersion: "v25", dv360Version: "v4" };
}

function list(value, normalize, maximum, label) {
  if (!Array.isArray(value) || value.length < 1 || value.length > maximum) throw new ExecutorError("CONFIGURATION_ERROR", `${label} has an invalid number of values`);
  const normalized = value.map(normalize);
  if (new Set(normalized).size !== normalized.length) throw new ExecutorError("CONFIGURATION_ERROR", `${label} contains duplicate values`);
  return normalized;
}

function supportedResource(provider, resourceType) {
  return provider === "google_ads" ? ["campaign", "campaign_budget", "ad_group"].includes(resourceType) : resourceType === "line_item";
}

function supportedOperation(provider, resourceType, operation) {
  const allowed = provider === "google_ads"
    ? { campaign: ["set_status"], campaign_budget: ["set_budget_micros"], ad_group: ["set_status", "set_bid_micros"] }[resourceType]
    : ["set_status", "set_bid_micros"];
  if (!allowed?.includes(operation)) throw new ExecutorError("CONFIGURATION_ERROR", "allowed target operation is unsupported");
  return operation;
}

export function parseCredential(raw) {
  if (typeof raw !== "string" || raw.length < 2 || raw.length > 128 * 1024) throw new ExecutorError("AUTH_MISSING", "AD_PLATFORM_PROVIDER_CREDENTIAL is unavailable or too large");
  const value = plain(parseJSON(raw, "AUTH_INVALID", "AD_PLATFORM_PROVIDER_CREDENTIAL must be valid JSON"), "provider credential");
  exact(value, ["type", "access_token", "developer_token", "login_customer_id"], "provider credential");
  if (value.type !== "access_token" || typeof value.access_token !== "string" || value.access_token.length < 20 || value.access_token.length > 8192) throw new ExecutorError("AUTH_INVALID", "provider access-token credential is incomplete");
  if (value.developer_token !== undefined && (typeof value.developer_token !== "string" || value.developer_token.length < 8 || value.developer_token.length > 256)) throw new ExecutorError("AUTH_INVALID", "Google Ads developer token is invalid");
  const loginCustomerID = value.login_customer_id === undefined ? null : numericID(value.login_customer_id, "login_customer_id");
  return { accessToken: value.access_token, developerToken: value.developer_token ?? null, loginCustomerID };
}

function parseDiff(value, operation) {
  plain(value, "proposed_diff");
  exact(value, ["field", "from", "to"], "proposed_diff");
  const expectedField = { set_status: "status", set_budget_micros: "budget_micros", set_bid_micros: "bid_micros" }[operation];
  if (value.field !== expectedField) throw new ExecutorError("INVALID_ARGUMENT", "proposed_diff field does not match operation");
  if (operation === "set_status") {
    const allowed = ["ENABLED", "PAUSED"];
    if (!allowed.includes(value.from) || !allowed.includes(value.to) || value.from === value.to) throw new ExecutorError("INVALID_ARGUMENT", "status diff is invalid");
  } else if (![value.from, value.to].every((entry) => Number.isSafeInteger(entry) && entry >= 1 && entry <= 1_000_000_000_000) || value.from === value.to) {
    throw new ExecutorError("INVALID_ARGUMENT", "numeric diff is invalid");
  }
  return { field: value.field, from: value.from, to: value.to };
}

export function validatePlan(raw, { now = () => new Date() } = {}) {
  const value = plain(raw, "change plan");
  exact(value, ["schema_version", "plan_id", "action_id", "provider", "account_id", "resource_type", "resource_id", "operation", "proposed_diff", "generated_at", "expires_at"], "change plan");
  if (value.schema_version !== PLAN_VERSION) throw new ExecutorError("INVALID_ARGUMENT", "change plan schema version is unsupported");
  const provider = value.provider;
  if (!['google_ads', 'dv360'].includes(provider)) throw new ExecutorError("INVALID_ARGUMENT", "provider is unsupported");
  const resourceType = String(value.resource_type ?? "");
  if (!supportedResource(provider, resourceType)) throw new ExecutorError("INVALID_ARGUMENT", "resource_type is unsupported");
  supportedOperation(provider, resourceType, value.operation);
  const generatedAt = timestamp(value.generated_at, "generated_at");
  const expiresAt = timestamp(value.expires_at, "expires_at");
  const current = now();
  if (generatedAt > current || expiresAt <= current || expiresAt <= generatedAt || expiresAt.valueOf() - generatedAt.valueOf() > 15 * 60_000) throw new ExecutorError("PLAN_EXPIRED", "change plan lifetime is invalid or expired");
  return {
    schema_version: PLAN_VERSION,
    plan_id: identifier(value.plan_id, "plan_id"),
    action_id: identifier(value.action_id, "action_id"),
    provider,
    account_id: numericID(value.account_id, "account_id"),
    resource_type: resourceType,
    resource_id: numericID(value.resource_id, "resource_id"),
    operation: value.operation,
    proposed_diff: parseDiff(value.proposed_diff, value.operation),
    generated_at: generatedAt.toISOString(),
    expires_at: expiresAt.toISOString(),
  };
}

function targetFor(plan) {
  return { provider: plan.provider, account_id: plan.account_id, resource_type: plan.resource_type, resource_id: plan.resource_id };
}

function assertPlanAuthority(plan, grant, policy, capability = "ad_platform.execute") {
  plain(grant, "execution grant");
  if (grant.capability !== capability || grant.action_id !== plan.action_id || grant.inputs_hash !== digest(plan) || canonicalJSON(grant.target) !== canonicalJSON(targetFor(plan))) throw new ExecutorError("AUTHORITY_MISMATCH", "execution grant does not bind the exact change plan");
  if (!SHA256.test(grant.inputs_hash)) throw new ExecutorError("AUTHORITY_MISMATCH", "execution grant input digest is invalid");
  assertTargetAllowed(plan, policy);
}

function assertTargetAllowed(plan, policy) {
  const allowed = policy.targets.some((target) => target.provider === plan.provider && target.account_id === plan.account_id && target.resource_type === plan.resource_type && target.resource_ids.includes(plan.resource_id) && target.operations.includes(plan.operation));
  if (!allowed) throw new ExecutorError("TARGET_NOT_ALLOWED", "change target or operation is outside executor policy");
}

function absoluteDirectory(value, label) {
  if (typeof value !== "string" || !path.isAbsolute(value) || value.length > 4096) throw new ExecutorError("CONFIGURATION_ERROR", `${label} must be an absolute path`);
  return path.resolve(value);
}

function parseControlPlaneIntegrity(raw) {
  if (typeof raw !== "string" || raw.length < 2 || raw.length > 4096) throw new ExecutorError("CONFIGURATION_ERROR", "CONTROL_PLANE_PACKAGE_INTEGRITY is unavailable or too large");
  const value = plain(parseJSON(raw, "CONFIGURATION_ERROR", "CONTROL_PLANE_PACKAGE_INTEGRITY must be valid JSON"), "control-plane package integrity");
  exact(value, ["version", "runtime", "tree_sha256", "runtime_contract_sha256"], "control-plane package integrity");
  if (value.version !== "0.2.1" || !["codex", "claude", "generic"].includes(value.runtime) || !RAW_SHA256.test(value.tree_sha256) || !RAW_SHA256.test(value.runtime_contract_sha256)) throw new ExecutorError("CONFIGURATION_ERROR", "control-plane package integrity binding is invalid");
  return { version: value.version, runtime: value.runtime, treeSHA256: value.tree_sha256, runtimeContractSHA256: value.runtime_contract_sha256 };
}

export function contextFromEnv(env = process.env, options = {}) {
  const auditKey = env.AD_PLATFORM_EXECUTOR_AUDIT_KEY;
  if (typeof auditKey !== "string" || Buffer.byteLength(auditKey) < 32 || Buffer.byteLength(auditKey) > 4096) throw new ExecutorError("CONFIGURATION_ERROR", "executor audit key must contain 32 to 4096 bytes");
  return {
    env,
    policy: options.policy ?? parsePolicy(env.AD_PLATFORM_EXECUTOR_POLICY),
    credential: options.credential ?? parseCredential(env.AD_PLATFORM_PROVIDER_CREDENTIAL),
    storeRoot: absoluteDirectory(env.AD_PLATFORM_EXECUTOR_STORE, "AD_PLATFORM_EXECUTOR_STORE"),
    controlPlaneRoot: absoluteDirectory(env.CONTROL_PLANE_PACKAGE_ROOT, "CONTROL_PLANE_PACKAGE_ROOT"),
    controlPlaneIntegrity: options.controlPlaneIntegrity ?? parseControlPlaneIntegrity(env.CONTROL_PLANE_PACKAGE_INTEGRITY),
    auditKey,
    now: options.now ?? (() => new Date()),
    clock: options.clock ?? Date.now,
    fetchImpl: options.fetchImpl ?? fetch,
    provider: options.provider,
    controlPlane: options.controlPlane,
    fault: options.fault ?? (() => {}),
  };
}

function remaining(deadline, clock) {
  const duration = deadline - clock();
  if (duration <= 0) throw new ExecutorError("UPSTREAM_TIMEOUT", "operation exceeded its absolute deadline", { retryable: true, ambiguous: true });
  return duration;
}

async function readBoundedJSON(response, controller, deadline, clock) {
  const declared = Number(response.headers.get("content-length") ?? 0);
  if (Number.isFinite(declared) && declared > MAX_UPSTREAM_BYTES) throw new ExecutorError("UPSTREAM_RESPONSE_TOO_LARGE", "provider response exceeds the safe size limit");
  const reader = response.body?.getReader();
  const chunks = [];
  let size = 0;
  if (reader) {
    while (true) {
      remaining(deadline, clock);
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > MAX_UPSTREAM_BYTES) { await reader.cancel(); throw new ExecutorError("UPSTREAM_RESPONSE_TOO_LARGE", "provider response exceeds the safe size limit"); }
      chunks.push(Buffer.from(value));
    }
  }
  try { return JSON.parse(Buffer.concat(chunks, size).toString("utf8")); }
  catch { controller.abort(); throw new ExecutorError("UPSTREAM_MALFORMED", "provider returned malformed JSON", { ambiguous: !response.ok }); }
}

async function fetchJSON(url, init, ctx, deadline, { mutation = false } = {}) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), remaining(deadline, ctx.clock));
  try {
    const response = await ctx.fetchImpl(url, { ...init, redirect: "error", signal: controller.signal });
    const payload = await readBoundedJSON(response, controller, deadline, ctx.clock);
    remaining(deadline, ctx.clock);
    if (!response.ok) {
      const retryable = response.status === 429 || response.status >= 500;
      throw new ExecutorError(response.status === 401 ? "AUTH_REVOKED" : response.status === 403 ? "AUTH_FORBIDDEN" : retryable ? "UPSTREAM_RETRYABLE" : "PROVIDER_REJECTED", "provider rejected the bounded request", { retryable, ambiguous: mutation && retryable, status: response.status });
    }
    return { response, payload };
  } catch (error) {
    if (error instanceof ExecutorError) throw error;
    if (error?.name === "AbortError") throw new ExecutorError("UPSTREAM_TIMEOUT", "operation exceeded its absolute deadline", { retryable: true, ambiguous: mutation });
    throw new ExecutorError("UPSTREAM_UNAVAILABLE", "provider request could not be completed", { retryable: true, ambiguous: mutation });
  } finally { clearTimeout(timer); }
}

function trustedOrigin(env, name, canonical) {
  const raw = env[name];
  let parsed;
  try { parsed = new URL(raw || canonical); } catch { throw new ExecutorError("CONFIGURATION_ERROR", `${name} is invalid`); }
  const loopback = ["localhost", "127.0.0.1", "::1"].includes(parsed.hostname);
  const test = raw && env.AD_PLATFORM_ALLOW_LOOPBACK_TEST_ENDPOINTS === "1" && loopback && parsed.protocol === "http:" && parsed.pathname === "/" && !parsed.search && !parsed.hash;
  if (parsed.username || parsed.password || (!test && parsed.origin !== canonical) || (!test && parsed.protocol !== "https:") || parsed.pathname !== "/" || parsed.search || parsed.hash) throw new ExecutorError("CONFIGURATION_ERROR", `${name} is not trusted`);
  return parsed.origin;
}

function providerHeaders(plan, credential) {
  const headers = { accept: "application/json", authorization: `Bearer ${credential.accessToken}`, "content-type": "application/json" };
  if (plan.provider === "google_ads") {
    if (credential.developerToken) headers["developer-token"] = credential.developerToken;
    if (credential.loginCustomerID) headers["login-customer-id"] = credential.loginCustomerID;
  }
  return headers;
}

function providerReference(providerName, raw, fallback) {
  const material = typeof raw === "string" && raw.length > 0 && raw.length <= 1024 ? raw : canonicalJSON(fallback);
  return `${providerName}:${digest(material)}`;
}

function googleSpec(plan) {
  if (plan.resource_type === "campaign") return { entity: "campaign", response: "campaign", field: "status", jsonField: "status", service: "campaigns" };
  if (plan.resource_type === "campaign_budget") return { entity: "campaign_budget", response: "campaignBudget", field: "amount_micros", jsonField: "amountMicros", service: "campaignBudgets" };
  if (plan.proposed_diff.field === "status") return { entity: "ad_group", response: "adGroup", field: "status", jsonField: "status", service: "adGroups" };
  return { entity: "ad_group", response: "adGroup", field: "cpc_bid_micros", jsonField: "cpcBidMicros", service: "adGroups" };
}

function normalizeProviderState(plan, payload) {
  const field = plan.proposed_diff.field;
  if (plan.provider === "google_ads") {
    const spec = googleSpec(plan);
    const batches = Array.isArray(payload) ? payload : [];
    const rows = batches.flatMap((batch) => Array.isArray(batch.results) ? batch.results : []);
    const expectedName = `customers/${plan.account_id}/${spec.service}/${plan.resource_id}`;
    if (rows.length !== 1 || !rows[0][spec.response] || rows[0][spec.response].resourceName !== expectedName) throw new ExecutorError("PROVIDER_STATE_INVALID", "Google Ads did not return the exact target resource");
    const raw = rows[0][spec.response][spec.jsonField];
    const value = field === "status" ? String(raw) : Number(raw);
    if ((field === "status" && !["ENABLED", "PAUSED"].includes(value)) || (field !== "status" && !Number.isSafeInteger(value))) throw new ExecutorError("PROVIDER_STATE_INVALID", "Google Ads returned invalid target state");
    return { target: targetFor(plan), field, value };
  }
  if (String(payload?.advertiserId ?? "") !== plan.account_id || String(payload?.lineItemId ?? "") !== plan.resource_id) throw new ExecutorError("PROVIDER_STATE_INVALID", "DV360 returned a different line item");
  const raw = field === "status" ? payload.entityStatus : payload.bidStrategy?.maxAverageCpmBidAmountMicros;
  const value = field === "status" ? String(raw) : Number(raw);
  if ((field === "status" && !["ENTITY_STATUS_ACTIVE", "ENTITY_STATUS_PAUSED"].includes(value)) || (field !== "status" && !Number.isSafeInteger(value))) throw new ExecutorError("PROVIDER_STATE_INVALID", "DV360 returned invalid line-item state");
  return { target: targetFor(plan), field, value: field === "status" ? value.replace("ENTITY_STATUS_ACTIVE", "ENABLED").replace("ENTITY_STATUS_PAUSED", "PAUSED") : value };
}

async function defaultRead(plan, ctx, deadline) {
  if (plan.provider === "google_ads") {
    const origin = trustedOrigin(ctx.env, "GOOGLE_ADS_API_ORIGIN", "https://googleads.googleapis.com");
    const spec = googleSpec(plan);
    const query = `SELECT ${spec.entity}.resource_name, ${spec.entity}.${spec.field} FROM ${spec.entity} WHERE ${spec.entity}.id = ${plan.resource_id}`;
    const url = `${origin}/${ctx.policy.googleAdsVersion}/customers/${plan.account_id}/googleAds:searchStream`;
    const { payload } = await fetchJSON(url, { method: "POST", headers: providerHeaders(plan, ctx.credential), body: JSON.stringify({ query }) }, ctx, deadline);
    return normalizeProviderState(plan, payload);
  }
  const origin = trustedOrigin(ctx.env, "DV360_API_ORIGIN", "https://displayvideo.googleapis.com");
  const url = `${origin}/${ctx.policy.dv360Version}/advertisers/${plan.account_id}/lineItems/${plan.resource_id}`;
  const { payload } = await fetchJSON(url, { method: "GET", headers: providerHeaders(plan, ctx.credential) }, ctx, deadline);
  return normalizeProviderState(plan, payload);
}

async function defaultMutate(plan, value, ctx, deadline) {
  if (plan.provider === "google_ads") {
    const origin = trustedOrigin(ctx.env, "GOOGLE_ADS_API_ORIGIN", "https://googleads.googleapis.com");
    const spec = googleSpec(plan);
    const resourceName = `customers/${plan.account_id}/${spec.service === "campaignBudgets" ? "campaignBudgets" : spec.service}/${plan.resource_id}`;
    const update = { resourceName, [spec.jsonField]: value };
    const body = { operations: [{ update, updateMask: spec.field }] };
    const url = `${origin}/${ctx.policy.googleAdsVersion}/customers/${plan.account_id}/${spec.service}:mutate`;
    const { response, payload } = await fetchJSON(url, { method: "POST", headers: providerHeaders(plan, ctx.credential), body: JSON.stringify(body) }, ctx, deadline, { mutation: true });
    if (!Array.isArray(payload?.results) || payload.results.length !== 1 || payload.results[0]?.resourceName !== resourceName) throw new ExecutorError("UPSTREAM_MALFORMED", "Google Ads mutation receipt does not bind the exact target", { ambiguous: true });
    const requestID = providerReference("google-ads-request", response.headers.get("request-id") || response.headers.get("google-ads-request-id"), payload);
    return { requestID, responseHash: digest(payload) };
  }
  const origin = trustedOrigin(ctx.env, "DV360_API_ORIGIN", "https://displayvideo.googleapis.com");
  const field = plan.proposed_diff.field === "status" ? "entityStatus" : "bidStrategy.maxAverageCpmBidAmountMicros";
  const body = plan.proposed_diff.field === "status"
    ? { advertiserId: plan.account_id, lineItemId: plan.resource_id, entityStatus: value === "ENABLED" ? "ENTITY_STATUS_ACTIVE" : "ENTITY_STATUS_PAUSED" }
    : { advertiserId: plan.account_id, lineItemId: plan.resource_id, bidStrategy: { maxAverageCpmBidAmountMicros: value } };
  const url = `${origin}/${ctx.policy.dv360Version}/advertisers/${plan.account_id}/lineItems/${plan.resource_id}?updateMask=${encodeURIComponent(field)}`;
  const { response, payload } = await fetchJSON(url, { method: "PATCH", headers: providerHeaders(plan, ctx.credential), body: JSON.stringify(body) }, ctx, deadline, { mutation: true });
  const requestID = providerReference("dv360-request", response.headers.get("x-guploader-uploadid") || response.headers.get("x-request-id"), payload);
  return { requestID, responseHash: digest(payload) };
}

function provider(ctx) {
  return ctx.provider ?? { read: (plan, deadline) => defaultRead(plan, ctx, deadline), mutate: (plan, value, deadline) => defaultMutate(plan, value, ctx, deadline) };
}

async function ensureStore(ctx) {
  await mkdir(ctx.storeRoot, { recursive: true, mode: 0o700 });
  const info = await lstat(ctx.storeRoot);
  if (!info.isDirectory() || info.isSymbolicLink()) throw new ExecutorError("CONFIGURATION_ERROR", "executor store must be a real directory");
  const resolved = await realpath(ctx.storeRoot);
  ctx.storeRoot = resolved;
  for (const child of ["intents", "receipts", "locks"]) await mkdir(path.join(ctx.storeRoot, child), { recursive: true, mode: 0o700 });
}

function storageKey(value) { return createHash("sha256").update(canonicalJSON(value)).digest("hex"); }
function executionID(grant, plan) { return `exec_${storageKey({ grant_id: grant.grant_id, plan: digest(plan) }).slice(0, 32)}`; }
function recordPath(ctx, type, id) { return path.join(ctx.storeRoot, type, `${storageKey(id)}.json`); }

async function syncDirectory(directory) {
  const handle = await open(directory, "r");
  try { await handle.sync(); } finally { await handle.close(); }
}

async function atomicWrite(file, value, { replace = true } = {}) {
  const directory = path.dirname(file);
  await mkdir(directory, { recursive: true, mode: 0o700 });
  const temporary = path.join(directory, `.${path.basename(file)}.${process.pid}.${Date.now()}.tmp`);
  const handle = await open(temporary, "wx", 0o600);
  try { await handle.writeFile(`${JSON.stringify(value)}\n`); await handle.sync(); } finally { await handle.close(); }
  try {
    if (!replace) {
      const target = await open(file, "wx", 0o600);
      try { await target.writeFile(`${JSON.stringify(value)}\n`); await target.sync(); } finally { await target.close(); }
      await unlink(temporary);
    } else await rename(temporary, file);
    await syncDirectory(directory);
  } catch (error) { await unlink(temporary).catch(() => {}); throw error; }
}

async function writeAuthenticated(ctx, type, id, value, options = {}) {
  const envelope = { value, mac: hmac(value, ctx.auditKey) };
  await atomicWrite(recordPath(ctx, type, id), envelope, options);
}

async function readAuthenticated(ctx, type, id, { required = true } = {}) {
  const file = recordPath(ctx, type, id);
  let raw;
  try { raw = await readFile(file, "utf8"); }
  catch (error) { if (!required && error?.code === "ENOENT") return null; throw new ExecutorError("EXECUTION_STATE_INVALID", "authenticated execution state is unavailable"); }
  const envelope = parseJSON(raw, "EXECUTION_STATE_INVALID", "authenticated execution state is malformed");
  if (!plain(envelope, "execution state") || !safeMAC(envelope.mac, hmac(envelope.value, ctx.auditKey))) throw new ExecutorError("EXECUTION_STATE_INVALID", "authenticated execution state failed verification");
  return envelope.value;
}

async function withResourceLock(ctx, target, fn) {
  await ensureStore(ctx);
  const lock = path.join(ctx.storeRoot, "locks", storageKey(target));
  const deadline = ctx.clock() + Math.min(10_000, ctx.policy.maxTimeoutMs);
  while (true) {
    try { await mkdir(lock, { mode: 0o700 }); break; }
    catch (error) {
      if (error?.code !== "EEXIST") throw new ExecutorError("EXECUTION_STORE_ERROR", "could not acquire resource lock", { retryable: true });
      let age = 0;
      try { age = ctx.clock() - (await stat(lock)).mtimeMs; } catch {}
      if (age > 2 * MAX_TIMEOUT_MS) { await rmdir(lock).catch(() => {}); continue; }
      if (ctx.clock() >= deadline) throw new ExecutorError("RESOURCE_BUSY", "another execution owns the target resource", { retryable: true });
      await new Promise((resolve) => setTimeout(resolve, 10));
    }
  }
  try { return await fn(); }
  finally { await rmdir(lock).catch(() => {}); await syncDirectory(path.dirname(lock)).catch(() => {}); }
}

function controlPlaneEnvironment(ctx) {
  const names = ["CONTROL_PLANE_STORE", "CONTROL_PLANE_IDENTITY", "CONTROL_PLANE_IDENTITY_PUBLIC_KEY", "CONTROL_PLANE_GRANT_ROLE_KEY", "CONTROL_PLANE_AUDIT_KEY"];
  const env = {};
  for (const name of names) if (typeof ctx.env[name] === "string") env[name] = ctx.env[name];
  return env;
}

export async function packageTreeSHA256(root) {
  const entries = [];
  async function walk(directory, relativeDirectory = "") {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      const relative = relativeDirectory ? `${relativeDirectory}/${entry.name}` : entry.name;
      if (relative === ".skills-hub-install.json") continue;
      const full = path.join(directory, entry.name);
      const info = await lstat(full);
      if (info.isSymbolicLink() || (!info.isDirectory() && !info.isFile())) throw new ExecutorError("CONTROL_PLANE_INTEGRITY_INVALID", "control-plane package contains an unsupported entry");
      entries.push({ relative, full, directory: info.isDirectory(), executable: !info.isDirectory() && (info.mode & 0o111) !== 0 });
      if (info.isDirectory()) await walk(full, relative);
    }
  }
  await walk(root);
  entries.sort((left, right) => left.relative.localeCompare(right.relative));
  const hash = createHash("sha256");
  for (const entry of entries) {
    const kind = entry.directory ? "dir" : "file";
    const mode = entry.directory || entry.executable ? "755" : "644";
    hash.update(`${kind}\0${entry.relative}\0${mode}\0`);
    if (!entry.directory) hash.update(await readFile(entry.full));
    hash.update("\0");
  }
  return hash.digest("hex");
}

async function verifyControlPlanePackage(ctx, root) {
  const receiptPath = path.join(root, ".skills-hub-install.json");
  let receipt;
  try {
    const raw = await readFile(receiptPath, "utf8");
    if (Buffer.byteLength(raw) > 64 * 1024) throw new Error("oversized receipt");
    receipt = plain(parseJSON(raw, "CONTROL_PLANE_INTEGRITY_INVALID", "control-plane install receipt is invalid"), "control-plane install receipt");
  } catch (error) {
    if (error instanceof ExecutorError) throw error;
    throw new ExecutorError("CONTROL_PLANE_INTEGRITY_INVALID", "control-plane install receipt is unavailable");
  }
  const expected = ctx.controlPlaneIntegrity;
  if (receipt.schema_version !== "skills-hub.install-receipt/v1" || receipt.module !== "tools" || receipt.id !== "agentops/agent-control-plane-server" || receipt.version !== expected.version || receipt.runtime !== expected.runtime || receipt.tree_sha256 !== expected.treeSHA256 || receipt.runtime_contract_sha256 !== expected.runtimeContractSHA256) throw new ExecutorError("CONTROL_PLANE_INTEGRITY_INVALID", "control-plane install receipt does not match the pinned dependency");
  const actualTree = await packageTreeSHA256(root);
  if (actualTree !== expected.treeSHA256) throw new ExecutorError("CONTROL_PLANE_INTEGRITY_INVALID", "control-plane package contents do not match the pinned dependency");
}

async function defaultControlPlane(ctx, tool, args, deadline) {
  let root;
  try {
    const rootInfo = await lstat(ctx.controlPlaneRoot);
    root = await realpath(ctx.controlPlaneRoot);
    if (!rootInfo.isDirectory() || rootInfo.isSymbolicLink() || root !== ctx.controlPlaneRoot) throw new Error("noncanonical package root");
  } catch { throw new ExecutorError("CONFIGURATION_ERROR", "CONTROL_PLANE_PACKAGE_ROOT must be a canonical package directory"); }
  const script = path.join(root, "scripts", "agent_control_plane_server.mjs");
  const manifest = path.join(root, "tool.yaml");
  try {
    await access(script, fsConstants.R_OK);
    if ((await lstat(script)).isSymbolicLink()) throw new Error("symlinked entrypoint");
    const manifestText = await readFile(manifest, "utf8");
    if (!/^id:\s+agentops\/agent-control-plane-server\s*$/m.test(manifestText)) throw new Error("wrong package");
  } catch { throw new ExecutorError("CONFIGURATION_ERROR", "CONTROL_PLANE_PACKAGE_ROOT is not the Agent Control Plane package"); }
  await verifyControlPlanePackage(ctx, root);
  const child = spawn(process.execPath, [script], { env: controlPlaneEnvironment(ctx), stdio: ["pipe", "pipe", "pipe"] });
  const request = { jsonrpc: "2.0", id: 1, method: "tools/call", params: { name: tool, arguments: args } };
  child.stdin.end(`${JSON.stringify(request)}\n`);
  const chunks = [];
  let size = 0;
  child.stdout.on("data", (chunk) => { size += chunk.length; if (size <= MAX_UPSTREAM_BYTES) chunks.push(chunk); else child.kill(); });
  child.stderr.resume();
  const timeout = setTimeout(() => child.kill(), remaining(deadline, ctx.clock));
  const exitCode = await new Promise((resolve, reject) => { child.once("error", reject); child.once("exit", resolve); });
  clearTimeout(timeout);
  if (size > MAX_UPSTREAM_BYTES || exitCode !== 0) throw new ExecutorError("CONTROL_PLANE_UNAVAILABLE", "control plane could not complete the effect-boundary check", { retryable: true });
  const line = Buffer.concat(chunks).toString("utf8").trim().split("\n").filter(Boolean).at(-1);
  const response = parseJSON(line, "CONTROL_PLANE_INVALID", "control plane returned an invalid response");
  if (response.error || response.result?.isError) {
    const safe = response.error?.data ?? response.result?.structuredContent?.error ?? {};
    throw new ExecutorError(typeof safe.code === "string" ? safe.code : "CONTROL_PLANE_REJECTED", typeof safe.message === "string" ? safe.message : "control plane rejected execution", { retryable: safe.retryable === true });
  }
  return response.result?.structuredContent;
}

async function callControlPlane(ctx, tool, args, deadline) {
  remaining(deadline, ctx.clock);
  return ctx.controlPlane ? ctx.controlPlane(tool, args, deadline) : defaultControlPlane(ctx, tool, args, deadline);
}

async function claimGrant(ctx, grant, executionIDValue, deadline, faultPoint) {
  try {
    const claim = await callControlPlane(ctx, "validate_execution_grant", { grant }, deadline);
    ctx.fault(faultPoint);
    if (typeof claim?.claim_id !== "string") throw new ExecutorError("CONTROL_PLANE_INVALID", "control plane returned an invalid execution claim");
    return { ...claim, recovered: false };
  } catch (error) {
    if (error?.code !== "ACTION_ALREADY_CLAIMED") throw error;
    const claim = await callControlPlane(ctx, "get_execution_claim", { grant }, deadline);
    if (claim?.valid !== true || typeof claim.claim_id !== "string" || claim.grant_id !== grant.grant_id || claim.inputs_hash !== grant.inputs_hash || canonicalJSON(claim.target) !== canonicalJSON(grant.target)) throw new ExecutorError("CONTROL_PLANE_INVALID", "control-plane claim lookup did not return the exact execution claim");
    return { ...claim, recovered: true, recovery_reference: `executor:${executionIDValue}:claim-recovered` };
  }
}

async function finishAuthorityCancellation(ctx, grant, executionIDValue, intent, deadline, operation) {
  if (intent.state !== "cancellation_pending" || typeof intent.cancellation_code !== "string" || typeof intent.cancellation_provider_receipt !== "string") throw new ExecutorError("EXECUTION_STATE_INVALID", "authority cancellation state is incomplete");
  await callControlPlane(ctx, "record_agent_action", {
    grant,
    status: "cancelled",
    outputs_hash: null,
    provider_receipt: intent.cancellation_provider_receipt,
  }, deadline);
  ctx.fault(`after_${operation}_cancellation_action`);
  await writeAuthenticated(ctx, "intents", executionIDValue, { ...intent, state: "terminal_no_effect", completed_at: ctx.now().toISOString() });
  throw new ExecutorError(intent.cancellation_code, "current execution authority ended before the provider effect began");
}

async function revalidateClaimBeforeEffect(ctx, grant, executionIDValue, intent, deadline, operation) {
  try {
    const current = await callControlPlane(ctx, "revalidate_execution_claim", { grant }, deadline);
    if (current?.valid !== true || current.current !== true || current.claim_id !== intent.claim_id || current.grant_id !== grant.grant_id || current.inputs_hash !== grant.inputs_hash || canonicalJSON(current.target) !== canonicalJSON(grant.target)) throw new ExecutorError("CONTROL_PLANE_INVALID", "control plane did not confirm current authority for the exact claim");
    return current;
  } catch (error) {
    const authorityEnded = ["GRANT_EXPIRED", "POLICY_CHANGED", "APPROVAL_EXPIRED", "SCOPE_MISMATCH", "GRANT_INVALID", "AUTH_EXPIRED"].includes(error?.code);
    if (!authorityEnded) throw error;
    const pending = {
      ...intent,
      state: "cancellation_pending",
      cancellation_code: error.code,
      cancellation_provider_receipt: `executor:${executionIDValue}:authority-ended`,
      cancellation_requested_at: ctx.now().toISOString(),
    };
    await writeAuthenticated(ctx, "intents", executionIDValue, pending);
    ctx.fault(`after_${operation}_cancellation_pending`);
    return finishAuthorityCancellation(ctx, grant, executionIDValue, pending, deadline, operation);
  }
}

export async function verifyExecutorAuthority(ctx) {
  const deadline = ctx.clock() + ctx.policy.maxTimeoutMs;
  const audit = await callControlPlane(ctx, "verify_audit_chain", {}, deadline);
  if (audit?.valid !== true) throw new ExecutorError("CONTROL_PLANE_INVALID", "control-plane audit authority is invalid");
  const identity = plain(parseJSON(ctx.env.CONTROL_PLANE_IDENTITY, "AUTH_INVALID", "executor identity binding is invalid"), "executor identity");
  if (!Array.isArray(identity.roles) || !identity.roles.includes("executor_runtime") || identity.roles.includes("agent_runtime")) throw new ExecutorError("AUTH_INVALID", "control-plane identity is not an executor identity");
  const identityExpiry = timestamp(identity.expires_at, "executor identity expires_at");
  if (identityExpiry <= ctx.now()) throw new ExecutorError("AUTH_EXPIRED", "executor identity is expired");
  const target = ctx.policy.targets[0];
  const operation = target.operations[0];
  const field = { set_status: "status", set_budget_micros: "budget_micros", set_bid_micros: "bid_micros" }[operation];
  const plan = {
    schema_version: PLAN_VERSION, plan_id: "readiness-probe", action_id: "readiness-probe",
    provider: target.provider, account_id: target.account_id, resource_type: target.resource_type,
    resource_id: target.resource_ids[0], operation, proposed_diff: { field, from: field === "status" ? "PAUSED" : 1, to: field === "status" ? "ENABLED" : 2 },
    generated_at: ctx.now().toISOString(), expires_at: new Date(ctx.now().valueOf() + 60_000).toISOString(),
  };
  const state = await provider(ctx).read(plan, deadline);
  return {
    principal: identifier(identity.actor_id, "executor actor_id"),
    account: `${target.provider}:${target.account_id}`,
    scopes: [...new Set(ctx.policy.targets.flatMap((item) => item.operations))].sort(),
    tier: ctx.policy.environmentTier,
    targetFingerprint: digest({ target: targetFor(plan), current_state_hash: digest(state) }),
    credentialGeneration: createHash("sha256").update(ctx.env.AD_PLATFORM_PROVIDER_CREDENTIAL).digest("hex").slice(0, 24),
    identityExpiresAt: identityExpiry.toISOString(),
  };
}

function timeoutFromInput(input, policy) {
  const value = input.timeout_ms ?? policy.maxTimeoutMs;
  if (!Number.isSafeInteger(value) || value < 1_000 || value > policy.maxTimeoutMs) throw new ExecutorError("INVALID_ARGUMENT", "timeout_ms is outside policy bounds");
  return value;
}

function assertState(state, expected, code = "STALE_PROVIDER_STATE") {
  if (state.field !== expected.field || state.value !== expected.from) throw new ExecutorError(code, "provider state no longer matches the approved precondition");
}

function publicReceipt(receipt) {
  return {
    schema_version: receipt.schema_version,
    execution_id: receipt.execution_id,
    grant_id: receipt.grant_id,
    plan_id: receipt.plan_id,
    target: receipt.target,
    status: receipt.status,
    pre_image_hash: receipt.pre_image_hash,
    post_image_hash: receipt.post_image_hash,
    rollback_patch_hash: receipt.rollback_patch_hash,
    provider_receipt: receipt.provider_receipt,
    recorded_at: receipt.recorded_at,
    reconciled: receipt.reconciled,
  };
}

async function ensureControlPlaneTerminal(ctx, grant, receipt, deadline) {
  const status = ["executed", "rolled_back"].includes(receipt.status) ? "executed" : "failed";
  return callControlPlane(ctx, "record_agent_action", {
    grant,
    status,
    outputs_hash: receipt.post_image_hash,
    provider_receipt: receipt.provider_receipt,
  }, deadline);
}

async function terminalReceipt(ctx, { executionID: id, grant, plan, pre, post, providerReceipt, status = "executed", reconciled = false }) {
  const patch = { field: plan.proposed_diff.field, from: post.value, to: pre.value };
  const receipt = {
    schema_version: "ad-platform.execution-receipt/v1",
    execution_id: id,
    grant_id: grant.grant_id,
    plan_id: plan.plan_id,
    plan_hash: digest(plan),
    target: targetFor(plan),
    status,
    pre_image: pre,
    post_image: post,
    pre_image_hash: digest(pre),
    post_image_hash: digest(post),
    rollback_patch: patch,
    rollback_patch_hash: digest(patch),
    provider_receipt: providerReceipt,
    recorded_at: ctx.now().toISOString(),
    reconciled,
  };
  await writeAuthenticated(ctx, "receipts", id, receipt, { replace: false });
  return receipt;
}

function assertReceiptIntentBinding(receipt, intent) {
  if (!intent || intent.execution_id !== receipt.execution_id || intent.grant_id !== receipt.grant_id || digest(intent.plan) !== receipt.plan_hash || canonicalJSON(targetFor(intent.plan)) !== canonicalJSON(receipt.target) || digest(receipt.pre_image) !== receipt.pre_image_hash || digest(receipt.post_image) !== receipt.post_image_hash) throw new ExecutorError("EXECUTION_STATE_INVALID", "execution receipt is not bound to its authenticated intent");
  if (intent.state === "terminal" && intent.receipt_hash !== digest(receipt)) throw new ExecutorError("EXECUTION_STATE_INVALID", "terminal execution intent does not bind the authenticated receipt");
}

export async function previewChange(ctx, raw) {
  const input = plain(raw, "preview request");
  exact(input, ["plan", "timeout_ms"], "preview request");
  const plan = validatePlan(input.plan, { now: ctx.now });
  assertPlanAuthority(plan, { capability: "ad_platform.execute", action_id: plan.action_id, inputs_hash: digest(plan), target: targetFor(plan) }, ctx.policy);
  const deadline = ctx.clock() + timeoutFromInput(input, ctx.policy);
  const state = await provider(ctx).read(plan, deadline);
  return { plan_id: plan.plan_id, target: targetFor(plan), mode: "preview", can_apply: state.field === plan.proposed_diff.field && state.value === plan.proposed_diff.from, current_state_hash: digest(state), proposed_post_state_hash: digest({ ...state, value: plan.proposed_diff.to }) };
}

export async function executeApprovedChange(ctx, raw) {
  const input = plain(raw, "execution request");
  exact(input, ["grant", "plan", "timeout_ms"], "execution request");
  const plan = validatePlan(input.plan, { now: ctx.now });
  assertPlanAuthority(plan, input.grant, ctx.policy);
  const deadline = ctx.clock() + timeoutFromInput(input, ctx.policy);
  const id = executionID(input.grant, plan);
  return withResourceLock(ctx, targetFor(plan), async () => {
    const existing = await readAuthenticated(ctx, "receipts", id, { required: false });
    if (existing) {
      const existingIntent = await readAuthenticated(ctx, "intents", id);
      assertReceiptIntentBinding(existing, existingIntent);
      await ensureControlPlaneTerminal(ctx, input.grant, existing, deadline);
      if (existingIntent.state !== "terminal") await writeAuthenticated(ctx, "intents", id, { ...existingIntent, state: "terminal", receipt_hash: digest(existing), completed_at: ctx.now().toISOString() });
      return publicReceipt(existing);
    }
    const oldIntent = await readAuthenticated(ctx, "intents", id, { required: false });
    if (oldIntent?.state === "cancellation_pending") await finishAuthorityCancellation(ctx, input.grant, id, oldIntent, deadline, "execution");
    if (oldIntent?.state === "terminal_no_effect") throw new ExecutorError("EXECUTION_CANCELLED", "execution authority ended before the provider effect began");
    if (oldIntent?.state === "claimed" || oldIntent?.state === "ambiguous") throw new ExecutorError("RECONCILIATION_REQUIRED", "a claimed execution requires provider-state reconciliation", { retryable: false, ambiguous: true });
    const pre = await provider(ctx).read(plan, deadline);
    assertState(pre, plan.proposed_diff);
    let intent = { schema_version: "ad-platform.execution-intent/v1", execution_id: id, grant_id: input.grant.grant_id, plan, plan_hash: digest(plan), pre_image: pre, pre_image_hash: digest(pre), state: "prepared", prepared_at: ctx.now().toISOString() };
    await writeAuthenticated(ctx, "intents", id, intent);
    const claim = await claimGrant(ctx, input.grant, id, deadline, "after_execution_control_plane_claim");
    intent = { ...intent, state: "claimed", claim_id: claim.claim_id, claim_recovered: claim.recovered, claimed_at: ctx.now().toISOString() };
    await writeAuthenticated(ctx, "intents", id, intent);
    await revalidateClaimBeforeEffect(ctx, input.grant, id, intent, deadline, "execution");
    await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "executing", outputs_hash: null, provider_receipt: `executor:${id}:claimed` }, deadline);
    ctx.fault("after_claim");
    let mutation;
    try { mutation = await provider(ctx).mutate(plan, plan.proposed_diff.to, deadline); }
    catch (error) {
      if (error?.ambiguous) { await writeAuthenticated(ctx, "intents", id, { ...intent, state: "ambiguous", ambiguous_at: ctx.now().toISOString() }); throw new ExecutorError("RECONCILIATION_REQUIRED", "provider effect is ambiguous and must be reconciled", { ambiguous: true }); }
      await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "failed", outputs_hash: null, provider_receipt: `executor:${id}:provider-rejected` }, deadline).catch(() => {});
      throw error;
    }
    ctx.fault("after_effect");
    const post = await provider(ctx).read(plan, deadline);
    const providerReceipt = `${mutation.requestID}:${mutation.responseHash}`.slice(0, 512);
    if (post.field !== plan.proposed_diff.field || post.value !== plan.proposed_diff.to) {
      const failedReceipt = await terminalReceipt(ctx, { executionID: id, grant: input.grant, plan, pre, post, providerReceipt, status: "verification_failed" });
      await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "failed", outputs_hash: digest(post), provider_receipt: providerReceipt }, deadline).catch(() => {});
      await writeAuthenticated(ctx, "intents", id, { ...intent, state: "terminal", receipt_hash: digest(failedReceipt), completed_at: ctx.now().toISOString() });
      throw new ExecutorError("MANUAL_RECONCILIATION_REQUIRED", "post-write verification observed conflicting provider state; no compensating mutation was attempted", { ambiguous: true });
    }
    const receipt = await terminalReceipt(ctx, { executionID: id, grant: input.grant, plan, pre, post, providerReceipt });
    ctx.fault("after_receipt");
    await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "executed", outputs_hash: receipt.post_image_hash, provider_receipt: providerReceipt }, deadline);
    await writeAuthenticated(ctx, "intents", id, { ...intent, state: "terminal", receipt_hash: digest(receipt), completed_at: ctx.now().toISOString() });
    return publicReceipt(receipt);
  });
}

export async function reconcileExecution(ctx, raw) {
  const input = plain(raw, "reconciliation request");
  exact(input, ["grant", "plan", "timeout_ms"], "reconciliation request");
  const plan = validatePlan(input.plan, { now: ctx.now });
  assertPlanAuthority(plan, input.grant, ctx.policy);
  const deadline = ctx.clock() + timeoutFromInput(input, ctx.policy);
  const id = executionID(input.grant, plan);
  return withResourceLock(ctx, targetFor(plan), async () => {
    const existing = await readAuthenticated(ctx, "receipts", id, { required: false });
    if (existing) {
      const existingIntent = await readAuthenticated(ctx, "intents", id);
      assertReceiptIntentBinding(existing, existingIntent);
      await ensureControlPlaneTerminal(ctx, input.grant, existing, deadline);
      if (existingIntent.state !== "terminal") await writeAuthenticated(ctx, "intents", id, { ...existingIntent, state: "terminal", receipt_hash: digest(existing), completed_at: ctx.now().toISOString() });
      return publicReceipt(existing);
    }
    const intent = await readAuthenticated(ctx, "intents", id);
    if (!['claimed', 'ambiguous'].includes(intent.state) || intent.grant_id !== input.grant.grant_id || intent.plan_hash && intent.plan_hash !== digest(plan)) throw new ExecutorError("RECONCILIATION_NOT_ALLOWED", "execution is not in a reconcilable state");
    const current = await provider(ctx).read(plan, deadline);
    if (current.field === plan.proposed_diff.field && current.value === plan.proposed_diff.to) {
      const providerReceipt = `reconciled:${id}:${digest(current)}`.slice(0, 512);
      const receipt = await terminalReceipt(ctx, { executionID: id, grant: input.grant, plan, pre: intent.pre_image, post: current, providerReceipt, reconciled: true });
      await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "executed", outputs_hash: receipt.post_image_hash, provider_receipt: providerReceipt }, deadline);
      await writeAuthenticated(ctx, "intents", id, { ...intent, state: "terminal", receipt_hash: digest(receipt), completed_at: ctx.now().toISOString() });
      return publicReceipt(receipt);
    }
    if (current.field === plan.proposed_diff.field && current.value === intent.pre_image.value) {
      const providerReceipt = `reconciled:${id}:no-effect`;
      await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "failed", outputs_hash: digest(current), provider_receipt: providerReceipt }, deadline);
      await writeAuthenticated(ctx, "intents", id, { ...intent, state: "terminal_no_effect", completed_at: ctx.now().toISOString() });
      return { execution_id: id, status: "no_effect", target: targetFor(plan), current_state_hash: digest(current), reconciled: true };
    }
    throw new ExecutorError("MANUAL_RECONCILIATION_REQUIRED", "provider state matches neither approved pre-image nor proposed post-image", { ambiguous: true });
  });
}

export function rollbackRequest(executionIDValue, receipt) {
  return { schema_version: ROLLBACK_VERSION, execution_id: executionIDValue, execution_receipt_hash: digest(receipt), rollback_patch_hash: receipt.rollback_patch_hash };
}

export async function rollbackExecution(ctx, raw) {
  const input = plain(raw, "rollback request");
  exact(input, ["grant", "execution_id", "timeout_ms"], "rollback request");
  const id = identifier(input.execution_id, "execution_id");
  const receipt = await readAuthenticated(ctx, "receipts", id);
  if (receipt.status !== "executed") throw new ExecutorError("ROLLBACK_NOT_SUPPORTED", "only a verified successful execution can be rolled back");
  const request = rollbackRequest(id, receipt);
  const plan = { ...receiptPlan(receipt), action_id: input.grant.action_id, operation: receipt.rollback_patch.field === "status" ? "set_status" : receipt.rollback_patch.field === "budget_micros" ? "set_budget_micros" : "set_bid_micros", proposed_diff: receipt.rollback_patch };
  if (input.grant.capability !== "ad_platform.rollback" || input.grant.inputs_hash !== digest(request) || canonicalJSON(input.grant.target) !== canonicalJSON(receipt.target)) throw new ExecutorError("AUTHORITY_MISMATCH", "rollback grant does not bind the exact execution receipt");
  assertTargetAllowed(plan, ctx.policy);
  const deadline = ctx.clock() + timeoutFromInput(input, ctx.policy);
  return withResourceLock(ctx, receipt.target, async () => {
    const rollbackID = `rollback_${storageKey(input.grant.grant_id).slice(0, 32)}`;
    const existing = await readAuthenticated(ctx, "receipts", rollbackID, { required: false });
    if (existing) {
      const existingIntent = await readAuthenticated(ctx, "intents", rollbackID);
      assertReceiptIntentBinding(existing, existingIntent);
      await ensureControlPlaneTerminal(ctx, input.grant, existing, deadline);
      ctx.fault("after_rollback_terminal_action");
      if (existingIntent.state !== "terminal") await writeAuthenticated(ctx, "intents", rollbackID, { ...existingIntent, state: "terminal", receipt_hash: digest(existing), completed_at: ctx.now().toISOString() });
      return publicReceipt(existing);
    }
    let intent = await readAuthenticated(ctx, "intents", rollbackID, { required: false });
    if (intent && (intent.grant_id !== input.grant.grant_id || intent.request_hash !== digest(request) || intent.execution_receipt_hash !== digest(receipt))) throw new ExecutorError("EXECUTION_STATE_INVALID", "rollback intent does not match the authorized request");
    if (intent?.state === "cancellation_pending") await finishAuthorityCancellation(ctx, input.grant, rollbackID, intent, deadline, "rollback");
    let current = await provider(ctx).read(plan, deadline);
    if (digest(current) === receipt.pre_image_hash) {
      if (!intent || !["effect_started", "ambiguous", "claimed"].includes(intent.state)) throw new ExecutorError("ROLLBACK_CONFLICT", "provider state changed outside a recoverable rollback");
      const providerReceipt = `reconciled:${rollbackID}:${digest(current)}`.slice(0, 512);
      const rollbackReceipt = await terminalReceipt(ctx, { executionID: rollbackID, grant: input.grant, plan, pre: receipt.post_image, post: current, providerReceipt, status: "rolled_back", reconciled: true });
      await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "executed", outputs_hash: rollbackReceipt.post_image_hash, provider_receipt: providerReceipt }, deadline);
      await writeAuthenticated(ctx, "intents", rollbackID, { ...intent, state: "terminal", receipt_hash: digest(rollbackReceipt), completed_at: ctx.now().toISOString() });
      return publicReceipt(rollbackReceipt);
    }
    if (digest(current) !== receipt.post_image_hash) throw new ExecutorError("ROLLBACK_CONFLICT", "provider state matches neither the execution post-image nor rollback result");
    if (intent?.state === "effect_started" || intent?.state === "ambiguous") throw new ExecutorError("RECONCILIATION_REQUIRED", "rollback effect remains ambiguous while the provider reports the original post-image", { ambiguous: true });
    if (!intent) {
      intent = { schema_version: "ad-platform.rollback-intent/v1", execution_id: rollbackID, original_execution_id: id, grant_id: input.grant.grant_id, request_hash: digest(request), execution_receipt_hash: digest(receipt), plan, state: "prepared", prepared_at: ctx.now().toISOString() };
      await writeAuthenticated(ctx, "intents", rollbackID, intent);
      const claim = await claimGrant(ctx, input.grant, rollbackID, deadline, "after_rollback_control_plane_claim");
      intent = { ...intent, state: "claimed", claim_id: claim.claim_id, claim_recovered: claim.recovered, claimed_at: ctx.now().toISOString() };
      await writeAuthenticated(ctx, "intents", rollbackID, intent);
    } else if (intent.state === "prepared") {
      const claim = await claimGrant(ctx, input.grant, rollbackID, deadline, "after_rollback_control_plane_claim");
      intent = { ...intent, state: "claimed", claim_id: claim.claim_id, claim_recovered: claim.recovered, claimed_at: ctx.now().toISOString() };
      await writeAuthenticated(ctx, "intents", rollbackID, intent);
    } else if (intent.state === "terminal_no_effect") throw new ExecutorError("ROLLBACK_CANCELLED", "rollback authority ended before the provider effect began");
    else if (intent.state !== "claimed") throw new ExecutorError("RECONCILIATION_REQUIRED", "rollback is not in a safely resumable state", { ambiguous: true });
    await revalidateClaimBeforeEffect(ctx, input.grant, rollbackID, intent, deadline, "rollback");
    await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "executing", outputs_hash: null, provider_receipt: `executor:${rollbackID}:claimed` }, deadline);
    ctx.fault("after_rollback_claim");
    intent = { ...intent, state: "effect_started", effect_started_at: ctx.now().toISOString() };
    await writeAuthenticated(ctx, "intents", rollbackID, intent);
    let mutation;
    try { mutation = await provider(ctx).mutate(plan, receipt.rollback_patch.to, deadline); }
    catch (error) {
      if (error?.ambiguous) {
        await writeAuthenticated(ctx, "intents", rollbackID, { ...intent, state: "ambiguous", ambiguous_at: ctx.now().toISOString() });
        throw new ExecutorError("RECONCILIATION_REQUIRED", "rollback provider effect is ambiguous and must be reconciled", { ambiguous: true });
      }
      await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "failed", outputs_hash: null, provider_receipt: `executor:${rollbackID}:provider-rejected` }, deadline).catch(() => {});
      throw error;
    }
    ctx.fault("after_rollback_effect");
    const restored = await provider(ctx).read(plan, deadline);
    if (digest(restored) !== receipt.pre_image_hash) throw new ExecutorError("MANUAL_RECONCILIATION_REQUIRED", "rollback post-write verification failed", { ambiguous: true });
    const providerReceipt = `${mutation.requestID}:${mutation.responseHash}`.slice(0, 512);
    const rollbackReceipt = await terminalReceipt(ctx, { executionID: rollbackID, grant: input.grant, plan, pre: current, post: restored, providerReceipt, status: "rolled_back" });
    ctx.fault("after_rollback_receipt");
    await callControlPlane(ctx, "record_agent_action", { grant: input.grant, status: "executed", outputs_hash: rollbackReceipt.post_image_hash, provider_receipt: providerReceipt }, deadline);
    ctx.fault("after_rollback_terminal_action");
    await writeAuthenticated(ctx, "intents", rollbackID, { ...intent, state: "terminal", receipt_hash: digest(rollbackReceipt), completed_at: ctx.now().toISOString() });
    return publicReceipt(rollbackReceipt);
  });
}

function receiptPlan(receipt) {
  return {
    schema_version: PLAN_VERSION,
    plan_id: `rollback-${receipt.plan_id}`.slice(0, 128),
    action_id: "placeholder",
    provider: receipt.target.provider,
    account_id: receipt.target.account_id,
    resource_type: receipt.target.resource_type,
    resource_id: receipt.target.resource_id,
    generated_at: receipt.recorded_at,
    expires_at: new Date(new Date(receipt.recorded_at).valueOf() + 15 * 60_000).toISOString(),
  };
}

export async function getExecution(ctx, raw) {
  const input = plain(raw, "execution lookup");
  exact(input, ["execution_id"], "execution lookup");
  await ensureStore(ctx);
  const id = identifier(input.execution_id, "execution_id");
  const receipt = await readAuthenticated(ctx, "receipts", id, { required: false });
  const intent = await readAuthenticated(ctx, "intents", id, { required: false });
  if (receipt) {
    assertReceiptIntentBinding(receipt, intent);
    return publicReceipt(receipt);
  }
  if (!intent) throw new ExecutorError("EXECUTION_NOT_FOUND", "execution was not found");
  if (intent.state === "terminal") throw new ExecutorError("EXECUTION_STATE_INVALID", "terminal execution receipt is unavailable");
  return { execution_id: id, grant_id: intent.grant_id, plan_id: intent.plan.plan_id, target: targetFor(intent.plan), status: intent.state };
}

export function publicError(error) {
  if (error instanceof ExecutorError) return { code: error.code, message: error.message, retryable: error.retryable, ambiguous: error.ambiguous };
  return { code: "INTERNAL_ERROR", message: "ad-platform executor operation failed", retryable: false, ambiguous: false };
}

export function assertWireBudget(value) {
  if (Buffer.byteLength(JSON.stringify(value), "utf8") > MAX_MCP_BYTES) throw new ExecutorError("RESULT_TOO_LARGE", "MCP response exceeds the aggregate size limit");
  return value;
}
