import { createHash, createHmac, createPrivateKey, createPublicKey, sign as cryptoSign, timingSafeEqual, verify as cryptoVerify } from "node:crypto";
import { constants as fsConstants } from "node:fs";
import { access, link, lstat, mkdir, open, readFile, readdir, rename, rmdir, stat, unlink } from "node:fs/promises";
import path from "node:path";

const ID = /^[a-zA-Z0-9][a-zA-Z0-9._:-]{1,127}$/;
const SHA256 = /^sha256:[a-f0-9]{64}$/;
const STATUS = new Set(["executing", "executed", "failed", "cancelled"]);
const EFFECTS = new Set(["allow", "require_approval", "deny"]);
const RISKS = new Set(["low", "medium", "high"]);
const MAX_FILE_BYTES = 1024 * 1024;
const LOCK_STALE_MS = 30_000;
const DOMAIN_KINDS = new Set(["policy-activations", "decisions", "approval-requests", "approval-decisions", "grants", "execution-claims", "actions"]);

export class ControlPlaneError extends Error {
  constructor(code, message, { retryable = false } = {}) {
    super(message);
    this.name = "ControlPlaneError";
    this.code = code;
    this.retryable = retryable;
  }
}

function plain(value, label) {
  if (!value || typeof value !== "object" || Array.isArray(value) || Object.getPrototypeOf(value) !== Object.prototype) {
    throw new ControlPlaneError("INVALID_ARGUMENT", `${label} must be an object`);
  }
  return value;
}

function exact(value, keys, label) {
  const allowed = new Set(keys);
  for (const key of Object.keys(value)) {
    if (!allowed.has(key)) throw new ControlPlaneError("INVALID_ARGUMENT", `${label} contains unsupported fields`);
  }
}

function identifier(value, label) {
  if (typeof value !== "string" || !ID.test(value)) throw new ControlPlaneError("INVALID_ARGUMENT", `${label} is invalid`);
  return value;
}

function timestamp(value, label) {
  if (typeof value !== "string") throw new ControlPlaneError("INVALID_ARGUMENT", `${label} must be an RFC3339 timestamp`);
  const parsed = new Date(value);
  if (Number.isNaN(parsed.valueOf()) || parsed.toISOString() !== value) throw new ControlPlaneError("INVALID_ARGUMENT", `${label} must be a canonical RFC3339 timestamp`);
  return value;
}

function stringList(value, label, { minimum = 1, maximum = 32 } = {}) {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum) throw new ControlPlaneError("INVALID_ARGUMENT", `${label} has an invalid number of values`);
  const normalized = value.map((item) => identifier(item, label));
  if (new Set(normalized).size !== normalized.length) throw new ControlPlaneError("INVALID_ARGUMENT", `${label} contains duplicates`);
  return normalized;
}

export function canonicalJSON(value) {
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
  if (value && typeof value === "object") {
    return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
  }
  return JSON.stringify(value);
}

export function digest(value) {
  return `sha256:${createHash("sha256").update(typeof value === "string" ? value : canonicalJSON(value)).digest("hex")}`;
}

function hmac(value, key) {
  return createHmac("sha256", key).update(canonicalJSON(value)).digest("hex");
}

function safeSignature(expected, actual) {
  if (typeof actual !== "string" || !/^[a-f0-9]{64}$/.test(actual)) return false;
  return timingSafeEqual(Buffer.from(expected, "hex"), Buffer.from(actual, "hex"));
}

function signAsymmetric(value, privateKey) {
  try { return cryptoSign(null, Buffer.from(canonicalJSON(value)), privateKey).toString("base64"); }
  catch { throw new ControlPlaneError("CONFIGURATION_ERROR", "private signing key is unavailable or invalid"); }
}

function verifyAsymmetric(value, signature, publicKey) {
  if (typeof signature !== "string" || signature.length > 512 || !/^[A-Za-z0-9+/]+={0,2}$/.test(signature)) return false;
  try { return cryptoVerify(null, Buffer.from(canonicalJSON(value)), publicKey, Buffer.from(signature, "base64")); }
  catch { throw new ControlPlaneError("CONFIGURATION_ERROR", "public verification key is unavailable or invalid"); }
}

export function signIdentity(claims, privateKey) {
  const normalized = validateIdentityClaims(claims);
  return { ...normalized, signature: signAsymmetric(normalized, privateKey) };
}

function validateScope(scope, label = "scope") {
  plain(scope, label);
  exact(scope, ["provider", "account_id"], label);
  return { provider: identifier(scope.provider, `${label}.provider`), account_id: identifier(scope.account_id, `${label}.account_id`) };
}

function validateIdentityClaims(raw) {
  const value = plain(raw, "identity");
  exact(value, ["schema_version", "actor_id", "tenant_id", "roles", "capabilities", "scopes", "issued_at", "expires_at", "key_id"], "identity");
  if (value.schema_version !== "control-plane.identity/v1") throw new ControlPlaneError("AUTH_INVALID", "identity schema version is unsupported");
  const scopes = value.scopes.map((scope, index) => validateScope(scope, `identity.scopes[${index}]`));
  if (scopes.length < 1 || scopes.length > 32 || new Set(scopes.map(canonicalJSON)).size !== scopes.length) throw new ControlPlaneError("AUTH_INVALID", "identity scopes are invalid");
  return {
    schema_version: value.schema_version,
    actor_id: identifier(value.actor_id, "identity.actor_id"),
    tenant_id: identifier(value.tenant_id, "identity.tenant_id"),
    roles: stringList(value.roles, "identity.roles"),
    capabilities: stringList(value.capabilities, "identity.capabilities"),
    scopes,
    issued_at: timestamp(value.issued_at, "identity.issued_at"),
    expires_at: timestamp(value.expires_at, "identity.expires_at"),
    key_id: identifier(value.key_id, "identity.key_id"),
  };
}

export function authenticateIdentity(raw, publicKey, { now = () => new Date(), requiredRole } = {}) {
  if (typeof raw !== "string" || raw.length > 64 * 1024) throw new ControlPlaneError("AUTH_INVALID", "runtime identity is unavailable");
  let parsed;
  try { parsed = JSON.parse(raw); } catch { throw new ControlPlaneError("AUTH_INVALID", "runtime identity is malformed"); }
  plain(parsed, "identity envelope");
  exact(parsed, ["schema_version", "actor_id", "tenant_id", "roles", "capabilities", "scopes", "issued_at", "expires_at", "key_id", "signature"], "identity envelope");
  const { signature, ...claims } = parsed;
  const normalized = validateIdentityClaims(claims);
  if (!verifyAsymmetric(normalized, signature, publicKey)) throw new ControlPlaneError("AUTH_INVALID", "runtime identity signature is invalid");
  const current = now();
  if (new Date(normalized.issued_at) > current || new Date(normalized.expires_at) <= current) throw new ControlPlaneError("AUTH_EXPIRED", "runtime identity is not currently valid");
  if (requiredRole && !normalized.roles.includes(requiredRole)) throw new ControlPlaneError("AUTH_FORBIDDEN", "runtime identity lacks the required authority");
  return Object.freeze(normalized);
}

