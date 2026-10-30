import assert from "node:assert/strict";
import { once } from "node:events";
import { mkdir, mkdtemp, readFile, readdir, realpath, unlink, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { createInterface } from "node:readline";
import { spawn } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  ExecutorError,
  canonicalJSON,
  contextFromEnv,
  digest,
  executeApprovedChange,
  getExecution,
  packageTreeSHA256,
  parsePolicy,
  previewChange,
  reconcileExecution,
  rollbackExecution,
  rollbackRequest,
  validatePlan,
} from "../scripts/ad_platform_executor_core.mjs";
import { handleRequest } from "../scripts/ad_platform_executor_server.mjs";
import { runDriver } from "../scripts/ad_platform_executor_auth_driver.mjs";

const NOW = new Date("2026-09-20T12:00:00.000Z");
const TEST_CONTROL_PLANE_INTEGRITY = JSON.stringify({ version: "0.2.1", runtime: "generic", tree_sha256: "a".repeat(64), runtime_contract_sha256: "b".repeat(64) });

function policy() {
  return {
    environmentTier: "sandbox", allowLive: false, maxTimeoutMs: 30_000, googleAdsVersion: "v25", dv360Version: "v4",
    targets: [
      { provider: "google_ads", account_id: "1234567890", resource_type: "campaign_budget", resource_ids: ["22222"], operations: ["set_budget_micros"] },
      { provider: "dv360", account_id: "987654", resource_type: "line_item", resource_ids: ["456789"], operations: ["set_status", "set_bid_micros"] },
    ],
  };
}

function plan(overrides = {}) {
  return {
    schema_version: "ad-platform.change-plan/v2",
    plan_id: "plan-123",
    action_id: "action-123",
    provider: "google_ads",
    account_id: "1234567890",
    resource_type: "campaign_budget",
    resource_id: "22222",
    operation: "set_budget_micros",
    proposed_diff: { field: "budget_micros", from: 1_000_000, to: 900_000 },
    generated_at: "2026-09-20T11:59:00.000Z",
    expires_at: "2026-09-20T12:05:00.000Z",
    ...overrides,
  };
}

function grant(changePlan = plan(), overrides = {}) {
  return {
    schema_version: "control-plane.execution-grant/v1",
    grant_id: "grant_11111111111111111111111111111111",
    action_id: changePlan.action_id,
    capability: "ad_platform.execute",
    target: { provider: changePlan.provider, account_id: changePlan.account_id, resource_type: changePlan.resource_type, resource_id: changePlan.resource_id },
    inputs_hash: digest(changePlan),
    ...overrides,
  };
}

function mutableProvider(initial = 1_000_000, options = {}) {
  let value = initial;
  let mutationCalls = 0;
  return {
    get value() { return value; },
    get mutationCalls() { return mutationCalls; },
    async read(changePlan) { if (options.read) return options.read(changePlan, value); return { target: { provider: changePlan.provider, account_id: changePlan.account_id, resource_type: changePlan.resource_type, resource_id: changePlan.resource_id }, field: changePlan.proposed_diff.field, value }; },
    async mutate(changePlan, next) {
      mutationCalls += 1;
      if (options.mutate) return options.mutate(changePlan, next, { get: () => value, set: (item) => { value = item; }, calls: mutationCalls });
      value = next;
      return { requestID: `provider:req-${mutationCalls}`, responseHash: digest({ next, mutationCalls }) };
    },
  };
}

async function fixture(options = {}) {
  const calls = [];
  const claimed = new Set();
  const provider = options.provider ?? mutableProvider();
  const storeRoot = await mkdtemp(path.join(os.tmpdir(), "ad-platform-executor-"));
  const ctx = {
    env: {}, policy: policy(), credential: {}, storeRoot, controlPlaneRoot: storeRoot,
    auditKey: "a".repeat(64), now: () => NOW, clock: options.clock ?? Date.now, provider,
    fault: options.fault ?? (() => {}),
    controlPlane: async (tool, args) => {
      calls.push({ tool, args });
      if (options.controlPlane) return options.controlPlane(tool, args, { calls, claimed });
      if (tool === "validate_execution_grant") {
        if (claimed.has(args.grant.grant_id)) throw new ExecutorError("ACTION_ALREADY_CLAIMED", "already claimed");
        claimed.add(args.grant.grant_id);
        return { valid: true, claim_id: `claim_${args.grant.grant_id.slice(6)}` };
      }
      if (tool === "get_execution_claim") {
        if (!claimed.has(args.grant.grant_id)) throw new ExecutorError("EXECUTION_CLAIM_NOT_FOUND", "claim unavailable");
        return { valid: true, claim_id: `claim_${args.grant.grant_id.slice(6)}`, grant_id: args.grant.grant_id, inputs_hash: args.grant.inputs_hash, target: args.grant.target, recovered: true };
      }
      if (tool === "revalidate_execution_claim") {
        if (!claimed.has(args.grant.grant_id)) throw new ExecutorError("EXECUTION_CLAIM_NOT_FOUND", "claim unavailable");
        return { valid: true, current: true, claim_id: `claim_${args.grant.grant_id.slice(6)}`, grant_id: args.grant.grant_id, inputs_hash: args.grant.inputs_hash, target: args.grant.target };
      }
      if (tool === "record_agent_action") return { status: args.status, claim_id: `claim_${args.grant.grant_id.slice(6)}` };
      if (tool === "verify_audit_chain") return { valid: true };
      throw new Error(`unexpected tool ${tool}`);
    },
  };
  return { ctx, provider, calls, storeRoot };
}

