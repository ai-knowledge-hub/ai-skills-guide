import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createHmac, generateKeyPairSync } from "node:crypto";
import { once } from "node:events";
import { mkdtemp, readFile, readdir, rename, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { createInterface } from "node:readline";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  ControlPlaneError,
  authenticateIdentity,
  authorizeAction,
  canonicalJSON,
  checkPolicy,
  contextFromEnv,
  decideApproval,
  digest,
  getApproval,
  getExecutionClaim,
  installPolicy,
  recordAction,
  revalidateExecutionClaim,
  recoverTransactions,
  requestApproval,
  signIdentity,
  storageSegment,
  validateExecutionGrant,
  verifyAuditChain,
  verifyGrant,
} from "../scripts/control_plane_core.mjs";
import { handleRequest } from "../scripts/agent_control_plane_server.mjs";
import { runDriver } from "../scripts/control_plane_auth_driver.mjs";

const identityKeys = generateKeyPairSync("ed25519");
const grantKeys = generateKeyPairSync("ed25519");
const IDENTITY_PRIVATE_KEY = identityKeys.privateKey.export({ type: "pkcs8", format: "pem" });
const IDENTITY_PUBLIC_KEY = identityKeys.publicKey.export({ type: "spki", format: "pem" });
const GRANT_PRIVATE_KEY = grantKeys.privateKey.export({ type: "pkcs8", format: "pem" });
const GRANT_PUBLIC_KEY = grantKeys.publicKey.export({ type: "spki", format: "pem" });
const AUDIT_KEY = "audit-key-for-tests-that-is-at-least-32-bytes";
const NOW = new Date("2026-09-20T10:00:00.000Z");
const TENANT = "tenant-acme";
const TARGET = { provider: "google_ads", account_id: "acct-123", resource_type: "campaign", resource_id: "cmp-456" };

function identity(overrides = {}) {
  return signIdentity({
    schema_version: "control-plane.identity/v1",
    actor_id: "campaign-agent",
    tenant_id: TENANT,
    roles: ["agent_runtime"],
    capabilities: ["apply_budget_shift"],
    scopes: [{ provider: TARGET.provider, account_id: TARGET.account_id }],
    issued_at: "2026-09-20T09:00:00.000Z",
    expires_at: "2026-09-20T11:00:00.000Z",
    key_id: "local-v1",
    ...overrides,
  }, IDENTITY_PRIVATE_KEY);
}

function adminIdentity(overrides = {}) {
  return identity({ actor_id: "control-owner", roles: ["policy_admin", "budget_owner"], ...overrides });
}

function executorIdentity(overrides = {}) {
  return identity({ actor_id: "google-ads-executor", roles: ["executor_runtime"], ...overrides });
}

function runtimeContext(storeRoot, overrides = {}) {
  return { storeRoot, identity: authenticateIdentity(JSON.stringify(identity()), IDENTITY_PUBLIC_KEY, { now: () => NOW, requiredRole: "agent_runtime" }), auditKey: AUDIT_KEY, grantPrivateKey: GRANT_PRIVATE_KEY, grantPublicKey: GRANT_PUBLIC_KEY, now: () => NOW, ...overrides };
}

function adminContext(storeRoot, overrides = {}) {
  return { storeRoot, identity: authenticateIdentity(JSON.stringify(adminIdentity()), IDENTITY_PUBLIC_KEY, { now: () => NOW }), auditKey: AUDIT_KEY, now: () => NOW, ...overrides };
}

function executorContext(storeRoot, overrides = {}) {
  return { storeRoot, identity: authenticateIdentity(JSON.stringify(executorIdentity()), IDENTITY_PUBLIC_KEY, { now: () => NOW, requiredRole: "executor_runtime" }), auditKey: AUDIT_KEY, grantPublicKey: GRANT_PUBLIC_KEY, now: () => NOW, ...overrides };
}

function policy(overrides = {}) {
  return {
    schema_version: "control-plane.policy/v1",
    policy_id: "paid-media-governance",
    version: "v1",
    tenant_id: TENANT,
    effective_at: "2026-09-20T09:00:00.000Z",
    expires_at: "2026-09-20T12:00:00.000Z",
    rules: [{
      rule_id: "budget-writes-require-owner",
      effect: "require_approval",
      capabilities: ["apply_budget_shift"],
      risk_tiers: ["high"],
      target: { provider: TARGET.provider, account_id: TARGET.account_id },
      approver_roles: ["budget_owner"],
      approval_ttl_seconds: 600,
    }],
    ...overrides,
  };
}

