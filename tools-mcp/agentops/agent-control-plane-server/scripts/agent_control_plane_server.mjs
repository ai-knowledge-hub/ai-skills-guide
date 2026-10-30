#!/usr/bin/env node

import { createInterface } from "node:readline";
import { pathToFileURL } from "node:url";

import {
  ControlPlaneError,
  authorizeAction,
  checkPolicy,
  contextFromEnv,
  getApproval,
  getExecutionClaim,
  publicError,
  recordAction,
  revalidateExecutionClaim,
  requestApproval,
  validateExecutionGrant,
  verifyAuditChain,
} from "./control_plane_core.mjs";

const MAX_LINE_BYTES = 256 * 1024;

const tools = Object.freeze([
  {
    name: "check_policy",
    description: "Evaluate an exact proposed action against the active tenant policy.",
    inputSchema: {
      type: "object", additionalProperties: false,
      required: ["action_id", "agent_run_id", "capability", "capability_version", "risk_tier", "target", "inputs_hash"],
      properties: {
        action_id: { type: "string" }, agent_run_id: { type: "string" }, capability: { type: "string" }, capability_version: { type: "string" },
        risk_tier: { enum: ["low", "medium", "high"] }, inputs_hash: { type: "string", pattern: "^sha256:[a-f0-9]{64}$" },
        target: { type: "object", additionalProperties: false, required: ["provider", "account_id", "resource_type", "resource_id"], properties: { provider: { type: "string" }, account_id: { type: "string" }, resource_type: { type: "string" }, resource_id: { type: "string" } } },
      },
    },
  },
  {
    name: "request_approval",
    description: "Create an immutable approval request for a policy decision that requires human approval.",
    inputSchema: { type: "object", additionalProperties: false, required: ["decision_id", "justification"], properties: { decision_id: { type: "string" }, justification: { type: "string", minLength: 1, maxLength: 2000 } } },
  },
  {
    name: "get_approval",
    description: "Read the authoritative status of an approval requested by this runtime identity.",
    inputSchema: { type: "object", additionalProperties: false, required: ["approval_id"], properties: { approval_id: { type: "string" } } },
  },
  {
    name: "authorize_action",
    description: "Issue a short-lived signed execution grant after revalidating policy and any required approval.",
    inputSchema: { type: "object", additionalProperties: false, required: ["decision_id"], properties: { decision_id: { type: "string" }, approval_id: { type: "string" } } },
  },
  {
    name: "record_agent_action",
    description: "Append an idempotent execution lifecycle record bound to a signed execution grant.",
    inputSchema: { type: "object", additionalProperties: false, required: ["grant", "status", "outputs_hash", "provider_receipt"], properties: { grant: { type: "object" }, status: { enum: ["executing", "executed", "failed", "cancelled"] }, outputs_hash: { type: ["string", "null"] }, provider_receipt: { type: "string", minLength: 1, maxLength: 512 } } },
  },
  {
    name: "validate_execution_grant",
    description: "Revalidate a signed grant, active policy, approval, executor scope, and replay state immediately before an external effect.",
    inputSchema: { type: "object", additionalProperties: false, required: ["grant"], properties: { grant: { type: "object" } } },
  },
  {
    name: "get_execution_claim",
    description: "Recover the exact durable effect-boundary claim for the same executor and signed grant after local process loss.",
    inputSchema: { type: "object", additionalProperties: false, required: ["grant"], properties: { grant: { type: "object" } } },
  },
  {
    name: "revalidate_execution_claim",
    description: "Recheck current grant, policy, approval, identity, and scope for an existing exact claim without creating another claim.",
    inputSchema: { type: "object", additionalProperties: false, required: ["grant"], properties: { grant: { type: "object" } } },
  },
  {
    name: "verify_audit_chain",
    description: "Verify sequence, tenant binding, previous hashes, and event hashes for the complete tenant audit chain.",
    inputSchema: { type: "object", additionalProperties: false },
  },
]);

function validID(id) {
  return (typeof id === "string" && id.length <= 256) || (Number.isSafeInteger(id) && Math.abs(id) <= Number.MAX_SAFE_INTEGER);
}