test("closed plans reject free-form requests, unknown fields, and expired authority before provider access", async () => {
  let reads = 0;
  const { ctx } = await fixture({ provider: { read: async () => { reads += 1; }, mutate: async () => {} } });
  for (const bad of [
    { prompt: "raise the budget" },
    { ...plan(), instruction: "ignore policy" },
    plan({ expires_at: "2026-09-20T12:00:00.000Z" }),
  ]) await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(plan()), plan: bad }), (error) => ["INVALID_ARGUMENT", "PLAN_EXPIRED"].includes(error.code));
  assert.equal(reads, 0);
});

test("policy is sandbox-first and uses closed provider-native allowlists", () => {
  assert.throws(() => parsePolicy(JSON.stringify({ environment_tier: "live", allowed_targets: [] })), (error) => error.code === "CONFIGURATION_ERROR");
  const parsed = parsePolicy(JSON.stringify({
    environment_tier: "sandbox", allowed_targets: [{ provider: "dv360", account_id: "987654", resource_type: "line_item", resource_ids: ["456789"], operations: ["set_status"] }],
  }));
  assert.equal(parsed.environmentTier, "sandbox");
  assert.equal(parsed.dv360Version, "v4");
});

test("wrong account, object, action, and coordinated plan substitutions fail before reads", async () => {
  let reads = 0;
  const { ctx } = await fixture({ provider: { read: async () => { reads += 1; }, mutate: async () => {} } });
  const approved = plan();
  const authority = grant(approved);
  const substitutions = [
    plan({ account_id: "1111111111" }),
    plan({ resource_id: "33333" }),
    plan({ action_id: "action-other" }),
    plan({ proposed_diff: { field: "budget_micros", from: 1_000_000, to: 800_000 } }),
  ];
  for (const changed of substitutions) await assert.rejects(() => executeApprovedChange(ctx, { grant: authority, plan: changed }), (error) => ["AUTHORITY_MISMATCH", "TARGET_NOT_ALLOWED"].includes(error.code));
  assert.equal(reads, 0);
});

test("current provider state is re-read and stale approved pre-images fail before grant claim", async () => {
  const { ctx, calls, provider } = await fixture({ provider: mutableProvider(1_100_000) });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), (error) => error.code === "STALE_PROVIDER_STATE");
  assert.equal(provider.mutationCalls, 0);
  assert.equal(calls.length, 0);
});

test("control-plane approval or policy rejection stops the effect at the claim boundary", async () => {
  const { ctx, provider } = await fixture({ controlPlane: async (tool) => { if (tool === "validate_execution_grant") throw new ExecutorError("APPROVAL_EXPIRED", "expired"); } });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), (error) => error.code === "APPROVAL_EXPIRED");
  assert.equal(provider.mutationCalls, 0);
});

test("successful effects are claimed once, verified, receipted, and idempotent", async () => {
  const { ctx, provider, calls } = await fixture();
  const first = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const second = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  assert.equal(first.execution_id, second.execution_id);
  assert.equal(first.status, "executed");
  assert.equal(first.post_image_hash, digest({ target: first.target, field: "budget_micros", value: 900_000 }));
  assert.equal(provider.mutationCalls, 1);
  assert.deepEqual(calls.map((call) => `${call.tool}:${call.args.status ?? "claim"}`), ["validate_execution_grant:claim", "revalidate_execution_claim:claim", "record_agent_action:executing", "record_agent_action:executed", "record_agent_action:executed"]);
  assert.deepEqual(await getExecution(ctx, { execution_id: first.execution_id }), first);
});

test("concurrent duplicates serialize per resource and produce one provider effect", async () => {
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const provider = mutableProvider(1_000_000, { mutate: async (_plan, next, state) => { await gate; state.set(next); return { requestID: "provider:one", responseHash: digest({ ok: true }) }; } });
  const { ctx } = await fixture({ provider });
  const first = executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const second = executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  await new Promise((resolve) => setTimeout(resolve, 20));
  release();
  const [left, right] = await Promise.all([first, second]);
  assert.equal(left.execution_id, right.execution_id);
  assert.equal(provider.mutationCalls, 1);
});

test("ambiguous provider effects require reconciliation and cannot be replayed", async () => {
  const provider = mutableProvider(1_000_000, { mutate: async (_plan, next, state) => { state.set(next); throw new ExecutorError("UPSTREAM_TIMEOUT", "timeout", { retryable: true, ambiguous: true }); } });
  const { ctx } = await fixture({ provider });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), (error) => error.code === "RECONCILIATION_REQUIRED");
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), (error) => error.code === "RECONCILIATION_REQUIRED");
  const reconciled = await reconcileExecution(ctx, { grant: grant(), plan: plan() });
  assert.equal(reconciled.status, "executed");
  assert.equal(reconciled.reconciled, true);
  assert.equal(provider.mutationCalls, 1);
});