function proposal(overrides = {}) {
  return {
    action_id: "action-001",
    agent_run_id: "run-001",
    capability: "apply_budget_shift",
    capability_version: "v1",
    risk_tier: "high",
    target: TARGET,
    inputs_hash: digest({ amount: 100, currency: "GBP" }),
    ...overrides,
  };
}

async function fixture(t) {
  const storeRoot = await mkdtemp(path.join(os.tmpdir(), "agent-control-plane-test-"));
  t.after(() => rm(storeRoot, { recursive: true, force: true }));
  const runtime = runtimeContext(storeRoot);
  const admin = adminContext(storeRoot);
  await installPolicy(admin, policy());
  return { storeRoot, runtime, admin };
}

async function approvedFixture(t) {
  const state = await fixture(t);
  const decision = await checkPolicy(state.runtime, proposal());
  const request = await requestApproval(state.runtime, { decision_id: decision.decision_id, justification: "Budget owner review is required." });
  const approval = await decideApproval(state.admin, { approval_id: request.approval_id, decision: "approved", reason: "Approved for the bounded test account." });
  assert.equal("justification" in request, false);
  assert.match(request.justification_hash, /^sha256:[a-f0-9]{64}$/);
  assert.equal("reason" in approval, false);
  assert.match(approval.reason_hash, /^sha256:[a-f0-9]{64}$/);
  return { ...state, decision, request, approval };
}

test("signed runtime identity is authoritative and cannot be self-amended", () => {
  const signed = identity();
  assert.equal(authenticateIdentity(JSON.stringify(signed), IDENTITY_PUBLIC_KEY, { now: () => NOW, requiredRole: "agent_runtime" }).actor_id, "campaign-agent");
  assert.throws(() => authenticateIdentity(JSON.stringify({ ...signed, tenant_id: "tenant-other" }), IDENTITY_PUBLIC_KEY, { now: () => NOW }), (error) => error.code === "AUTH_INVALID");
  assert.throws(() => authenticateIdentity(JSON.stringify(identity({ expires_at: "2026-09-20T09:59:59.000Z" })), IDENTITY_PUBLIC_KEY, { now: () => NOW }), (error) => error.code === "AUTH_EXPIRED");
  const { signature: ignored, ...claims } = signed;
  assert.throws(() => signIdentity(claims, IDENTITY_PUBLIC_KEY), (error) => error.code === "CONFIGURATION_ERROR");
});

test("issuer private keys are required for agents and rejected from executors", async (t) => {
  const storeRoot = await mkdtemp(path.join(os.tmpdir(), "agent-control-plane-keys-"));
  t.after(() => rm(storeRoot, { recursive: true, force: true }));
  const common = { CONTROL_PLANE_STORE: storeRoot, CONTROL_PLANE_IDENTITY_PUBLIC_KEY: IDENTITY_PUBLIC_KEY, CONTROL_PLANE_AUDIT_KEY: AUDIT_KEY };
  assert.throws(() => contextFromEnv({ ...common, CONTROL_PLANE_IDENTITY: JSON.stringify(identity()) }, { now: () => NOW }), (error) => error.code === "CONFIGURATION_ERROR");
  const issuer = contextFromEnv({ ...common, CONTROL_PLANE_IDENTITY: JSON.stringify(identity()), CONTROL_PLANE_GRANT_ROLE_KEY: GRANT_PRIVATE_KEY }, { now: () => NOW });
  assert.ok(issuer.grantPrivateKey);
  const executor = contextFromEnv({ ...common, CONTROL_PLANE_IDENTITY: JSON.stringify(executorIdentity()), CONTROL_PLANE_GRANT_ROLE_KEY: GRANT_PUBLIC_KEY }, { now: () => NOW });
  assert.equal(executor.grantPrivateKey, undefined);
  assert.throws(() => contextFromEnv({ ...common, CONTROL_PLANE_IDENTITY: JSON.stringify(executorIdentity()), CONTROL_PLANE_GRANT_ROLE_KEY: GRANT_PRIVATE_KEY }, { now: () => NOW }), (error) => error.code === "CONFIGURATION_ERROR");
});