export function contextFromEnv(env = process.env, options = {}) {
  const storeRoot = env.CONTROL_PLANE_STORE;
  if (typeof storeRoot !== "string" || !path.isAbsolute(storeRoot)) throw new ControlPlaneError("CONFIGURATION_ERROR", "CONTROL_PLANE_STORE must be an absolute path");
  const admin = options.admin === true;
  const identity = authenticateIdentity(
    admin ? env.CONTROL_PLANE_ADMIN_IDENTITY : env.CONTROL_PLANE_IDENTITY,
    env.CONTROL_PLANE_IDENTITY_PUBLIC_KEY,
    { now: options.now },
  );
  if (admin && !identity.roles.some((role) => role === "policy_admin" || role === "approver")) throw new ControlPlaneError("AUTH_FORBIDDEN", "administrator identity has no control-plane authority");
  if (!admin && !identity.roles.some((role) => role === "agent_runtime" || role === "executor_runtime")) throw new ControlPlaneError("AUTH_FORBIDDEN", "runtime identity has no control-plane runtime role");
  const auditKey = env.CONTROL_PLANE_AUDIT_KEY;
  if (typeof auditKey !== "string" || auditKey.length < 32) throw new ControlPlaneError("CONFIGURATION_ERROR", "CONTROL_PLANE_AUDIT_KEY is unavailable");
  const grantRoleKey = env.CONTROL_PLANE_GRANT_ROLE_KEY;
  let grantPrivateKey;
  let grantPublicKey;
  const isAgent = identity.roles.includes("agent_runtime");
  const isExecutor = identity.roles.includes("executor_runtime");
  if (!admin && isAgent === isExecutor) throw new ControlPlaneError("CONFIGURATION_ERROR", "runtime identity must have exactly one runtime role");
  if (isAgent) {
    try {
      grantPrivateKey = createPrivateKey(grantRoleKey);
      grantPublicKey = createPublicKey(grantPrivateKey);
    } catch { throw new ControlPlaneError("CONFIGURATION_ERROR", "agent grant-role binding must contain the issuer private key"); }
  }
  if (isExecutor) {
    try { createPrivateKey(grantRoleKey); } catch {
      try { grantPublicKey = createPublicKey(grantRoleKey); } catch { throw new ControlPlaneError("CONFIGURATION_ERROR", "executor grant-role binding must contain the grant public key"); }
    }
    if (!grantPublicKey) throw new ControlPlaneError("CONFIGURATION_ERROR", "executor grant-role binding must not contain a private key");
  }
  return { storeRoot, identity, auditKey, grantPrivateKey, grantPublicKey, now: options.now ?? (() => new Date()), fault: options.fault };
}

export function storageSegment(value) {
  return Buffer.from(identifier(value, "storage identifier"), "utf8").toString("base64url");
}

async function exists(file) {
  try { await access(file, fsConstants.F_OK); return true; } catch { return false; }
}

async function readJSON(file) {
  const info = await lstat(file);
  if (!info.isFile() || info.size > MAX_FILE_BYTES) throw new ControlPlaneError("STORE_CORRUPT", "control-plane record is invalid");
  try { return JSON.parse(await readFile(file, "utf8")); } catch { throw new ControlPlaneError("STORE_CORRUPT", "control-plane record cannot be decoded"); }
}

async function syncDirectory(directory) {
  const handle = await open(directory, "r");
  try { await handle.sync(); } finally { await handle.close(); }
}

async function atomicCreateJSON(file, value) {
  await mkdir(path.dirname(file), { recursive: true, mode: 0o700 });
  const temp = path.join(path.dirname(file), `.tmp-${process.pid}-${Date.now()}-${Math.random().toString(16).slice(2)}`);
  const handle = await open(temp, "wx", 0o600);
  try {
    await handle.writeFile(`${canonicalJSON(value)}\n`, "utf8");
    await handle.sync();
  } finally { await handle.close(); }
  try {
    await link(temp, file);
    await syncDirectory(path.dirname(file));
  } catch (error) {
    if (error?.code !== "EEXIST") throw error;
    const current = await readJSON(file);
    if (canonicalJSON(current) !== canonicalJSON(value)) throw new ControlPlaneError("IDEMPOTENCY_CONFLICT", "an immutable record already exists with different content");
  } finally { await unlink(temp).catch(() => {}); }
}

async function atomicReplaceJSON(file, value) {
  await mkdir(path.dirname(file), { recursive: true, mode: 0o700 });
  const temp = path.join(path.dirname(file), `.tmp-${process.pid}-${Date.now()}-${Math.random().toString(16).slice(2)}`);
  const handle = await open(temp, "wx", 0o600);
  try {
    await handle.writeFile(`${canonicalJSON(value)}\n`, "utf8");
    await handle.sync();
  } finally { await handle.close(); }
  await rename(temp, file);
  await syncDirectory(path.dirname(file));
}

async function processAlive(pid) {
  if (!Number.isSafeInteger(pid) || pid < 1) return false;
  try { process.kill(pid, 0); return true; } catch (error) { return error?.code === "EPERM"; }
}

async function acquireLock(root, tenantID) {
  const locks = path.join(root, "locks");
  await mkdir(locks, { recursive: true, mode: 0o700 });
  const lockPath = path.join(locks, `${storageSegment(tenantID)}.lock`);
  for (let attempt = 0; attempt < 200; attempt += 1) {
    try {
      await mkdir(lockPath, { mode: 0o700 });
      await atomicCreateJSON(path.join(lockPath, "owner.json"), { pid: process.pid, acquired_at: new Date().toISOString() });
      return lockPath;
    } catch (error) {
      if (error?.code !== "EEXIST") throw error;
      const info = await stat(lockPath).catch(() => null);
      const owner = await readJSON(path.join(lockPath, "owner.json")).catch(() => ({}));
      const stale = info && Date.now() - info.mtimeMs > LOCK_STALE_MS && !(await processAlive(Number(owner.pid)));
      if (stale) {
        const stalePath = `${lockPath}.stale-${Date.now()}-${process.pid}`;
        await rename(lockPath, stalePath).catch(() => {});
        continue;
      }
      await new Promise((resolve) => setTimeout(resolve, 10));
    }
  }
  throw new ControlPlaneError("STORE_BUSY", "control-plane state is busy", { retryable: true });
}

async function releaseLock(lockPath) {
  await unlink(path.join(lockPath, "owner.json")).catch(() => {});
  await rmdir(lockPath).catch(() => {});
}

async function withTenantLock(ctx, fn, { recover = true } = {}) {
  await mkdir(ctx.storeRoot, { recursive: true, mode: 0o700 });
  const rootInfo = await lstat(ctx.storeRoot);
  if (!rootInfo.isDirectory() || (rootInfo.mode & 0o077) !== 0) throw new ControlPlaneError("CONFIGURATION_ERROR", "CONTROL_PLANE_STORE must be a private directory");
  const lock = await acquireLock(ctx.storeRoot, ctx.identity.tenant_id);
  try {
    ctx.auditRecords = await readRecords(tenantDir(ctx, "audit"));
    if (ctx.metrics) ctx.metrics.audit_directory_reads = (ctx.metrics.audit_directory_reads ?? 0) + 1;
    if (recover) {
      await recoverTransactionsLocked(ctx);
      const audit = await verifyAuditLocked(ctx);
      if (!audit.valid) throw new ControlPlaneError("STORE_CORRUPT", "tenant audit chain verification failed");
    }
    return await fn();
  } finally {
    delete ctx.auditRecords;
    await releaseLock(lock);
  }
}

function tenantDir(ctx, kind) {
  return path.join(ctx.storeRoot, kind, storageSegment(ctx.identity.tenant_id));
}

async function sortedJSONFiles(directory) {
  const names = await readdir(directory).catch((error) => error?.code === "ENOENT" ? [] : Promise.reject(error));
  return names.filter((name) => name.endsWith(".json") && !name.startsWith(".tmp-")).sort();
}

async function readRecords(directory) {
  const records = [];
  for (const name of await sortedJSONFiles(directory)) records.push(await readJSON(path.join(directory, name)));
  return records;
}

async function auditRecords(ctx) {
  if (!Array.isArray(ctx.auditRecords)) {
    ctx.auditRecords = await readRecords(tenantDir(ctx, "audit"));
    if (ctx.metrics) ctx.metrics.audit_directory_reads = (ctx.metrics.audit_directory_reads ?? 0) + 1;
  }
  return ctx.auditRecords;
}