test("a crash after provider effect is recovered from provider state without repeating the effect", async () => {
  let crashed = false;
  const provider = mutableProvider();
  const { ctx } = await fixture({ provider, fault: (point) => { if (point === "after_effect" && !crashed) { crashed = true; throw new Error("crash"); } } });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), /crash/);
  assert.equal(provider.value, 900_000);
  assert.equal(provider.mutationCalls, 1);
  const receipt = await reconcileExecution({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() });
  assert.equal(receipt.reconciled, true);
  assert.equal(provider.mutationCalls, 1);
});

test("a crash after durable receipt returns the receipt without another mutation", async () => {
  const provider = mutableProvider();
  const { ctx, calls } = await fixture({ provider, fault: (point) => { if (point === "after_receipt") throw new Error("crash"); } });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), /crash/);
  const recovered = await executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() });
  assert.equal(recovered.status, "executed");
  assert.equal(provider.mutationCalls, 1);
  assert.equal(calls.at(-1).args.status, "executed");
});

test("forward execution recovers a durable remote claim lost before local recording", async () => {
  let crashed = false;
  const provider = mutableProvider();
  const { ctx, calls } = await fixture({ provider, fault: (point) => { if (point === "after_execution_control_plane_claim" && !crashed) { crashed = true; throw new Error("claim response lost"); } } });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), /claim response lost/);
  assert.equal(provider.mutationCalls, 0);
  const recovered = await executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() });
  assert.equal(recovered.status, "executed");
  assert.equal(provider.mutationCalls, 1);
  assert.equal(calls.filter((call) => call.tool === "get_execution_claim").length, 1);
});

test("a recovered historical claim cannot begin an effect after authority ends", async () => {
  let crashed = false;
  const provider = mutableProvider();
  const { ctx } = await fixture({ provider, fault: (point) => { if (point === "after_execution_control_plane_claim" && !crashed) { crashed = true; throw new Error("claim response lost"); } } });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), /claim response lost/);
  const authoritative = ctx.controlPlane;
  ctx.controlPlane = async (tool, args, deadline) => {
    if (tool === "revalidate_execution_claim") throw new ExecutorError("POLICY_CHANGED", "policy replaced");
    return authoritative(tool, args, deadline);
  };
  await assert.rejects(() => executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() }), (error) => error.code === "POLICY_CHANGED");
  assert.equal(provider.mutationCalls, 0);
  await assert.rejects(() => executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() }), (error) => error.code === "EXECUTION_CANCELLED");
});

test("authority cancellation resumes across both local and remote commit windows", async () => {
  for (const faultPoint of ["after_execution_cancellation_pending", "after_execution_cancellation_action"]) {
    let claimResponseLost = false;
    let cancellationFaulted = false;
    let revalidationAttempts = 0;
    const provider = mutableProvider();
    const { ctx, calls, storeRoot } = await fixture({ provider, fault: (point) => {
      if (point === "after_execution_control_plane_claim" && !claimResponseLost) { claimResponseLost = true; throw new Error("claim response lost"); }
    } });
    await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), /claim response lost/);
    const authoritative = ctx.controlPlane;
    ctx.controlPlane = async (tool, args, deadline) => {
      if (tool === "revalidate_execution_claim") { revalidationAttempts += 1; throw new ExecutorError("POLICY_CHANGED", "policy replaced"); }
      return authoritative(tool, args, deadline);
    };
    ctx.fault = (point) => { if (point === faultPoint && !cancellationFaulted) { cancellationFaulted = true; throw new Error(`crash:${point}`); } };
    await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), new RegExp(`crash:${faultPoint}`));
    const intentFile = (await readdir(path.join(storeRoot, "intents")))[0];
    const pending = JSON.parse(await readFile(path.join(storeRoot, "intents", intentFile), "utf8")).value;
    assert.equal(pending.state, "cancellation_pending");
    assert.equal(pending.cancellation_code, "POLICY_CHANGED");
    await assert.rejects(() => executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() }), (error) => error.code === "POLICY_CHANGED");
    assert.equal(provider.mutationCalls, 0);
    assert.equal(revalidationAttempts, 1);
    const cancellations = calls.filter((call) => call.tool === "record_agent_action" && call.args.status === "cancelled");
    assert.ok(cancellations.length >= 1);
    assert.ok(cancellations.every((call) => canonicalJSON(call.args) === canonicalJSON(cancellations[0].args)));
    await assert.rejects(() => executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() }), (error) => error.code === "EXECUTION_CANCELLED");
  }
});