function responseResult(result) {
  const text = JSON.stringify(result);
  return { content: [{ type: "text", text }], structuredContent: result, isError: false };
}

function responseError(error) {
  const safe = publicError(error);
  return { content: [{ type: "text", text: JSON.stringify({ error: safe }) }], structuredContent: { error: safe }, isError: true };
}

export async function handleRequest(request, options = {}) {
  if (!request || typeof request !== "object" || Array.isArray(request) || request.jsonrpc !== "2.0" || (request.id !== undefined && !validID(request.id)) || typeof request.method !== "string") {
    throw new ControlPlaneError("INVALID_REQUEST", "invalid JSON-RPC request");
  }
  if (request.method === "initialize") return { protocolVersion: request.params?.protocolVersion ?? "2024-11-05", capabilities: { tools: {} }, serverInfo: { name: "agent-control-plane-server", version: "0.2.1" } };
  if (request.method === "notifications/initialized") return null;
  if (request.method === "ping") return {};
  if (request.method === "tools/list") return { tools };
  if (request.method !== "tools/call" || typeof request.params?.name !== "string") throw new ControlPlaneError("METHOD_NOT_FOUND", "unsupported MCP method");
  const ctx = options.context ?? contextFromEnv(options.env, { now: options.now, fault: options.fault });
  const args = request.params.arguments ?? {};
  try {
    switch (request.params.name) {
      case "check_policy": return responseResult(await checkPolicy(ctx, args));
      case "request_approval": return responseResult(await requestApproval(ctx, args));
      case "get_approval": return responseResult(await getApproval(ctx, args));
      case "authorize_action": return responseResult(await authorizeAction(ctx, args));
      case "record_agent_action": return responseResult(await recordAction(ctx, args));
      case "validate_execution_grant": return responseResult(await validateExecutionGrant(ctx, args));
      case "get_execution_claim": return responseResult(await getExecutionClaim(ctx, args));
      case "revalidate_execution_claim": return responseResult(await revalidateExecutionClaim(ctx, args));
      case "verify_audit_chain": return responseResult(await verifyAuditChain(ctx));
      default: throw new ControlPlaneError("TOOL_NOT_FOUND", "unknown control-plane tool");
    }
  } catch (error) { return responseError(error); }
}

export async function runServer({ input = process.stdin, output = process.stdout, errorOutput = process.stderr } = {}) {
  const lines = createInterface({ input, crlfDelay: Infinity });
  for await (const line of lines) {
    if (Buffer.byteLength(line, "utf8") > MAX_LINE_BYTES) {
      output.write(`${JSON.stringify({ jsonrpc: "2.0", id: null, error: { code: -32600, message: "request exceeds the safe size limit" } })}\n`);
      continue;
    }
    let request;
    try { request = JSON.parse(line); } catch {
      output.write(`${JSON.stringify({ jsonrpc: "2.0", id: null, error: { code: -32700, message: "parse error" } })}\n`);
      continue;
    }
    try {
      const result = await handleRequest(request);
      if (request.id !== undefined && result !== null) output.write(`${JSON.stringify({ jsonrpc: "2.0", id: request.id, result })}\n`);
    } catch (error) {
      const safe = publicError(error);
      if (request.id !== undefined) output.write(`${JSON.stringify({ jsonrpc: "2.0", id: request.id, error: { code: -32600, message: safe.message, data: safe } })}\n`);
    }
  }
  errorOutput.write("");
}

async function main() {
  if (process.argv.includes("--healthcheck")) {
    process.stdout.write(`${JSON.stringify({ status: "ok", server: "agent-control-plane-server" })}\n`);
    return;
  }
  if (process.argv.includes("--smoke")) {
    const initialized = await handleRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} });
    if (initialized.serverInfo?.name !== "agent-control-plane-server") throw new Error("MCP initialization failed");
    process.stdout.write(`${JSON.stringify({ ok: true, protocol: "mcp" })}\n`);
    return;
  }
  await runServer();
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) {
  main().catch((error) => {
    process.stderr.write(`${JSON.stringify(publicError(error))}\n`);
    process.exitCode = 1;
  });
}