async function readAuthenticatedDomainPath(ctx, relative) {
  validateDomainPath(ctx, relative);
  const record = await readJSON(path.join(ctx.storeRoot, relative));
  const recordDigest = digest(record);
  const events = await auditRecords(ctx);
  const event = events.find((candidate) => candidate.domain_path === relative);
  if (!event || event.domain_digest !== recordDigest) throw new ControlPlaneError("STORE_CORRUPT", "domain record is not bound to authenticated audit evidence");
  return record;
}

async function readAuthenticatedRecords(ctx, kind, suffix = []) {
  const relativeDirectory = path.join(kind, storageSegment(ctx.identity.tenant_id), ...suffix.map(storageSegment));
  const records = [];
  for (const name of await sortedJSONFiles(path.join(ctx.storeRoot, relativeDirectory))) records.push(await readAuthenticatedDomainPath(ctx, path.join(relativeDirectory, name)));
  return records;
}

function eventHash(event) {
  const { event_hash: ignored, ...material } = event;
  return digest(material);
}

function auditHead(ctx, event) {
  const material = { schema_version: "control-plane.audit-head/v1", tenant_id: ctx.identity.tenant_id, sequence: event.sequence, head_hash: event.event_hash };
  return { ...material, signature: hmac(material, ctx.auditKey) };
}

async function writeAuditHead(ctx, event) {
  await atomicReplaceJSON(path.join(ctx.storeRoot, "audit-heads", `${storageSegment(ctx.identity.tenant_id)}.json`), auditHead(ctx, event));
}

async function appendAuditLocked(ctx, base) {
  const directory = tenantDir(ctx, "audit");
  const records = await auditRecords(ctx);
  const existing = records.find((event) => event.transaction_id === base.transaction_id);
  if (existing) {
    const { schema_version: ignoredSchema, sequence: ignoredSequence, previous_event_hash: ignoredPrevious, event_hash: ignoredHash, ...storedBase } = existing;
    if (canonicalJSON(storedBase) !== canonicalJSON(base)) throw new ControlPlaneError("STORE_CORRUPT", "audit event conflicts with its authenticated transaction");
    await writeAuditHead(ctx, records.at(-1));
    return existing;
  }
  const sequence = records.length + 1;
  const previous = records.length ? records.at(-1).event_hash : null;
  const event = {
    schema_version: "control-plane.audit-event/v1",
    sequence,
    previous_event_hash: previous,
    ...base,
  };
  event.event_hash = eventHash(event);
  await atomicCreateJSON(path.join(directory, `${String(sequence).padStart(20, "0")}.json`), event);
  records.push(event);
  await writeAuditHead(ctx, event);
  return event;
}

function validateDomainPath(ctx, relative) {
  if (typeof relative !== "string" || relative.length < 1 || relative.length > 512 || path.isAbsolute(relative) || relative.includes("\\") || path.normalize(relative) !== relative) {
    throw new ControlPlaneError("STORE_CORRUPT", "transaction domain path is invalid");
  }
  const parts = relative.split(path.sep);
  if (!DOMAIN_KINDS.has(parts[0]) || parts[1] !== storageSegment(ctx.identity.tenant_id) || !parts.at(-1)?.endsWith(".json") || parts.some((part) => !part || part === "." || part === "..")) {
    throw new ControlPlaneError("STORE_CORRUPT", "transaction domain path escapes its tenant boundary");
  }
  if ((parts[0] === "actions" && parts.length !== 4) || (parts[0] !== "actions" && parts.length !== 3)) throw new ControlPlaneError("STORE_CORRUPT", "transaction domain path has an invalid shape");
  const absolute = path.resolve(ctx.storeRoot, relative);
  const root = path.resolve(ctx.storeRoot);
  if (!absolute.startsWith(`${root}${path.sep}`)) throw new ControlPlaneError("STORE_CORRUPT", "transaction domain path escapes the control-plane store");
  return relative;
}

function validateTransactionIntent(ctx, raw) {
  const value = plain(raw, "transaction intent");
  exact(value, ["schema_version", "tenant_id", "transaction_id", "domain", "audit", "signature"], "transaction intent");
  const { signature, ...material } = value;
  if (material.schema_version !== "control-plane.transaction-intent/v1" || material.tenant_id !== ctx.identity.tenant_id || !safeSignature(hmac(material, ctx.auditKey), signature)) {
    throw new ControlPlaneError("STORE_CORRUPT", "transaction intent authentication failed");
  }
  identifier(material.transaction_id, "transaction.transaction_id");
  const domain = plain(material.domain, "transaction.domain");
  exact(domain, ["path", "value"], "transaction.domain");
  validateDomainPath(ctx, domain.path);
  plain(domain.value, "transaction.domain.value");
  const audit = plain(material.audit, "transaction.audit");
  exact(audit, ["transaction_id", "event_id", "event_type", "tenant_id", "actor_id", "timestamp", "subject", "domain_path", "domain_digest"], "transaction.audit");
  if (audit.transaction_id !== material.transaction_id || audit.tenant_id !== material.tenant_id || audit.domain_path !== domain.path || audit.domain_digest !== digest(domain.value)) throw new ControlPlaneError("STORE_CORRUPT", "transaction intent relationships are invalid");
  identifier(audit.event_id, "transaction.audit.event_id");
  identifier(audit.event_type, "transaction.audit.event_type");
  identifier(audit.actor_id, "transaction.audit.actor_id");
  timestamp(audit.timestamp, "transaction.audit.timestamp");
  plain(audit.subject, "transaction.audit.subject");
  return value;
}

async function commitTransactionLocked(ctx, transaction) {
  transaction = validateTransactionIntent(ctx, transaction);
  const receiptPath = path.join(tenantDir(ctx, "transaction-receipts"), `${storageSegment(transaction.transaction_id)}.json`);
  if (await exists(receiptPath)) {
    const receipt = await readJSON(receiptPath);
    const { signature, ...material } = receipt;
    const records = await auditRecords(ctx);
    const event = records.find((candidate) => candidate.transaction_id === transaction.transaction_id);
    const persistedDomain = await readJSON(path.join(ctx.storeRoot, validateDomainPath(ctx, transaction.domain.path)));
    if (!event || material.schema_version !== "control-plane.transaction-receipt/v1" || material.tenant_id !== ctx.identity.tenant_id || material.transaction_id !== transaction.transaction_id || material.sequence !== event.sequence || material.event_hash !== event.event_hash || material.domain_path !== transaction.domain.path || material.domain_digest !== digest(transaction.domain.value) || material.domain_digest !== digest(persistedDomain) || !safeSignature(hmac(material, ctx.auditKey), signature)) {
      throw new ControlPlaneError("STORE_CORRUPT", "committed control-plane transaction is missing or altered");
    }
    await removeTransactionIntentLocked(ctx, transaction.transaction_id);
    return event;
  }
  if (ctx.fault) await ctx.fault("before_domain", transaction);
  if (transaction.domain) await atomicCreateJSON(path.join(ctx.storeRoot, transaction.domain.path), transaction.domain.value);
  if (ctx.fault) await ctx.fault("after_domain", transaction);
  const event = await appendAuditLocked(ctx, transaction.audit);
  if (ctx.fault) await ctx.fault("after_audit", transaction);
  const receiptMaterial = { schema_version: "control-plane.transaction-receipt/v1", tenant_id: ctx.identity.tenant_id, transaction_id: transaction.transaction_id, sequence: event.sequence, event_hash: event.event_hash, domain_path: transaction.domain.path, domain_digest: digest(transaction.domain.value) };
  await atomicCreateJSON(receiptPath, { ...receiptMaterial, signature: hmac(receiptMaterial, ctx.auditKey) });
  if (ctx.fault) await ctx.fault("after_receipt", transaction);
  await removeTransactionIntentLocked(ctx, transaction.transaction_id);
  return event;
}