test("control-plane unavailability leaves authority cancellation pending and retryable", async () => {
  let claimResponseLost = false;
  let cancellationAttempts = 0;
  const provider = mutableProvider();
  const { ctx, storeRoot } = await fixture({ provider, fault: (point) => {
    if (point === "after_execution_control_plane_claim" && !claimResponseLost) { claimResponseLost = true; throw new Error("claim response lost"); }
  } });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), /claim response lost/);
  const authoritative = ctx.controlPlane;
  ctx.controlPlane = async (tool, args, deadline) => {
    if (tool === "revalidate_execution_claim") throw new ExecutorError("POLICY_CHANGED", "policy replaced");
    if (tool === "record_agent_action" && args.status === "cancelled" && cancellationAttempts++ === 0) throw new ExecutorError("CONTROL_PLANE_UNAVAILABLE", "unavailable", { retryable: true });
    return authoritative(tool, args, deadline);
  };
  await assert.rejects(() => executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() }), (error) => error.code === "CONTROL_PLANE_UNAVAILABLE" && error.retryable === true);
  const intentFile = (await readdir(path.join(storeRoot, "intents")))[0];
  const pending = JSON.parse(await readFile(path.join(storeRoot, "intents", intentFile), "utf8")).value;
  assert.equal(pending.state, "cancellation_pending");
  await assert.rejects(() => executeApprovedChange({ ...ctx, fault: () => {} }, { grant: grant(), plan: plan() }), (error) => error.code === "POLICY_CHANGED");
  assert.equal(provider.mutationCalls, 0);
});

test("post-write verification never overwrites a concurrent third state", async () => {
  const provider = mutableProvider(1_000_000, { mutate: async (_plan, next, state) => {
    if (state.calls === 1) state.set(950_000); else state.set(next);
    return { requestID: `provider:${state.calls}`, responseHash: digest({ call: state.calls }) };
  } });
  const { ctx } = await fixture({ provider });
  await assert.rejects(() => executeApprovedChange(ctx, { grant: grant(), plan: plan() }), (error) => error.code === "MANUAL_RECONCILIATION_REQUIRED");
  assert.equal(provider.value, 950_000);
  assert.equal(provider.mutationCalls, 1);
});

test("rollback requires a new exact grant and refuses state changed after execution", async () => {
  const { ctx, provider, storeRoot } = await fixture();
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const files = await import("node:fs/promises").then(({ readdir }) => readdir(path.join(storeRoot, "receipts")));
  const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
  const request = rollbackRequest(executed.execution_id, envelope.value);
  const rollbackGrant = grant(plan(), { grant_id: "grant_22222222222222222222222222222222", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
  await assert.rejects(() => rollbackExecution(ctx, { grant: { ...rollbackGrant, inputs_hash: digest({ wrong: true }) }, execution_id: executed.execution_id }), (error) => error.code === "AUTHORITY_MISMATCH");
  const rolledBack = await rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id });
  assert.equal(rolledBack.status, "rolled_back");
  assert.equal(provider.value, 1_000_000);
});

test("provider-state divergence blocks rollback without applying a mutation", async () => {
  const { ctx, provider, storeRoot } = await fixture();
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const files = await import("node:fs/promises").then(({ readdir }) => readdir(path.join(storeRoot, "receipts")));
  const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
  const request = rollbackRequest(executed.execution_id, envelope.value);
  const rollbackGrant = grant(plan(), { grant_id: "grant_33333333333333333333333333333333", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
  await provider.mutate(plan(), 850_000);
  const before = provider.mutationCalls;
  await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), (error) => error.code === "ROLLBACK_CONFLICT");
  assert.equal(provider.mutationCalls, before);
});

test("rollback rechecks current executor policy before provider access", async () => {
  let reads = 0;
  const underlying = mutableProvider();
  const provider = {
    get mutationCalls() { return underlying.mutationCalls; },
    read: async (...args) => { reads += 1; return underlying.read(...args); },
    mutate: (...args) => underlying.mutate(...args),
  };
  const { ctx, storeRoot } = await fixture({ provider });
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const files = await readdir(path.join(storeRoot, "receipts"));
  const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
  const request = rollbackRequest(executed.execution_id, envelope.value);
  const rollbackGrant = grant(plan(), { grant_id: "grant_55555555555555555555555555555555", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
  ctx.policy = { ...ctx.policy, targets: [] };
  const beforeReads = reads;
  await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), (error) => error.code === "TARGET_NOT_ALLOWED");
  assert.equal(reads, beforeReads);
  assert.equal(provider.mutationCalls, 1);
});

test("rollback resumes a crash after claim without reclaiming and reconciles a crash after effect", async () => {
  for (const faultPoint of ["after_rollback_claim", "after_rollback_effect"]) {
    let crashed = false;
    const provider = mutableProvider();
    const { ctx, storeRoot, calls } = await fixture({ provider, fault: (point) => { if (point === faultPoint && !crashed) { crashed = true; throw new Error(`crash:${point}`); } } });
    const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
    const files = await readdir(path.join(storeRoot, "receipts"));
    const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
    const request = rollbackRequest(executed.execution_id, envelope.value);
    const rollbackGrant = grant(plan(), { grant_id: faultPoint === "after_rollback_claim" ? "grant_66666666666666666666666666666666" : "grant_77777777777777777777777777777777", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
    await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), /crash:after_rollback/);
    const recovered = await rollbackExecution({ ...ctx, fault: () => {} }, { grant: rollbackGrant, execution_id: executed.execution_id });
    assert.equal(recovered.status, "rolled_back");
    assert.equal(recovered.reconciled, faultPoint === "after_rollback_effect");
    assert.equal(provider.value, 1_000_000);
    assert.equal(provider.mutationCalls, 2);
    assert.equal(calls.filter((call) => call.tool === "validate_execution_grant" && call.args.grant.grant_id === rollbackGrant.grant_id).length, 1);
  }
});