test("packaged authentication bootstrap supports role-specific issuer and executor bindings", async (t) => {
  const storeRoot = await mkdtemp(path.join(os.tmpdir(), "agent-control-plane-bootstrap-"));
  t.after(() => rm(storeRoot, { recursive: true, force: true }));
  const credentialBindings = ["CONTROL_PLANE_IDENTITY", "CONTROL_PLANE_IDENTITY_PUBLIC_KEY", "CONTROL_PLANE_GRANT_ROLE_KEY", "CONTROL_PLANE_AUDIT_KEY", "CONTROL_PLANE_STORE"];
  const request = { method: "custom", flow: "custom", credential_bindings: credentialBindings };
  const common = { CONTROL_PLANE_STORE: storeRoot, CONTROL_PLANE_IDENTITY_PUBLIC_KEY: IDENTITY_PUBLIC_KEY, CONTROL_PLANE_AUDIT_KEY: AUDIT_KEY };
  const issuer = await runDriver("custom", "bootstrap", request, { now: () => NOW, env: { ...common, CONTROL_PLANE_IDENTITY: JSON.stringify(identity()), CONTROL_PLANE_GRANT_ROLE_KEY: GRANT_PRIVATE_KEY } });
  const executor = await runDriver("custom", "bootstrap", request, { now: () => NOW, env: { ...common, CONTROL_PLANE_IDENTITY: JSON.stringify(executorIdentity()), CONTROL_PLANE_GRANT_ROLE_KEY: GRANT_PUBLIC_KEY } });
  assert.equal(issuer.account, `${TENANT}:campaign-agent`);
  assert.equal(executor.account, `${TENANT}:google-ads-executor`);
});

test("tenant storage encoding is injective for otherwise colliding identifiers", async (t) => {
  const storeRoot = await mkdtemp(path.join(os.tmpdir(), "agent-control-plane-tenants-"));
  t.after(() => rm(storeRoot, { recursive: true, force: true }));
  assert.notEqual(storageSegment("tenant:a"), storageSegment("tenant_a"));
  for (const tenantID of ["tenant:a", "tenant_a"]) {
    const admin = adminContext(storeRoot, { identity: authenticateIdentity(JSON.stringify(adminIdentity({ tenant_id: tenantID })), IDENTITY_PUBLIC_KEY, { now: () => NOW }) });
    await installPolicy(admin, policy({ tenant_id: tenantID }));
    assert.equal((await verifyAuditChain(admin)).valid, true);
  }
  assert.deepEqual((await readdir(path.join(storeRoot, "audit"))).sort(), [storageSegment("tenant:a"), storageSegment("tenant_a")].sort());
});

test("policy administration is separate and tenant-bound", async (t) => {
  const storeRoot = await mkdtemp(path.join(os.tmpdir(), "agent-control-plane-admin-"));
  t.after(() => rm(storeRoot, { recursive: true, force: true }));
  await assert.rejects(() => installPolicy(runtimeContext(storeRoot), policy()), (error) => error.code === "AUTH_FORBIDDEN");
  await assert.rejects(() => installPolicy(adminContext(storeRoot), policy({ tenant_id: "tenant-other" })), (error) => error.code === "SCOPE_MISMATCH");
  const activation = await installPolicy(adminContext(storeRoot), policy());
  assert.equal(activation.policy_id, "paid-media-governance");
  await assert.rejects(
    () => installPolicy(adminContext(storeRoot), policy({ rules: [{ ...policy().rules[0], approval_ttl_seconds: 300 }] })),
    (error) => error.code === "POLICY_VERSION_CONFLICT",
  );
});