async function removeTransactionIntentLocked(ctx, transactionID) {
  const directory = tenantDir(ctx, "transactions");
  const file = path.join(directory, `${storageSegment(transactionID)}.json`);
  try {
    await unlink(file);
    await syncDirectory(directory);
  } catch (error) {
    if (error?.code !== "ENOENT") throw error;
  }
}

async function startTransactionLocked(ctx, transaction) {
  plain(transaction, "transaction");
  exact(transaction, ["transaction_id", "domain", "audit"], "transaction");
  const domainPath = validateDomainPath(ctx, transaction.domain?.path);
  const audit = { ...transaction.audit, domain_path: domainPath, domain_digest: digest(transaction.domain?.value) };
  const material = { schema_version: "control-plane.transaction-intent/v1", tenant_id: ctx.identity.tenant_id, transaction_id: transaction.transaction_id, domain: transaction.domain, audit };
  const intent = validateTransactionIntent(ctx, { ...material, signature: hmac(material, ctx.auditKey) });
  const file = path.join(tenantDir(ctx, "transactions"), `${storageSegment(intent.transaction_id)}.json`);
  await atomicCreateJSON(file, intent);
  if (ctx.fault) await ctx.fault("after_intent", intent);
  return commitTransactionLocked(ctx, intent);
}

async function recoverTransactionsLocked(ctx) {
  for (const raw of await readRecords(tenantDir(ctx, "transactions"))) await commitTransactionLocked({ ...ctx, fault: undefined }, validateTransactionIntent(ctx, raw));
}

export async function recoverTransactions(ctx) {
  return withTenantLock({ ...ctx, fault: undefined }, async () => ({ recovered: true }));
}

function validateTarget(raw, label = "target") {
  const value = plain(raw, label);
  exact(value, ["provider", "account_id", "resource_type", "resource_id"], label);
  return {
    provider: identifier(value.provider, `${label}.provider`),
    account_id: identifier(value.account_id, `${label}.account_id`),
    resource_type: identifier(value.resource_type, `${label}.resource_type`),
    resource_id: identifier(value.resource_id, `${label}.resource_id`),
  };
}

function scopeAllows(identity, target) {
  return identity.scopes.some((scope) => scope.provider === target.provider && scope.account_id === target.account_id);
}

function requireRole(ctx, role) {
  if (!ctx.identity.roles.includes(role)) throw new ControlPlaneError("AUTH_FORBIDDEN", `operation requires ${role} authority`);
}

function requireExecutor(ctx) {
  requireRole(ctx, "executor_runtime");
  if (ctx.grantPrivateKey !== undefined) throw new ControlPlaneError("CONFIGURATION_ERROR", "executor context must not contain the execution-grant private key");
}

export function validateProposal(raw) {
  const value = plain(raw, "proposal");
  exact(value, ["action_id", "agent_run_id", "capability", "capability_version", "risk_tier", "target", "inputs_hash"], "proposal");
  if (!RISKS.has(value.risk_tier)) throw new ControlPlaneError("INVALID_ARGUMENT", "proposal.risk_tier is invalid");
  if (!SHA256.test(value.inputs_hash)) throw new ControlPlaneError("INVALID_ARGUMENT", "proposal.inputs_hash must be a SHA-256 digest");
  return {
    action_id: identifier(value.action_id, "proposal.action_id"),
    agent_run_id: identifier(value.agent_run_id, "proposal.agent_run_id"),
    capability: identifier(value.capability, "proposal.capability"),
    capability_version: identifier(value.capability_version, "proposal.capability_version"),
    risk_tier: value.risk_tier,
    target: validateTarget(value.target, "proposal.target"),
    inputs_hash: value.inputs_hash,
  };
}

export function validatePolicy(raw) {
  const value = plain(raw, "policy");
  exact(value, ["schema_version", "policy_id", "version", "tenant_id", "effective_at", "expires_at", "rules"], "policy");
  if (value.schema_version !== "control-plane.policy/v1") throw new ControlPlaneError("INVALID_ARGUMENT", "policy schema version is unsupported");
  if (!Array.isArray(value.rules) || value.rules.length < 1 || value.rules.length > 100) throw new ControlPlaneError("INVALID_ARGUMENT", "policy rules are invalid");
  const rules = value.rules.map((rawRule, index) => {
    const rule = plain(rawRule, `policy.rules[${index}]`);
    exact(rule, ["rule_id", "effect", "capabilities", "risk_tiers", "target", "approver_roles", "approval_ttl_seconds"], `policy.rules[${index}]`);
    if (!EFFECTS.has(rule.effect)) throw new ControlPlaneError("INVALID_ARGUMENT", "policy rule effect is invalid");
    const approverRoles = rule.effect === "require_approval" ? stringList(rule.approver_roles, "rule.approver_roles") : [];
    const ttl = rule.effect === "require_approval" ? Number(rule.approval_ttl_seconds) : 0;
    if (rule.effect === "require_approval" && (!Number.isSafeInteger(ttl) || ttl < 30 || ttl > 86_400)) throw new ControlPlaneError("INVALID_ARGUMENT", "approval TTL is invalid");
    return {
      rule_id: identifier(rule.rule_id, "rule.rule_id"),
      effect: rule.effect,
      capabilities: stringList(rule.capabilities, "rule.capabilities"),
      risk_tiers: stringList(rule.risk_tiers, "rule.risk_tiers").map((risk) => {
        if (!RISKS.has(risk)) throw new ControlPlaneError("INVALID_ARGUMENT", "rule risk tier is invalid");
        return risk;
      }),
      target: validateScope(rule.target, "rule.target"),
      approver_roles: approverRoles,
      approval_ttl_seconds: ttl,
    };
  });
  if (new Set(rules.map((rule) => rule.rule_id)).size !== rules.length) throw new ControlPlaneError("INVALID_ARGUMENT", "policy rule IDs must be unique");
  const effectiveAt = timestamp(value.effective_at, "policy.effective_at");
  const expiresAt = timestamp(value.expires_at, "policy.expires_at");
  if (new Date(expiresAt) <= new Date(effectiveAt)) throw new ControlPlaneError("INVALID_ARGUMENT", "policy expiry must follow its effective time");
  return {
    schema_version: value.schema_version,
    policy_id: identifier(value.policy_id, "policy.policy_id"),
    version: identifier(value.version, "policy.version"),
    tenant_id: identifier(value.tenant_id, "policy.tenant_id"),
    effective_at: effectiveAt,
    expires_at: expiresAt,
    rules,
  };
}

async function activePolicyLocked(ctx) {
  const auditEvents = await readRecords(tenantDir(ctx, "audit"));
  const activationEvents = auditEvents.filter((event) => event.event_type === "policy_activated");
  if (!activationEvents.length) throw new ControlPlaneError("POLICY_UNAVAILABLE", "no active policy is installed");
  const activation = activationEvents.at(-1).subject;
  const persisted = (await readAuthenticatedRecords(ctx, "policy-activations")).find((entry) => entry.activation_id === activation.activation_id);
  if (!persisted || canonicalJSON(persisted) !== canonicalJSON(activation)) throw new ControlPlaneError("STORE_CORRUPT", "active policy activation record is missing or altered");
  const policy = await readJSON(path.join(ctx.storeRoot, activation.policy_path));
  const now = ctx.now();
  if (new Date(policy.effective_at) > now || new Date(policy.expires_at) <= now) throw new ControlPlaneError("POLICY_EXPIRED", "the active policy is not currently valid");
  if (digest(policy) !== activation.policy_digest) throw new ControlPlaneError("STORE_CORRUPT", "active policy integrity check failed");
  return { policy, activation };
}