test("rollback recovers a durable remote claim lost before local recording", async () => {
  let crashed = false;
  const provider = mutableProvider();
  const { ctx, storeRoot, calls } = await fixture({ provider, fault: (point) => { if (point === "after_rollback_control_plane_claim" && !crashed) { crashed = true; throw new Error("rollback claim response lost"); } } });
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const files = await readdir(path.join(storeRoot, "receipts"));
  const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
  const request = rollbackRequest(executed.execution_id, envelope.value);
  const rollbackGrant = grant(plan(), { grant_id: "grant_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
  await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), /rollback claim response lost/);
  assert.equal(provider.value, 900_000);
  const recovered = await rollbackExecution({ ...ctx, fault: () => {} }, { grant: rollbackGrant, execution_id: executed.execution_id });
  assert.equal(recovered.status, "rolled_back");
  assert.equal(provider.mutationCalls, 2);
  assert.equal(calls.filter((call) => call.tool === "get_execution_claim" && call.args.grant.grant_id === rollbackGrant.grant_id).length, 1);
});

test("a locally claimed rollback cannot begin after current authority ends", async () => {
  let crashed = false;
  const provider = mutableProvider();
  const { ctx, storeRoot } = await fixture({ provider, fault: (point) => { if (point === "after_rollback_claim" && !crashed) { crashed = true; throw new Error("rollback paused after claim"); } } });
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const files = await readdir(path.join(storeRoot, "receipts"));
  const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
  const request = rollbackRequest(executed.execution_id, envelope.value);
  const rollbackGrant = grant(plan(), { grant_id: "grant_dddddddddddddddddddddddddddddddd", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
  await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), /rollback paused after claim/);
  const authoritative = ctx.controlPlane;
  ctx.controlPlane = async (tool, args, deadline) => {
    if (tool === "revalidate_execution_claim") throw new ExecutorError("APPROVAL_EXPIRED", "approval expired");
    return authoritative(tool, args, deadline);
  };
  let cancellationCrashed = false;
  ctx.fault = (point) => { if (point === "after_rollback_cancellation_action" && !cancellationCrashed) { cancellationCrashed = true; throw new Error("crash:after_rollback_cancellation_action"); } };
  await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), /crash:after_rollback_cancellation_action/);
  assert.equal(provider.value, 900_000);
  assert.equal(provider.mutationCalls, 1);
  await assert.rejects(() => rollbackExecution({ ...ctx, fault: () => {} }, { grant: rollbackGrant, execution_id: executed.execution_id }), (error) => error.code === "APPROVAL_EXPIRED");
  await assert.rejects(() => rollbackExecution({ ...ctx, fault: () => {} }, { grant: rollbackGrant, execution_id: executed.execution_id }), (error) => error.code === "ROLLBACK_CANCELLED");
});

test("rollback receipt recovery completes terminal recording across both crash windows", async () => {
  for (const faultPoint of ["after_rollback_receipt", "after_rollback_terminal_action"]) {
    let crashed = false;
    const provider = mutableProvider();
    const { ctx, storeRoot, calls } = await fixture({ provider, fault: (point) => { if (point === faultPoint && !crashed) { crashed = true; throw new Error(`crash:${point}`); } } });
    const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
    const files = await readdir(path.join(storeRoot, "receipts"));
    const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
    const request = rollbackRequest(executed.execution_id, envelope.value);
    const rollbackGrant = grant(plan(), { grant_id: faultPoint === "after_rollback_receipt" ? "grant_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" : "grant_cccccccccccccccccccccccccccccccc", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
    await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), /crash:after_rollback/);
    assert.equal(provider.value, 1_000_000);
    const recovered = await rollbackExecution({ ...ctx, fault: () => {} }, { grant: rollbackGrant, execution_id: executed.execution_id });
    assert.equal(recovered.status, "rolled_back");
    assert.equal(provider.mutationCalls, 2);
    assert.ok(calls.some((call) => call.tool === "record_agent_action" && call.args.grant.grant_id === rollbackGrant.grant_id && call.args.status === "executed"));
  }
});

test("ambiguous rollback reconciles restored state and rejects a conflicting third state", async () => {
  for (const finalValue of [1_000_000, 850_000]) {
    const provider = mutableProvider(1_000_000, { mutate: async (_plan, next, state) => {
      if (state.calls === 1) { state.set(next); return { requestID: "forward", responseHash: digest({ next }) }; }
      state.set(finalValue);
      throw new ExecutorError("UPSTREAM_TIMEOUT", "timeout", { retryable: true, ambiguous: true });
    } });
    const { ctx, storeRoot } = await fixture({ provider });
    const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
    const files = await readdir(path.join(storeRoot, "receipts"));
    const envelope = JSON.parse(await readFile(path.join(storeRoot, "receipts", files[0]), "utf8"));
    const request = rollbackRequest(executed.execution_id, envelope.value);
    const rollbackGrant = grant(plan(), { grant_id: finalValue === 1_000_000 ? "grant_88888888888888888888888888888888" : "grant_99999999999999999999999999999999", action_id: "rollback-action", capability: "ad_platform.rollback", inputs_hash: digest(request) });
    await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), (error) => error.code === "RECONCILIATION_REQUIRED");
    if (finalValue === 1_000_000) {
      const recovered = await rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id });
      assert.equal(recovered.status, "rolled_back");
      assert.equal(recovered.reconciled, true);
    } else await assert.rejects(() => rollbackExecution(ctx, { grant: rollbackGrant, execution_id: executed.execution_id }), (error) => error.code === "ROLLBACK_CONFLICT");
    assert.equal(provider.mutationCalls, 2);
  }
});