test("effective risk and approval cannot be downgraded by model-supplied classification", async (t) => {
  const storeRoot = await mkdtemp(path.join(os.tmpdir(), "agent-control-plane-risk-"));
  t.after(() => rm(storeRoot, { recursive: true, force: true }));
  const rules = [
    { ...policy().rules[0], rule_id: "low-risk-allow", effect: "allow", risk_tiers: ["low"], approver_roles: undefined, approval_ttl_seconds: undefined },
    { ...policy().rules[0], rule_id: "high-risk-approval", risk_tiers: ["high"] },
  ];
  await installPolicy(adminContext(storeRoot), policy({ rules }));
  const decision = await checkPolicy(runtimeContext(storeRoot), proposal({ risk_tier: "low" }));
  assert.equal(decision.effect, "require_approval");
  assert.equal(decision.effective_risk_tier, "high");
  assert.deepEqual(decision.matched_rule_ids, ["high-risk-approval", "low-risk-allow"]);
});

test("model proposals cannot expand runtime capability, tenant, or account authority", async (t) => {
  const { runtime } = await fixture(t);
  await assert.rejects(() => checkPolicy(runtime, proposal({ capability: "publish_campaign" })), (error) => error.code === "SCOPE_MISMATCH");
  await assert.rejects(() => checkPolicy(runtime, proposal({ target: { ...TARGET, account_id: "acct-other" } })), (error) => error.code === "SCOPE_MISMATCH");
  await assert.rejects(() => checkPolicy(runtime, { ...proposal(), actor_id: "control-owner" }), (error) => error.code === "INVALID_ARGUMENT");
});

test("approval and grant bind exact actor, target, action, input hash, policy, scope, and expiry", async (t) => {
  const { runtime, decision, request } = await approvedFixture(t);
  const grant = await authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id });
  assert.equal(grant.actor_id, "campaign-agent");
  assert.equal(grant.action_id, proposal().action_id);
  assert.deepEqual(grant.target, TARGET);
  assert.equal(grant.inputs_hash, proposal().inputs_hash);
  assert.equal(grant.policy_version, "v1");
  assert.equal(grant.approval_id, request.approval_id);
  assert.deepEqual(await authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id }), grant);
  assert.deepEqual(verifyGrant(grant, GRANT_PUBLIC_KEY, { now: () => NOW }), grant);
  assert.throws(() => verifyGrant({ ...grant, inputs_hash: digest({ amount: 1000 }) }, GRANT_PUBLIC_KEY, { now: () => NOW }), (error) => error.code === "GRANT_INVALID");
});

test("approval authority is role-bound, immutable, and cannot approve a different decision", async (t) => {
  const { storeRoot, runtime, decision, request } = await approvedFixture(t);
  const wrongAdmin = adminContext(storeRoot, { identity: authenticateIdentity(JSON.stringify(adminIdentity({ actor_id: "security-reviewer", roles: ["approver"] })), IDENTITY_PUBLIC_KEY, { now: () => NOW }) });
  const second = await checkPolicy(runtime, proposal({ action_id: "action-002", inputs_hash: digest({ amount: 200 }) }));
  await assert.rejects(() => decideApproval(wrongAdmin, { approval_id: request.approval_id, decision: "approved", reason: "Not the configured role." }), (error) => error.code === "AUTH_FORBIDDEN");
  await assert.rejects(() => authorizeAction(runtime, { decision_id: second.decision_id, approval_id: request.approval_id }), (error) => ["APPROVAL_INVALID", "STORE_CORRUPT"].includes(error.code));
  await assert.rejects(() => decideApproval(adminContext(storeRoot), { approval_id: request.approval_id, decision: "rejected", reason: "Attempt to replace decision." }), (error) => error.code === "APPROVAL_ALREADY_DECIDED");
});

test("policy replacement invalidates earlier decisions and approvals", async (t) => {
  const { runtime, admin, decision, request } = await approvedFixture(t);
  await installPolicy(admin, policy({ version: "v2", rules: [{ ...policy().rules[0], effect: "deny", approver_roles: undefined, approval_ttl_seconds: undefined }] }));
  await assert.rejects(() => authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id }), (error) => error.code === "POLICY_CHANGED");
});