export async function installPolicy(ctx, rawPolicy) {
  if (!ctx.identity.roles.includes("policy_admin")) throw new ControlPlaneError("AUTH_FORBIDDEN", "policy administration requires policy_admin authority");
  const policy = validatePolicy(rawPolicy);
  if (policy.tenant_id !== ctx.identity.tenant_id) throw new ControlPlaneError("SCOPE_MISMATCH", "policy tenant does not match administrator tenant");
  return withTenantLock(ctx, async () => {
    const policyDigest = digest(policy);
    const installedPolicies = await readRecords(path.join(ctx.storeRoot, "policies", storageSegment(policy.tenant_id)));
    const sameVersion = installedPolicies.find((entry) => entry.policy_id === policy.policy_id && entry.version === policy.version);
    if (sameVersion && digest(sameVersion) !== policyDigest) throw new ControlPlaneError("POLICY_VERSION_CONFLICT", "policy ID and version already identify different immutable content");
    const relative = path.join("policies", storageSegment(policy.tenant_id), `${storageSegment(policy.policy_id)}-${storageSegment(policy.version)}-${policyDigest.slice(7, 23)}.json`);
    await atomicCreateJSON(path.join(ctx.storeRoot, relative), policy);
    const existing = (await readAuthenticatedRecords(ctx, "policy-activations")).find((entry) => entry.policy_digest === policyDigest);
    if (existing) return existing;
    const activationID = `pact_${policyDigest.slice(7, 31)}`;
    const activatedAt = ctx.now().toISOString();
    const activation = { activation_id: activationID, policy_path: relative, policy_digest: policyDigest, policy_id: policy.policy_id, policy_version: policy.version, activated_at: activatedAt, actor_id: ctx.identity.actor_id };
    const transactionID = `txn_${digest({ type: "policy_activation", activation }).slice(7, 39)}`;
    await startTransactionLocked(ctx, {
      transaction_id: transactionID,
      domain: { path: path.join("policy-activations", storageSegment(policy.tenant_id), `${String((await readAuthenticatedRecords(ctx, "policy-activations")).length + 1).padStart(20, "0")}.json`), value: activation },
      audit: { transaction_id: transactionID, event_id: `evt_${transactionID.slice(4)}`, event_type: "policy_activated", tenant_id: policy.tenant_id, actor_id: ctx.identity.actor_id, timestamp: activatedAt, subject: activation },
    });
    return activation;
  });
}

function matchingRules(policy, proposal) {
  return policy.rules.filter((rule) => rule.capabilities.includes(proposal.capability) && rule.target.provider === proposal.target.provider && rule.target.account_id === proposal.target.account_id);
}

function evaluate(policy, proposal) {
  const matches = matchingRules(policy, proposal);
  if (!matches.length) return { effect: "deny", effective_risk_tier: "high", matched_rule_ids: [], approver_roles: [], approval_ttl_seconds: 0, reason_code: "NO_MATCHING_RULE" };
  const riskOrder = { low: 0, medium: 1, high: 2 };
  const effectiveRiskTier = matches.flatMap((rule) => rule.risk_tiers).sort((left, right) => riskOrder[right] - riskOrder[left])[0];
  if (matches.some((rule) => rule.effect === "deny")) return { effect: "deny", effective_risk_tier: effectiveRiskTier, matched_rule_ids: matches.map((rule) => rule.rule_id).sort(), approver_roles: [], approval_ttl_seconds: 0, reason_code: "RULE_DENY" };
  const approvals = matches.filter((rule) => rule.effect === "require_approval");
  if (approvals.length) return { effect: "require_approval", effective_risk_tier: effectiveRiskTier, matched_rule_ids: matches.map((rule) => rule.rule_id).sort(), approver_roles: [...new Set(approvals.flatMap((rule) => rule.approver_roles))].sort(), approval_ttl_seconds: Math.min(...approvals.map((rule) => rule.approval_ttl_seconds)), reason_code: "RULE_REQUIRE_APPROVAL" };
  return { effect: "allow", effective_risk_tier: effectiveRiskTier, matched_rule_ids: matches.map((rule) => rule.rule_id).sort(), approver_roles: [], approval_ttl_seconds: 0, reason_code: "RULE_ALLOW" };
}

async function readDomain(ctx, kind, id) {
  return readAuthenticatedDomainPath(ctx, path.join(kind, storageSegment(ctx.identity.tenant_id), `${storageSegment(id)}.json`));
}

export async function checkPolicy(ctx, rawProposal) {
  requireRole(ctx, "agent_runtime");
  const proposal = validateProposal(rawProposal);
  if (!ctx.identity.capabilities.includes(proposal.capability) || !scopeAllows(ctx.identity, proposal.target)) throw new ControlPlaneError("SCOPE_MISMATCH", "runtime identity is not authorized for the proposed capability and target");
  return withTenantLock(ctx, async () => {
    const { policy, activation } = await activePolicyLocked(ctx);
    const result = evaluate(policy, proposal);
    const material = { tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, proposal, policy_digest: activation.policy_digest, result };
    const decisionID = `pdec_${digest(material).slice(7, 39)}`;
    const existingPath = path.join(tenantDir(ctx, "decisions"), `${storageSegment(decisionID)}.json`);
    if (await exists(existingPath)) return readDomain(ctx, "decisions", decisionID);
    const decidedAt = ctx.now().toISOString();
    const decision = { schema_version: "control-plane.policy-decision/v1", decision_id: decisionID, tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, proposal, policy_id: policy.policy_id, policy_version: policy.version, policy_digest: activation.policy_digest, ...result, decided_at: decidedAt };
    const transactionID = `txn_${decisionID.slice(5)}`;
    await startTransactionLocked(ctx, {
      transaction_id: transactionID,
      domain: { path: path.join("decisions", storageSegment(ctx.identity.tenant_id), `${storageSegment(decisionID)}.json`), value: decision },
      audit: { transaction_id: transactionID, event_id: `evt_${transactionID.slice(4)}`, event_type: "policy_decided", tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, timestamp: decidedAt, subject: decision },
    });
    return decision;
  });
}

function assertDecisionOwner(ctx, decision) {
  if (decision.tenant_id !== ctx.identity.tenant_id || decision.actor_id !== ctx.identity.actor_id) throw new ControlPlaneError("SCOPE_MISMATCH", "policy decision belongs to a different runtime identity");
  if (!ctx.identity.capabilities.includes(decision.proposal.capability) || !scopeAllows(ctx.identity, decision.proposal.target)) throw new ControlPlaneError("SCOPE_MISMATCH", "current runtime identity no longer covers the policy decision");
}

export async function requestApproval(ctx, input) {
  requireRole(ctx, "agent_runtime");
  const value = plain(input, "approval request");
  exact(value, ["decision_id", "justification"], "approval request");
  const decisionID = identifier(value.decision_id, "decision_id");
  if (typeof value.justification !== "string" || value.justification.length < 1 || value.justification.length > 2_000) throw new ControlPlaneError("INVALID_ARGUMENT", "approval justification is invalid");
  return withTenantLock(ctx, async () => {
    const decision = await readDomain(ctx, "decisions", decisionID);
    assertDecisionOwner(ctx, decision);
    if (decision.effect !== "require_approval") throw new ControlPlaneError("APPROVAL_NOT_REQUIRED", "the policy decision does not require approval");
    const active = await activePolicyLocked(ctx);
    if (active.activation.policy_digest !== decision.policy_digest) throw new ControlPlaneError("POLICY_CHANGED", "policy changed after the decision was recorded");
    const justificationHash = digest(value.justification);
    const approvalID = `apr_${digest({ decision_id: decisionID, justification_hash: justificationHash }).slice(7, 39)}`;
    const existing = path.join(tenantDir(ctx, "approval-requests"), `${storageSegment(approvalID)}.json`);
    if (await exists(existing)) return readDomain(ctx, "approval-requests", approvalID);
    const requestedAt = ctx.now();
    const expiresAt = new Date(Math.min(requestedAt.valueOf() + decision.approval_ttl_seconds * 1000, new Date(active.policy.expires_at).valueOf()));
    if (expiresAt <= requestedAt) throw new ControlPlaneError("POLICY_EXPIRED", "policy expires before approval can be issued");
    const request = { schema_version: "control-plane.approval-request/v1", approval_id: approvalID, decision_id: decisionID, tenant_id: ctx.identity.tenant_id, requester_id: ctx.identity.actor_id, proposal: decision.proposal, policy_id: decision.policy_id, policy_version: decision.policy_version, policy_digest: decision.policy_digest, required_roles: decision.approver_roles, justification_hash: justificationHash, requested_at: requestedAt.toISOString(), expires_at: expiresAt.toISOString() };
    const transactionID = `txn_${approvalID.slice(4)}`;
    await startTransactionLocked(ctx, {
      transaction_id: transactionID,
      domain: { path: path.join("approval-requests", storageSegment(ctx.identity.tenant_id), `${storageSegment(approvalID)}.json`), value: request },
      audit: { transaction_id: transactionID, event_id: `evt_${transactionID.slice(4)}`, event_type: "approval_requested", tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, timestamp: request.requested_at, subject: request },
    });
    return request;
  });
}