test("a deleted receipt cannot be projected from its terminal intent", async () => {
  const { ctx, storeRoot } = await fixture();
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const files = await readdir(path.join(storeRoot, "receipts"));
  await unlink(path.join(storeRoot, "receipts", files[0]));
  await assert.rejects(() => getExecution(ctx, { execution_id: executed.execution_id }), (error) => error.code === "EXECUTION_STATE_INVALID");
});

test("authenticated executor records reject coordinated local tampering", async () => {
  const { ctx, storeRoot } = await fixture();
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  const files = await import("node:fs/promises").then(({ readdir }) => readdir(path.join(storeRoot, "receipts")));
  const file = path.join(storeRoot, "receipts", files[0]);
  const envelope = JSON.parse(await readFile(file, "utf8"));
  envelope.value.status = "failed";
  await import("node:fs/promises").then(({ writeFile }) => writeFile(file, JSON.stringify(envelope)));
  await assert.rejects(() => getExecution(ctx, { execution_id: executed.execution_id }), (error) => error.code === "EXECUTION_STATE_INVALID");
});

test("preview is read-only and reports stale/current state without claiming authority", async () => {
  const { ctx, provider, calls } = await fixture();
  const result = await previewChange(ctx, { plan: plan() });
  assert.equal(result.mode, "preview");
  assert.equal(result.can_apply, true);
  assert.equal(provider.mutationCalls, 0);
  assert.equal(calls.length, 0);
});

test("the default control-plane path launches only the receipt-bound pinned package", async () => {
  const packageRoot = await realpath(await mkdtemp(path.join(os.tmpdir(), "control-plane-package-")));
  await mkdir(path.join(packageRoot, "scripts"));
  const server = path.join(packageRoot, "scripts", "agent_control_plane_server.mjs");
  await writeFile(path.join(packageRoot, "tool.yaml"), "schema_version: \"2.0\"\nid: agentops/agent-control-plane-server\nversion: 0.2.1\n");
  await writeFile(server, `let input = ""; process.stdin.on("data", (chunk) => { input += chunk; }); process.stdin.on("end", () => { const request = JSON.parse(input); const name = request.params.name; const grant = request.params.arguments.grant; const structuredContent = name === "validate_execution_grant" ? { valid: true, claim_id: "claim_default_path" } : name === "revalidate_execution_claim" ? { valid: true, current: true, claim_id: "claim_default_path", grant_id: grant.grant_id, inputs_hash: grant.inputs_hash, target: grant.target } : name === "verify_audit_chain" ? { valid: true } : { status: request.params.arguments.status }; process.stdout.write(JSON.stringify({ jsonrpc: "2.0", id: request.id, result: { structuredContent } }) + "\\n"); });\n`);
  const treeSHA256 = await packageTreeSHA256(packageRoot);
  const runtimeContractSHA256 = "c".repeat(64);
  await writeFile(path.join(packageRoot, ".skills-hub-install.json"), `${JSON.stringify({ schema_version: "skills-hub.install-receipt/v1", source: "remote", module: "tools", id: "agentops/agent-control-plane-server", version: "0.2.1", runtime: "generic", artifact_sha256: "d".repeat(64), runtime_contract_sha256: runtimeContractSHA256, tree_sha256: treeSHA256, installed_at: "2026-09-20T12:00:00Z" })}\n`);
  const env = {
    AD_PLATFORM_EXECUTOR_POLICY: JSON.stringify({ environment_tier: "sandbox", allowed_targets: [{ provider: "google_ads", account_id: "1234567890", resource_type: "campaign_budget", resource_ids: ["22222"], operations: ["set_budget_micros"] }] }),
    AD_PLATFORM_PROVIDER_CREDENTIAL: JSON.stringify({ type: "access_token", access_token: "provider-access-token-placeholder", developer_token: "developer-token" }),
    AD_PLATFORM_EXECUTOR_STORE: await mkdtemp(path.join(os.tmpdir(), "default-cp-store-")), AD_PLATFORM_EXECUTOR_AUDIT_KEY: "q".repeat(64),
    CONTROL_PLANE_PACKAGE_ROOT: packageRoot, CONTROL_PLANE_PACKAGE_INTEGRITY: JSON.stringify({ version: "0.2.1", runtime: "generic", tree_sha256: treeSHA256, runtime_contract_sha256: runtimeContractSHA256 }),
    CONTROL_PLANE_STORE: "/tmp/default-cp-state", CONTROL_PLANE_IDENTITY: "{}", CONTROL_PLANE_IDENTITY_PUBLIC_KEY: "public", CONTROL_PLANE_GRANT_ROLE_KEY: "public", CONTROL_PLANE_AUDIT_KEY: "r".repeat(64),
  };
  const provider = mutableProvider();
  const ctx = contextFromEnv(env, { provider, now: () => NOW });
  const executed = await executeApprovedChange(ctx, { grant: grant(), plan: plan() });
  assert.equal(executed.status, "executed");
  assert.equal(provider.mutationCalls, 1);

  await writeFile(server, "process.stdout.write('substituted');\n");
  const substitutedProvider = mutableProvider();
  const substituted = contextFromEnv({ ...env, AD_PLATFORM_EXECUTOR_STORE: await mkdtemp(path.join(os.tmpdir(), "substituted-cp-store-")) }, { provider: substitutedProvider, now: () => NOW });
  await assert.rejects(() => executeApprovedChange(substituted, { grant: grant(), plan: plan() }), (error) => error.code === "CONTROL_PLANE_INTEGRITY_INVALID");
  assert.equal(substitutedProvider.mutationCalls, 0);
});