test("executor revalidates current policy and approval at the external effect boundary", async (t) => {
  const { storeRoot, runtime, admin, decision, request } = await approvedFixture(t);
  const grant = await authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id });
  const executor = executorContext(storeRoot);
  const validation = await validateExecutionGrant(executor, { grant });
  assert.equal(validation.valid, true);
  assert.equal(validation.executor_id, "google-ads-executor");
  assert.match(validation.claim_id, /^claim_[a-f0-9]{32}$/);
  assert.equal((await revalidateExecutionClaim(executor, { grant })).current, true);
  await assert.rejects(() => validateExecutionGrant(executor, { grant }), (error) => error.code === "ACTION_ALREADY_CLAIMED");
  await assert.rejects(() => validateExecutionGrant(runtime, { grant }), (error) => error.code === "AUTH_FORBIDDEN");
  await installPolicy(admin, policy({ version: "v2", rules: [{ ...policy().rules[0], effect: "deny", approver_roles: undefined, approval_ttl_seconds: undefined }] }));
  assert.equal((await getExecutionClaim(executor, { grant })).claim_id, validation.claim_id);
  await assert.rejects(() => revalidateExecutionClaim(executor, { grant }), (error) => error.code === "POLICY_CHANGED");
  await assert.rejects(() => validateExecutionGrant(executor, { grant }), (error) => error.code === "POLICY_CHANGED");
});

test("duplicate requests are idempotent and concurrent policy checks record one decision", async (t) => {
  const { runtime } = await fixture(t);
  const decisions = await Promise.all(Array.from({ length: 8 }, () => checkPolicy(runtime, proposal())));
  assert.equal(new Set(decisions.map((item) => item.decision_id)).size, 1);
  const requests = await Promise.all(Array.from({ length: 5 }, () => requestApproval(runtime, { decision_id: decisions[0].decision_id, justification: "Same exact request." })));
  assert.equal(new Set(requests.map((item) => item.approval_id)).size, 1);
  const audit = await verifyAuditChain(runtime);
  assert.equal(audit.valid, true);
  assert.equal(audit.checked_events, 3);
});

test("the effect-boundary claim is atomic, single-use, and recoverable after an ambiguous effect", async (t) => {
  const { storeRoot, runtime, decision, request } = await approvedFixture(t);
  const grant = await authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id });
  const firstExecutor = executorContext(storeRoot);
  const secondExecutor = executorContext(storeRoot, { identity: authenticateIdentity(JSON.stringify(executorIdentity({ actor_id: "backup-executor" })), IDENTITY_PUBLIC_KEY, { now: () => NOW, requiredRole: "executor_runtime" }) });
  const results = await Promise.allSettled([validateExecutionGrant(firstExecutor, { grant }), validateExecutionGrant(secondExecutor, { grant })]);
  assert.equal(results.filter((result) => result.status === "fulfilled").length, 1);
  assert.equal(results.filter((result) => result.status === "rejected" && result.reason.code === "ACTION_ALREADY_CLAIMED").length, 1);
  const owner = results[0].status === "fulfilled" ? firstExecutor : secondExecutor;
  const nonOwner = owner === firstExecutor ? secondExecutor : firstExecutor;
  const recovered = await getExecutionClaim({ ...owner, now: () => new Date("2026-09-20T12:00:00.000Z") }, { grant });
  assert.equal(recovered.claim_id, results.find((result) => result.status === "fulfilled").value.claim_id);
  assert.equal(recovered.recovered, true);
  await assert.rejects(() => revalidateExecutionClaim({ ...owner, now: () => new Date("2026-09-20T12:00:00.000Z") }, { grant }), (error) => error.code === "GRANT_EXPIRED");
  await assert.rejects(() => getExecutionClaim(nonOwner, { grant }), (error) => error.code === "SCOPE_MISMATCH");
  const reconciled = await recordAction(owner, { grant, status: "executed", outputs_hash: digest({ provider_state: "reconciled" }), provider_receipt: "provider-reconciliation-001" });
  assert.equal(reconciled.status, "executed");
  await assert.rejects(() => getExecutionClaim(owner, { grant }), (error) => error.code === "ACTION_ALREADY_RECORDED");
  await assert.rejects(() => validateExecutionGrant(owner, { grant }), (error) => error.code === "ACTION_ALREADY_RECORDED");
});