export async function decideApproval(ctx, raw) {
  const value = plain(raw, "approval decision");
  exact(value, ["approval_id", "decision", "reason"], "approval decision");
  const approvalID = identifier(value.approval_id, "approval_id");
  if (!["approved", "rejected"].includes(value.decision) || typeof value.reason !== "string" || value.reason.length < 1 || value.reason.length > 2_000) throw new ControlPlaneError("INVALID_ARGUMENT", "approval decision is invalid");
  return withTenantLock(ctx, async () => {
    const request = await readDomain(ctx, "approval-requests", approvalID);
    if (request.tenant_id !== ctx.identity.tenant_id || !request.required_roles.every((role) => ctx.identity.roles.includes(role))) throw new ControlPlaneError("AUTH_FORBIDDEN", "administrator is not an authorized approver for this request");
    if (new Date(request.expires_at) <= ctx.now()) throw new ControlPlaneError("APPROVAL_EXPIRED", "approval request has expired");
    const existingPath = path.join(tenantDir(ctx, "approval-decisions"), `${storageSegment(approvalID)}.json`);
    const decidedAt = ctx.now().toISOString();
    const decision = { schema_version: "control-plane.approval-decision/v1", approval_id: approvalID, tenant_id: ctx.identity.tenant_id, decision: value.decision, reason_hash: digest(value.reason), approver_id: ctx.identity.actor_id, approver_roles: ctx.identity.roles.filter((role) => request.required_roles.includes(role)), decided_at: decidedAt, expires_at: request.expires_at, request_digest: digest(request) };
    if (await exists(existingPath)) {
      const existing = await readDomain(ctx, "approval-decisions", approvalID);
      if (existing.decision !== decision.decision || existing.approver_id !== decision.approver_id || existing.reason_hash !== decision.reason_hash) throw new ControlPlaneError("APPROVAL_ALREADY_DECIDED", "approval already has an immutable decision");
      return existing;
    }
    const transactionID = `txn_${digest({ type: "approval_decision", decision }).slice(7, 39)}`;
    await startTransactionLocked(ctx, {
      transaction_id: transactionID,
      domain: { path: path.join("approval-decisions", storageSegment(ctx.identity.tenant_id), `${storageSegment(approvalID)}.json`), value: decision },
      audit: { transaction_id: transactionID, event_id: `evt_${transactionID.slice(4)}`, event_type: `approval_${value.decision}`, tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, timestamp: decidedAt, subject: decision },
    });
    return decision;
  });
}

export async function getApproval(ctx, input) {
  requireRole(ctx, "agent_runtime");
  const value = plain(input, "approval lookup");
  exact(value, ["approval_id"], "approval lookup");
  return withTenantLock(ctx, async () => {
    const request = await readDomain(ctx, "approval-requests", identifier(value.approval_id, "approval_id"));
    if (request.tenant_id !== ctx.identity.tenant_id || request.requester_id !== ctx.identity.actor_id) throw new ControlPlaneError("SCOPE_MISMATCH", "approval request belongs to another runtime identity");
    const decisionPath = path.join(tenantDir(ctx, "approval-decisions"), `${storageSegment(request.approval_id)}.json`);
    return { request, decision: await exists(decisionPath) ? await readDomain(ctx, "approval-decisions", request.approval_id) : null };
  });
}

function grantMaterial(grant) {
  const { signature: ignored, ...material } = grant;
  return material;
}

export function verifyGrant(grant, publicKey, { now = () => new Date(), requireCurrent = true } = {}) {
  const value = plain(grant, "execution grant");
  exact(value, ["schema_version", "grant_id", "tenant_id", "actor_id", "decision_id", "approval_id", "policy_id", "policy_version", "policy_digest", "action_id", "agent_run_id", "capability", "capability_version", "risk_tier", "target", "inputs_hash", "issued_at", "expires_at", "signature"], "execution grant");
  if (value.schema_version !== "control-plane.execution-grant/v1" || !verifyAsymmetric(grantMaterial(value), value.signature, publicKey)) throw new ControlPlaneError("GRANT_INVALID", "execution grant signature is invalid");
  const current = now();
  if (new Date(timestamp(value.issued_at, "grant.issued_at")) > current) throw new ControlPlaneError("GRANT_INVALID", "execution grant is not yet valid");
  if (requireCurrent && new Date(timestamp(value.expires_at, "grant.expires_at")) <= current) throw new ControlPlaneError("GRANT_EXPIRED", "execution grant is expired");
  return value;
}

export async function authorizeAction(ctx, input) {
  requireRole(ctx, "agent_runtime");
  const value = plain(input, "authorization request");
  exact(value, ["decision_id", "approval_id"], "authorization request");
  const decisionID = identifier(value.decision_id, "decision_id");
  return withTenantLock(ctx, async () => {
    const decision = await readDomain(ctx, "decisions", decisionID);
    assertDecisionOwner(ctx, decision);
    if (decision.effect === "deny") throw new ControlPlaneError("POLICY_DENIED", "policy denied the proposed action");
    const active = await activePolicyLocked(ctx);
    if (active.activation.policy_digest !== decision.policy_digest) throw new ControlPlaneError("POLICY_CHANGED", "policy changed after the decision was recorded");
    let approval = null;
    if (decision.effect === "require_approval") {
      const approvalID = identifier(value.approval_id, "approval_id");
      const request = await readDomain(ctx, "approval-requests", approvalID);
      approval = await readDomain(ctx, "approval-decisions", approvalID);
      if (request.decision_id !== decisionID || approval.request_digest !== digest(request) || approval.decision !== "approved") throw new ControlPlaneError("APPROVAL_INVALID", "approval does not authorize this policy decision");
      if (new Date(approval.expires_at) <= ctx.now()) throw new ControlPlaneError("APPROVAL_EXPIRED", "approval is expired");
    } else if (value.approval_id !== undefined) throw new ControlPlaneError("INVALID_ARGUMENT", "approval_id is not valid for this policy decision");
    const grantID = `grant_${digest({ decision_id: decisionID, approval_id: approval?.approval_id ?? null }).slice(7, 39)}`;
    const grantPath = path.join(tenantDir(ctx, "grants"), `${storageSegment(grantID)}.json`);
    if (await exists(grantPath)) {
      const existing = await readDomain(ctx, "grants", grantID);
      verifyGrant(existing, ctx.grantPublicKey, { now: ctx.now });
      const actions = await readAuthenticatedRecords(ctx, "actions", [grantID]);
      if (actions.some((record) => ["executed", "failed", "cancelled"].includes(record.status))) throw new ControlPlaneError("ACTION_ALREADY_RECORDED", "the authorized action already has a terminal record");
      return existing;
    }
    const issuedAt = ctx.now();
    const expiresAt = new Date(Math.min(issuedAt.valueOf() + 60_000, new Date(active.policy.expires_at).valueOf(), approval ? new Date(approval.expires_at).valueOf() : Number.POSITIVE_INFINITY));
    if (expiresAt <= issuedAt) throw new ControlPlaneError("AUTHORIZATION_EXPIRED", "authorization authority has expired");
    const proposal = decision.proposal;
    const material = { schema_version: "control-plane.execution-grant/v1", grant_id: grantID, tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, decision_id: decisionID, approval_id: approval?.approval_id ?? null, policy_id: decision.policy_id, policy_version: decision.policy_version, policy_digest: decision.policy_digest, ...proposal, risk_tier: decision.effective_risk_tier, issued_at: issuedAt.toISOString(), expires_at: expiresAt.toISOString() };
    const grant = { ...material, signature: signAsymmetric(material, ctx.grantPrivateKey) };
    const transactionID = `txn_${grantID.slice(6)}`;
    await startTransactionLocked(ctx, {
      transaction_id: transactionID,
      domain: { path: path.join("grants", storageSegment(ctx.identity.tenant_id), `${storageSegment(grantID)}.json`), value: grant },
      audit: { transaction_id: transactionID, event_id: `evt_${transactionID.slice(4)}`, event_type: "grant_issued", tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, timestamp: grant.issued_at, subject: { grant_id: grant.grant_id, tenant_id: grant.tenant_id, actor_id: grant.actor_id, decision_id: grant.decision_id, approval_id: grant.approval_id, policy_digest: grant.policy_digest, action_id: grant.action_id, capability: grant.capability, target: grant.target, inputs_hash: grant.inputs_hash, expires_at: grant.expires_at } },
    });
    return grant;
  });
}