test("packaged auth driver binds the signed executor identity to a live read-only provider target", async () => {
  const store = await mkdtemp(path.join(os.tmpdir(), "executor-auth-"));
  const env = {
    AD_PLATFORM_EXECUTOR_POLICY: JSON.stringify({ environment_tier: "sandbox", allowed_targets: [{ provider: "google_ads", account_id: "1234567890", resource_type: "campaign_budget", resource_ids: ["22222"], operations: ["set_budget_micros"] }] }),
    AD_PLATFORM_PROVIDER_CREDENTIAL: JSON.stringify({ type: "access_token", access_token: "provider-access-token-placeholder" }),
    AD_PLATFORM_EXECUTOR_STORE: store, AD_PLATFORM_EXECUTOR_AUDIT_KEY: "k".repeat(64), CONTROL_PLANE_PACKAGE_ROOT: "/tmp/control-plane", CONTROL_PLANE_PACKAGE_INTEGRITY: TEST_CONTROL_PLANE_INTEGRITY,
    CONTROL_PLANE_STORE: "/tmp/control-plane-store", CONTROL_PLANE_IDENTITY: JSON.stringify({ actor_id: "executor-1", roles: ["executor_runtime"], expires_at: "2026-09-20T13:00:00.000Z" }), CONTROL_PLANE_IDENTITY_PUBLIC_KEY: "public", CONTROL_PLANE_GRANT_ROLE_KEY: "public", CONTROL_PLANE_AUDIT_KEY: "a".repeat(64),
  };
  const provider = mutableProvider();
  const options = { env, provider, now: () => NOW, controlPlane: async (tool) => { assert.equal(tool, "verify_audit_chain"); return { valid: true }; } };
  const bindings = ["AD_PLATFORM_PROVIDER_CREDENTIAL", "AD_PLATFORM_EXECUTOR_POLICY", "AD_PLATFORM_EXECUTOR_STORE", "AD_PLATFORM_EXECUTOR_AUDIT_KEY", "CONTROL_PLANE_PACKAGE_ROOT", "CONTROL_PLANE_PACKAGE_INTEGRITY", "CONTROL_PLANE_STORE", "CONTROL_PLANE_IDENTITY", "CONTROL_PLANE_IDENTITY_PUBLIC_KEY", "CONTROL_PLANE_GRANT_ROLE_KEY", "CONTROL_PLANE_AUDIT_KEY"];
  const bootstrap = await runDriver("custom", "bootstrap", { method: "custom", flow: "custom", credential_bindings: bindings }, options);
  assert.deepEqual(bootstrap, { completed: true, account: "google_ads:1234567890", scopes: ["set_budget_micros"] });
  const status = await runDriver("custom", "status", {}, options);
  assert.equal(status.authenticated, true);
  assert.equal(status.principal, "executor-1");
  assert.equal(status.account, "google_ads:1234567890");
  assert.equal(status.tier, "sandbox");
  assert.match(status.attestation_reference, /^control-plane-provider:[a-f0-9]{64}$/);
  assert.ok(!JSON.stringify(status).includes("provider-access-token-placeholder"));
});