test("an interrupted effect-boundary claim recovers as claimed at every durable boundary", async (t) => {
  for (const faultPoint of ["after_intent", "after_domain", "after_audit", "after_receipt"]) {
    await t.test(faultPoint, async (subtest) => {
      const { storeRoot, runtime, decision, request } = await approvedFixture(subtest);
      const grant = await authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id });
      let injected = false;
      const interrupted = executorContext(storeRoot, { fault: async (point, transaction) => {
        if (!injected && point === faultPoint && transaction.audit.event_type === "execution_claimed") {
          injected = true;
          throw new Error("simulated claim process loss");
        }
      } });
      await assert.rejects(() => validateExecutionGrant(interrupted, { grant }), /simulated claim process loss/);
      const restarted = executorContext(storeRoot);
      await recoverTransactions(restarted);
      await assert.rejects(() => validateExecutionGrant(restarted, { grant }), (error) => error.code === "ACTION_ALREADY_CLAIMED");
      assert.equal((await getExecutionClaim(restarted, { grant })).grant_id, grant.grant_id);
      assert.equal((await recordAction(restarted, { grant, status: "cancelled", outputs_hash: null, provider_receipt: `reconciled-${faultPoint}` })).status, "cancelled");
    });
  }
});

test("authorization rejects domain records that diverge from authenticated audit evidence", async (t) => {
  const { storeRoot, runtime, decision, request } = await approvedFixture(t);
  const decisionPath = path.join(storeRoot, "decisions", storageSegment(TENANT), `${storageSegment(decision.decision_id)}.json`);
  const tamperedDecision = JSON.parse(await readFile(decisionPath, "utf8"));
  tamperedDecision.effect = "allow";
  await writeFile(decisionPath, JSON.stringify(tamperedDecision));
  await assert.rejects(() => authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id }), (error) => error.code === "STORE_CORRUPT");
});

test("approval records cannot be substituted independently of their audit event", async (t) => {
  const { storeRoot, runtime, decision, request } = await approvedFixture(t);
  const approvalPath = path.join(storeRoot, "approval-decisions", storageSegment(TENANT), `${storageSegment(request.approval_id)}.json`);
  const tamperedApproval = JSON.parse(await readFile(approvalPath, "utf8"));
  tamperedApproval.decision = "rejected";
  await writeFile(approvalPath, JSON.stringify(tamperedApproval));
  await assert.rejects(() => authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id }), (error) => error.code === "STORE_CORRUPT");
});

test("action lifecycle rejects reordering, conflicting duplicates, and terminal resurrection", async (t) => {
  const { storeRoot, runtime, decision, request } = await approvedFixture(t);
  const grant = await authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id });
  const executor = executorContext(storeRoot);
  assert.equal((await validateExecutionGrant(executor, { grant })).valid, true);
  await assert.rejects(() => recordAction(runtime, { grant, status: "executing", outputs_hash: null, provider_receipt: "provider:request-1" }), (error) => error.code === "AUTH_FORBIDDEN");
  await assert.rejects(() => authorizeAction(executor, { decision_id: decision.decision_id, approval_id: request.approval_id }), (error) => error.code === "AUTH_FORBIDDEN");
  const secretReceipt = "xoxb-this-would-be-a-durable-secret";
  const executing = await recordAction(executor, { grant, status: "executing", outputs_hash: null, provider_receipt: secretReceipt });
  assert.equal(executing.actor_id, "campaign-agent");
  assert.equal(executing.executor_id, "google-ads-executor");
  assert.equal("provider_receipt" in executing, false);
  assert.equal(executing.provider_receipt_hash, digest(secretReceipt));
  assert.equal((await recordAction(executor, { grant, status: "executing", outputs_hash: null, provider_receipt: secretReceipt })).record_id, executing.record_id);
  await assert.rejects(() => recordAction(executor, { grant, status: "executing", outputs_hash: null, provider_receipt: "provider:request-other" }), (error) => error.code === "IDEMPOTENCY_CONFLICT");
  await recordAction(executor, { grant, status: "executed", outputs_hash: digest({ ok: true }), provider_receipt: "provider:receipt-1" });
  await assert.rejects(() => recordAction(executor, { grant, status: "failed", outputs_hash: null, provider_receipt: "provider:late-failure" }), (error) => error.code === "INVALID_TRANSITION");
  await assert.rejects(() => authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id }), (error) => error.code === "ACTION_ALREADY_RECORDED");
});