export async function validateExecutionGrant(ctx, input) {
  requireExecutor(ctx);
  const value = plain(input, "grant validation request");
  exact(value, ["grant"], "grant validation request");
  const grant = verifyGrant(value.grant, ctx.grantPublicKey, { now: ctx.now });
  if (grant.tenant_id !== ctx.identity.tenant_id || !ctx.identity.capabilities.includes(grant.capability) || !scopeAllows(ctx.identity, grant.target)) throw new ControlPlaneError("SCOPE_MISMATCH", "executor identity does not cover the execution grant");
  return withTenantLock(ctx, async () => {
    const persisted = await readDomain(ctx, "grants", grant.grant_id);
    if (canonicalJSON(persisted) !== canonicalJSON(grant)) throw new ControlPlaneError("GRANT_INVALID", "execution grant does not match the authoritative record");
    const active = await activePolicyLocked(ctx);
    if (active.activation.policy_digest !== grant.policy_digest) throw new ControlPlaneError("POLICY_CHANGED", "policy changed after the execution grant was issued");
    if (grant.approval_id !== null) {
      const approval = await readDomain(ctx, "approval-decisions", grant.approval_id);
      if (approval.decision !== "approved" || new Date(approval.expires_at) <= ctx.now()) throw new ControlPlaneError("APPROVAL_EXPIRED", "execution approval is no longer valid");
    }
    const actions = await readAuthenticatedRecords(ctx, "actions", [grant.grant_id]);
    if (actions.some((record) => ["executed", "failed", "cancelled"].includes(record.status))) throw new ControlPlaneError("ACTION_ALREADY_RECORDED", "the execution grant already has a terminal action record");
    const claimID = `claim_${digest({ grant_id: grant.grant_id }).slice(7, 39)}`;
    const claimPath = path.join(tenantDir(ctx, "execution-claims"), `${storageSegment(claimID)}.json`);
    if (await exists(claimPath)) throw new ControlPlaneError("ACTION_ALREADY_CLAIMED", "the execution grant already has a durable effect-boundary claim");
    const claimedAt = ctx.now().toISOString();
    const claim = { schema_version: "control-plane.execution-claim/v1", claim_id: claimID, grant_id: grant.grant_id, tenant_id: grant.tenant_id, actor_id: grant.actor_id, executor_id: ctx.identity.actor_id, decision_id: grant.decision_id, policy_digest: grant.policy_digest, target: grant.target, inputs_hash: grant.inputs_hash, claimed_at: claimedAt };
    const transactionID = `txn_${claimID.slice(6)}`;
    await startTransactionLocked(ctx, {
      transaction_id: transactionID,
      domain: { path: path.join("execution-claims", storageSegment(ctx.identity.tenant_id), `${storageSegment(claimID)}.json`), value: claim },
      audit: { transaction_id: transactionID, event_id: `evt_${transactionID.slice(4)}`, event_type: "execution_claimed", tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, timestamp: claimedAt, subject: claim },
    });
    return { valid: true, claim_id: claimID, grant_id: grant.grant_id, tenant_id: grant.tenant_id, actor_id: grant.actor_id, executor_id: ctx.identity.actor_id, policy_digest: grant.policy_digest, approval_id: grant.approval_id, target: grant.target, inputs_hash: grant.inputs_hash, expires_at: grant.expires_at };
  });
}

export async function getExecutionClaim(ctx, input) {
  requireExecutor(ctx);
  const value = plain(input, "execution claim lookup");
  exact(value, ["grant"], "execution claim lookup");
  const grant = verifyGrant(value.grant, ctx.grantPublicKey, { now: ctx.now, requireCurrent: false });
  if (grant.tenant_id !== ctx.identity.tenant_id || !ctx.identity.capabilities.includes(grant.capability) || !scopeAllows(ctx.identity, grant.target)) throw new ControlPlaneError("SCOPE_MISMATCH", "executor identity does not cover the execution grant");
  return withTenantLock(ctx, async () => {
    const persisted = await readDomain(ctx, "grants", grant.grant_id);
    if (canonicalJSON(persisted) !== canonicalJSON(grant)) throw new ControlPlaneError("GRANT_INVALID", "execution grant does not match the authoritative record");
    const claimID = `claim_${digest({ grant_id: grant.grant_id }).slice(7, 39)}`;
    const claim = await readDomain(ctx, "execution-claims", claimID);
    if (claim.grant_id !== grant.grant_id || claim.tenant_id !== grant.tenant_id || claim.actor_id !== grant.actor_id || claim.executor_id !== ctx.identity.actor_id || claim.decision_id !== grant.decision_id || claim.policy_digest !== grant.policy_digest || canonicalJSON(claim.target) !== canonicalJSON(grant.target) || claim.inputs_hash !== grant.inputs_hash) throw new ControlPlaneError("SCOPE_MISMATCH", "execution claim does not bind the exact grant and executor");
    const actions = await readAuthenticatedRecords(ctx, "actions", [grant.grant_id]);
    if (actions.some((record) => ["executed", "failed", "cancelled"].includes(record.status))) throw new ControlPlaneError("ACTION_ALREADY_RECORDED", "the execution grant already has a terminal action record");
    return { valid: true, claim_id: claim.claim_id, grant_id: grant.grant_id, tenant_id: grant.tenant_id, actor_id: grant.actor_id, executor_id: ctx.identity.actor_id, policy_digest: grant.policy_digest, approval_id: grant.approval_id, target: grant.target, inputs_hash: grant.inputs_hash, expires_at: grant.expires_at, recovered: true };
  });
}