test("canonical Google Ads and DV360 adapters constrain paths, methods, and update masks", async () => {
  const requests = [];
  let googleBudget = "1000000";
  let dvStatus = "ENTITY_STATUS_PAUSED";
  const fetchImpl = async (url, init) => {
    requests.push({ url: String(url), init });
    if (String(url).includes("googleAds:searchStream")) return new Response(JSON.stringify([{ results: [{ campaignBudget: { resourceName: "customers/1234567890/campaignBudgets/22222", amountMicros: googleBudget } }] }]), { status: 200, headers: { "content-type": "application/json" } });
    if (String(url).includes("campaignBudgets:mutate")) { googleBudget = "900000"; return new Response(JSON.stringify({ results: [{ resourceName: "customers/1234567890/campaignBudgets/22222" }] }), { status: 200, headers: { "request-id": "ghp_secret-provider-header" } }); }
    if (init.method === "GET") return new Response(JSON.stringify({ advertiserId: "987654", lineItemId: "456789", entityStatus: dvStatus }), { status: 200 });
    dvStatus = "ENTITY_STATUS_ACTIVE";
    return new Response(JSON.stringify({ advertiserId: "987654", lineItemId: "456789", entityStatus: dvStatus }), { status: 200, headers: { "x-request-id": "dv-1" } });
  };
  const baseEnv = {
    AD_PLATFORM_EXECUTOR_POLICY: JSON.stringify({ environment_tier: "sandbox", allowed_targets: [
      { provider: "google_ads", account_id: "1234567890", resource_type: "campaign_budget", resource_ids: ["22222"], operations: ["set_budget_micros"] },
      { provider: "dv360", account_id: "987654", resource_type: "line_item", resource_ids: ["456789"], operations: ["set_status"] },
    ] }),
    AD_PLATFORM_PROVIDER_CREDENTIAL: JSON.stringify({ type: "access_token", access_token: "token-value-that-is-long-enough", developer_token: "developer-token" }),
    AD_PLATFORM_EXECUTOR_STORE: await mkdtemp(path.join(os.tmpdir(), "adapter-store-")), AD_PLATFORM_EXECUTOR_AUDIT_KEY: "z".repeat(64), CONTROL_PLANE_PACKAGE_ROOT: "/tmp/control-plane", CONTROL_PLANE_PACKAGE_INTEGRITY: TEST_CONTROL_PLANE_INTEGRITY,
  };
  const googleCtx = contextFromEnv(baseEnv, { fetchImpl, now: () => NOW, controlPlane: async (tool, args) => tool === "validate_execution_grant" ? { claim_id: "claim_google" } : tool === "revalidate_execution_claim" ? { valid: true, current: true, claim_id: "claim_google", grant_id: args.grant.grant_id, inputs_hash: args.grant.inputs_hash, target: args.grant.target } : { status: args.status } });
  const googleResult = await executeApprovedChange(googleCtx, { grant: grant(), plan: plan() });
  assert.match(googleResult.provider_receipt, /^google-ads-request:sha256:[a-f0-9]{64}:sha256:[a-f0-9]{64}$/);
  assert.equal(JSON.stringify(googleResult).includes("ghp_secret-provider-header"), false);
  assert.match(requests[0].url, /\/v25\/customers\/1234567890\/googleAds:searchStream$/);
  assert.match(requests[1].url, /\/v25\/customers\/1234567890\/campaignBudgets:mutate$/);
  assert.equal(JSON.parse(requests[1].init.body).operations[0].updateMask, "amount_micros");
  const dvPlan = plan({ provider: "dv360", account_id: "987654", resource_type: "line_item", resource_id: "456789", operation: "set_status", proposed_diff: { field: "status", from: "PAUSED", to: "ENABLED" } });
  const dvGrant = grant(dvPlan, { grant_id: "grant_44444444444444444444444444444444" });
  const dvCtx = contextFromEnv({ ...baseEnv, AD_PLATFORM_EXECUTOR_STORE: await mkdtemp(path.join(os.tmpdir(), "adapter-dv-store-")) }, { fetchImpl, now: () => NOW, controlPlane: async (tool, args) => tool === "validate_execution_grant" ? { claim_id: "claim_dv" } : tool === "revalidate_execution_claim" ? { valid: true, current: true, claim_id: "claim_dv", grant_id: args.grant.grant_id, inputs_hash: args.grant.inputs_hash, target: args.grant.target } : { status: args.status } });
  await executeApprovedChange(dvCtx, { grant: dvGrant, plan: dvPlan });
  const patch = requests.find((item) => item.init.method === "PATCH");
  assert.match(patch.url, /\/v4\/advertisers\/987654\/lineItems\/456789\?updateMask=entityStatus$/);
});

test("MCP protocol exposes only bounded structured execution tools", async () => {
  const initialized = await handleRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} });
  assert.equal(initialized.serverInfo.name, "governed-ad-platform-executor");
  const listed = await handleRequest({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} });
  assert.deepEqual(listed.tools.map((item) => item.name), ["preview_ad_platform_change", "execute_approved_change", "reconcile_ambiguous_execution", "rollback_execution", "get_execution"]);
  assert.equal(listed.tools.some((item) => /raw|request|http/i.test(item.name)), false);
});

test("launchable stdio server completes initialize and tools/list", async () => {
  const script = fileURLToPath(new URL("../scripts/ad_platform_executor_server.mjs", import.meta.url));
  const child = spawn(process.execPath, [script], { stdio: ["pipe", "pipe", "pipe"] });
  const responses = [];
  const lines = createInterface({ input: child.stdout, crlfDelay: Infinity });
  lines.on("line", (line) => responses.push(JSON.parse(line)));
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`);
  child.stdin.end();
  await once(child, "exit");
  assert.equal(child.exitCode, 0);
  assert.equal(responses[0].result.serverInfo.name, "governed-ad-platform-executor");
  assert.equal(responses[1].result.tools.length, 5);
});

test("plan canonicalization is stable and unknown provider fields cannot influence authority", () => {
  const validated = validatePlan(plan(), { now: () => NOW });
  assert.equal(digest(validated), digest(JSON.parse(canonicalJSON(validated))));
  assert.throws(() => validatePlan({ ...plan(), proposed_diff: { ...plan().proposed_diff, hidden: true } }, { now: () => NOW }), (error) => error.code === "INVALID_ARGUMENT");
});