test("restart recovery completes transactions interrupted at every durable boundary", async (t) => {
  for (const faultPoint of ["after_intent", "after_domain", "after_audit", "after_receipt"]) {
    await t.test(faultPoint, async (subtest) => {
      const { storeRoot } = await fixture(subtest);
      let injected = false;
      const interrupted = runtimeContext(storeRoot, { fault: async (point, transaction) => {
        if (!injected && point === faultPoint && transaction.audit.event_type === "policy_decided") {
          injected = true;
          throw new Error("simulated process loss");
        }
      } });
      await assert.rejects(() => checkPolicy(interrupted, proposal()), /simulated process loss/);
      const restarted = runtimeContext(storeRoot);
      await recoverTransactions(restarted);
      const recovered = await checkPolicy(restarted, proposal());
      assert.match(recovered.decision_id, /^pdec_/);
      assert.equal((await verifyAuditChain(restarted)).valid, true);
    });
  }
});

test("recovery rejects unauthenticated and path-escaping transaction intents", async (t) => {
  const { storeRoot, runtime } = await fixture(t);
  let injected = false;
  const interrupted = runtimeContext(storeRoot, { fault: async (point, transaction) => {
    if (!injected && point === "after_intent" && transaction.audit.event_type === "policy_decided") {
      injected = true;
      throw new Error("retain authenticated intent for mutation probe");
    }
  } });
  await assert.rejects(() => checkPolicy(interrupted, proposal()), /retain authenticated intent/);
  const transactionsDir = path.join(storeRoot, "transactions", storageSegment(TENANT));
  const existingName = (await readdir(transactionsDir))[0];
  const existing = JSON.parse(await readFile(path.join(transactionsDir, existingName), "utf8"));

  const unsignedMutation = structuredClone(existing);
  unsignedMutation.transaction_id = "txn_unsigned-injection";
  unsignedMutation.audit.transaction_id = unsignedMutation.transaction_id;
  await writeFile(path.join(transactionsDir, "unsigned.json"), JSON.stringify(unsignedMutation));
  await assert.rejects(() => recoverTransactions(runtime), (error) => error.code === "STORE_CORRUPT");
  await rm(path.join(transactionsDir, "unsigned.json"));

  const { signature: ignored, ...baseMaterial } = existing;
  const material = structuredClone(baseMaterial);
  material.transaction_id = "txn_path-injection";
  material.domain.path = `decisions/${storageSegment(TENANT)}/../../escaped.json`;
  material.audit.transaction_id = material.transaction_id;
  material.audit.event_id = "evt_path-injection";
  material.audit.domain_path = material.domain.path;
  const signature = createHmac("sha256", AUDIT_KEY).update(canonicalJSON(material)).digest("hex");
  await writeFile(path.join(transactionsDir, "path-escape.json"), JSON.stringify({ ...material, signature }));
  await assert.rejects(() => recoverTransactions(runtime), (error) => error.code === "STORE_CORRUPT");
  await assert.rejects(() => readFile(path.join(storeRoot, "escaped.json"), "utf8"), (error) => error.code === "ENOENT");
});

test("completed transaction history is compacted and one operation loads audit evidence once", async (t) => {
  const { storeRoot, runtime } = await fixture(t);
  for (let index = 0; index < 32; index += 1) {
    await checkPolicy(runtime, proposal({ action_id: `action-cardinality-${index}`, inputs_hash: digest({ index }) }));
  }
  const transactionsDir = path.join(storeRoot, "transactions", storageSegment(TENANT));
  assert.deepEqual(await readdir(transactionsDir), []);
  const metrics = {};
  await checkPolicy({ ...runtime, metrics }, proposal({ action_id: "action-cardinality-final", inputs_hash: digest({ index: 32 }) }));
  assert.equal(metrics.audit_directory_reads, 1);
  assert.deepEqual(await readdir(transactionsDir), []);
});

async function tamperFixture(t) {
  const state = await approvedFixture(t);
  await authorizeAction(state.runtime, { decision_id: state.decision.decision_id, approval_id: state.request.approval_id });
  const auditDir = path.join(state.storeRoot, "audit", storageSegment(TENANT));
  return { ...state, auditDir, files: await readdir(auditDir) };
}