export async function revalidateExecutionClaim(ctx, input) {
  requireExecutor(ctx);
  const value = plain(input, "execution claim revalidation");
  exact(value, ["grant"], "execution claim revalidation");
  const grant = verifyGrant(value.grant, ctx.grantPublicKey, { now: ctx.now });
  if (grant.tenant_id !== ctx.identity.tenant_id || !ctx.identity.capabilities.includes(grant.capability) || !scopeAllows(ctx.identity, grant.target)) throw new ControlPlaneError("SCOPE_MISMATCH", "executor identity does not cover the execution grant");
  return withTenantLock(ctx, async () => {
    const persisted = await readDomain(ctx, "grants", grant.grant_id);
    if (canonicalJSON(persisted) !== canonicalJSON(grant)) throw new ControlPlaneError("GRANT_INVALID", "execution grant does not match the authoritative record");
    const claimID = `claim_${digest({ grant_id: grant.grant_id }).slice(7, 39)}`;
    const claim = await readDomain(ctx, "execution-claims", claimID);
    if (claim.grant_id !== grant.grant_id || claim.tenant_id !== grant.tenant_id || claim.actor_id !== grant.actor_id || claim.executor_id !== ctx.identity.actor_id || claim.decision_id !== grant.decision_id || claim.policy_digest !== grant.policy_digest || canonicalJSON(claim.target) !== canonicalJSON(grant.target) || claim.inputs_hash !== grant.inputs_hash) throw new ControlPlaneError("SCOPE_MISMATCH", "execution claim does not bind the exact grant and executor");
    const actions = await readAuthenticatedRecords(ctx, "actions", [grant.grant_id]);
    if (actions.some((record) => ["executed", "failed", "cancelled"].includes(record.status))) throw new ControlPlaneError("ACTION_ALREADY_RECORDED", "the execution grant already has a terminal action record");
    const active = await activePolicyLocked(ctx);
    if (active.activation.policy_digest !== grant.policy_digest) throw new ControlPlaneError("POLICY_CHANGED", "policy changed after the execution grant was claimed");
    if (grant.approval_id !== null) {
      const approval = await readDomain(ctx, "approval-decisions", grant.approval_id);
      if (approval.decision !== "approved" || new Date(approval.expires_at) <= ctx.now()) throw new ControlPlaneError("APPROVAL_EXPIRED", "execution approval is no longer valid");
    }
    return { valid: true, current: true, claim_id: claim.claim_id, grant_id: grant.grant_id, tenant_id: grant.tenant_id, actor_id: grant.actor_id, executor_id: ctx.identity.actor_id, policy_digest: grant.policy_digest, approval_id: grant.approval_id, target: grant.target, inputs_hash: grant.inputs_hash, expires_at: grant.expires_at };
  });
}

export async function recordAction(ctx, input) {
  requireExecutor(ctx);
  const value = plain(input, "action record");
  exact(value, ["grant", "status", "outputs_hash", "provider_receipt"], "action record");
  if (!STATUS.has(value.status)) throw new ControlPlaneError("INVALID_ARGUMENT", "action status is invalid");
  if (value.outputs_hash !== null && !SHA256.test(value.outputs_hash)) throw new ControlPlaneError("INVALID_ARGUMENT", "outputs_hash must be null or a SHA-256 digest");
  if (typeof value.provider_receipt !== "string" || value.provider_receipt.length < 1 || value.provider_receipt.length > 512) throw new ControlPlaneError("INVALID_ARGUMENT", "provider receipt reference is invalid");
  const providerReceiptHash = digest(value.provider_receipt);
  const grant = verifyGrant(value.grant, ctx.grantPublicKey, { now: ctx.now, requireCurrent: value.status === "executing" });
  if (grant.tenant_id !== ctx.identity.tenant_id || !ctx.identity.capabilities.includes(grant.capability) || !scopeAllows(ctx.identity, grant.target)) throw new ControlPlaneError("SCOPE_MISMATCH", "executor identity does not cover the execution grant");
  return withTenantLock(ctx, async () => {
    const persisted = await readDomain(ctx, "grants", grant.grant_id);
    if (canonicalJSON(persisted) !== canonicalJSON(grant)) throw new ControlPlaneError("GRANT_INVALID", "execution grant does not match the authoritative record");
    const claimID = `claim_${digest({ grant_id: grant.grant_id }).slice(7, 39)}`;
    const claim = await readDomain(ctx, "execution-claims", claimID);
    if (claim.grant_id !== grant.grant_id || claim.executor_id !== ctx.identity.actor_id || claim.policy_digest !== grant.policy_digest || claim.inputs_hash !== grant.inputs_hash) throw new ControlPlaneError("SCOPE_MISMATCH", "execution claim belongs to a different executor or grant");
    const records = await readAuthenticatedRecords(ctx, "actions", [grant.grant_id]);
    const existing = records.find((record) => record.status === value.status);
    if (existing) {
      if (existing.outputs_hash !== value.outputs_hash || existing.provider_receipt_hash !== providerReceiptHash) throw new ControlPlaneError("IDEMPOTENCY_CONFLICT", "action status already exists with different evidence");
      return existing;
    }
    const terminal = records.find((record) => ["executed", "failed", "cancelled"].includes(record.status));
    if (terminal || (records.length > 0 && value.status === "executing")) throw new ControlPlaneError("INVALID_TRANSITION", "action status transition is not allowed");
    const recordedAt = ctx.now().toISOString();
    const record = { schema_version: "control-plane.action-record/v1", record_id: `actrec_${digest({ grant_id: grant.grant_id, status: value.status }).slice(7, 39)}`, tenant_id: ctx.identity.tenant_id, actor_id: grant.actor_id, executor_id: ctx.identity.actor_id, grant_id: grant.grant_id, claim_id: claim.claim_id, decision_id: grant.decision_id, approval_id: grant.approval_id, action_id: grant.action_id, agent_run_id: grant.agent_run_id, capability: grant.capability, capability_version: grant.capability_version, target: grant.target, inputs_hash: grant.inputs_hash, outputs_hash: value.outputs_hash, status: value.status, provider_receipt_hash: providerReceiptHash, recorded_at: recordedAt };
    const transactionID = `txn_${record.record_id.slice(7)}`;
    await startTransactionLocked(ctx, {
      transaction_id: transactionID,
      domain: { path: path.join("actions", storageSegment(ctx.identity.tenant_id), storageSegment(grant.grant_id), `${String(records.length + 1).padStart(20, "0")}.json`), value: record },
      audit: { transaction_id: transactionID, event_id: `evt_${transactionID.slice(4)}`, event_type: `action_${value.status}`, tenant_id: ctx.identity.tenant_id, actor_id: ctx.identity.actor_id, timestamp: recordedAt, subject: record },
    });
    return record;
  });
}

async function verifyAuditLocked(ctx) {
  const records = await auditRecords(ctx);
  let previous = null;
  for (let index = 0; index < records.length; index += 1) {
    const event = records[index];
    if (event.sequence !== index + 1 || event.previous_event_hash !== previous || event.event_hash !== eventHash(event) || event.tenant_id !== ctx.identity.tenant_id) {
      return { valid: false, checked_events: index, first_invalid_sequence: index + 1 };
    }
    previous = event.event_hash;
  }
  const headPath = path.join(ctx.storeRoot, "audit-heads", `${storageSegment(ctx.identity.tenant_id)}.json`);
  if (records.length === 0) return { valid: !(await exists(headPath)), checked_events: 0, head_hash: null };
  if (!(await exists(headPath))) return { valid: false, checked_events: records.length, first_invalid_sequence: records.length };
  const head = await readJSON(headPath);
  const { signature, ...material } = head;
  if (material.schema_version !== "control-plane.audit-head/v1" || material.tenant_id !== ctx.identity.tenant_id || material.sequence !== records.length || material.head_hash !== previous || !safeSignature(hmac(material, ctx.auditKey), signature)) {
    return { valid: false, checked_events: records.length, first_invalid_sequence: records.length };
  }
  return { valid: true, checked_events: records.length, head_hash: previous };
}

export async function verifyAuditChain(ctx) {
  return withTenantLock(ctx, () => verifyAuditLocked(ctx), { recover: false });
}

export function publicError(error) {
  if (error instanceof ControlPlaneError) return { code: error.code, message: error.message, retryable: error.retryable };
  return { code: "INTERNAL_ERROR", message: "control-plane operation failed", retryable: false };
}