test("audit verification detects deletion, substitution, and reordering", async (t) => {
  await t.test("deletion", async (subtest) => {
    const { runtime, auditDir, files } = await tamperFixture(subtest);
    await rm(path.join(auditDir, files[1]));
    assert.equal((await verifyAuditChain(runtime)).valid, false);
  });
  await t.test("tail deletion", async (subtest) => {
    const { runtime, auditDir, files } = await tamperFixture(subtest);
    await rm(path.join(auditDir, files.at(-1)));
    assert.equal((await verifyAuditChain(runtime)).valid, false);
  });
  await t.test("substitution", async (subtest) => {
    const { runtime, auditDir, files } = await tamperFixture(subtest);
    const file = path.join(auditDir, files[1]);
    const event = JSON.parse(await readFile(file, "utf8"));
    event.actor_id = "substituted-agent";
    await writeFile(file, JSON.stringify(event));
    assert.equal((await verifyAuditChain(runtime)).valid, false);
  });
  await t.test("reordering", async (subtest) => {
    const { runtime, auditDir, files } = await tamperFixture(subtest);
    const first = path.join(auditDir, files[0]);
    const second = path.join(auditDir, files[1]);
    const temp = path.join(auditDir, "swap.tmp");
    await rename(first, temp);
    await rename(second, first);
    await rename(temp, second);
    assert.equal((await verifyAuditChain(runtime)).valid, false);
  });
  await t.test("tampering blocks later authorization work", async (subtest) => {
    const { runtime, auditDir, files, decision, request } = await tamperFixture(subtest);
    await rm(path.join(auditDir, files.at(-1)));
    await assert.rejects(() => authorizeAction(runtime, { decision_id: decision.decision_id, approval_id: request.approval_id }), (error) => error.code === "STORE_CORRUPT");
  });
});

test("MCP protocol exposes bounded governance tools and returns structured denials", async (t) => {
  const { runtime } = await fixture(t);
  const initialized = await handleRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} }, { context: runtime });
  assert.equal(initialized.serverInfo.name, "agent-control-plane-server");
  const listed = await handleRequest({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} }, { context: runtime });
  assert.deepEqual(listed.tools.map((tool) => tool.name), ["check_policy", "request_approval", "get_approval", "authorize_action", "record_agent_action", "validate_execution_grant", "get_execution_claim", "revalidate_execution_claim", "verify_audit_chain"]);
  const denied = await handleRequest({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "check_policy", arguments: proposal({ target: { ...TARGET, account_id: "acct-other" } }) } }, { context: runtime });
  assert.equal(denied.isError, true);
  assert.equal(denied.structuredContent.error.code, "SCOPE_MISMATCH");
});

test("auth status is identity-bound, time-bounded, and audit-verified", async (t) => {
  const { runtime } = await fixture(t);
  const status = await runDriver("custom", "status", {}, { context: runtime, now: () => NOW });
  assert.equal(status.principal, "campaign-agent");
  assert.equal(status.account, TENANT);
  assert.equal(status.attestation_expires_at, "2026-09-20T10:05:00.000Z");
  assert.match(status.attestation_reference, /^audit-head:sha256:/);
});

test("launchable stdio process completes initialize and tools/list on a clean local profile", async (t) => {
  const { storeRoot } = await fixture(t);
  const child = spawn(process.execPath, [fileURLToPath(new URL("../scripts/agent_control_plane_server.mjs", import.meta.url))], {
    env: { ...process.env, CONTROL_PLANE_STORE: storeRoot, CONTROL_PLANE_IDENTITY: JSON.stringify(identity()), CONTROL_PLANE_IDENTITY_PUBLIC_KEY: IDENTITY_PUBLIC_KEY, CONTROL_PLANE_GRANT_ROLE_KEY: GRANT_PRIVATE_KEY, CONTROL_PLANE_AUDIT_KEY: AUDIT_KEY },
    stdio: ["pipe", "pipe", "pipe"],
  });
  const responses = [];
  const lines = createInterface({ input: child.stdout, crlfDelay: Infinity });
  lines.on("line", (line) => responses.push(JSON.parse(line)));
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`);
  child.stdin.end();
  await once(child, "exit");
  assert.equal(child.exitCode, 0);
  assert.equal(responses[0].result.serverInfo.name, "agent-control-plane-server");
  assert.equal(responses[1].result.tools.length, 9);
});
