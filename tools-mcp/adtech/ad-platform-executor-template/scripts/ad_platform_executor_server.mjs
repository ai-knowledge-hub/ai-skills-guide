#!/usr/bin/env node

import { createInterface } from "node:readline";
import { pathToFileURL } from "node:url";

import {
  ExecutorError,
  assertWireBudget,
  contextFromEnv,
  executeApprovedChange,
  getExecution,
  previewChange,
  publicError,
  reconcileExecution,
  rollbackExecution,
} from "./ad_platform_executor_core.mjs";

const MAX_LINE_BYTES = 256 * 1024;

const tools = Object.freeze([
  {
    name: "preview_ad_platform_change",
    description: "Read current provider state and compare it with a closed structured plan without claiming a grant or mutating the provider.",
    inputSchema: planInput(false),
  },
  {
    name: "execute_approved_change",
    description: "Claim an exact control-plane grant, apply one bounded provider mutation, verify resulting state, and persist an authenticated receipt.",
    inputSchema: planInput(true),
  },
  {
    name: "reconcile_ambiguous_execution",
    description: "Re-read provider state after an ambiguous effect and record either the verified effect or verified absence of effect.",
    inputSchema: planInput(true),
  },
  {
    name: "rollback_execution",
    description: "Use a separately authorized rollback grant to restore the verified pre-image of one completed execution.",
    inputSchema: {
      type: "object", additionalProperties: false, required: ["grant", "execution_id"],
      properties: { grant: { type: "object" }, execution_id: { type: "string" }, timeout_ms: { type: "integer", minimum: 1000, maximum: 120000 } },
    },
  },
  {
    name: "get_execution",
    description: "Read redacted authenticated executor state for one execution identifier.",
    inputSchema: { type: "object", additionalProperties: false, required: ["execution_id"], properties: { execution_id: { type: "string" } } },
  },
]);

function planInput(grant) {
  return {
    type: "object", additionalProperties: false, required: grant ? ["grant", "plan"] : ["plan"],
    properties: { grant: { type: "object" }, plan: { type: "object" }, timeout_ms: { type: "integer", minimum: 1000, maximum: 120000 } },
  };
}

function validID(id) {
  return (typeof id === "string" && id.length <= 256) || (Number.isSafeInteger(id) && Math.abs(id) <= Number.MAX_SAFE_INTEGER);
}

function responseResult(value) {
  const text = JSON.stringify(value);
  return assertWireBudget({ content: [{ type: "text", text }], structuredContent: value, isError: false });
}

function responseError(error) {
  const safe = publicError(error);
  return { content: [{ type: "text", text: JSON.stringify({ error: safe }) }], structuredContent: { error: safe }, isError: true };
}

export async function handleRequest(request, options = {}) {
  if (!request || typeof request !== "object" || Array.isArray(request) || request.jsonrpc !== "2.0" || (request.id !== undefined && !validID(request.id)) || typeof request.method !== "string") throw new ExecutorError("INVALID_REQUEST", "invalid JSON-RPC request");
  if (request.method === "initialize") return { protocolVersion: request.params?.protocolVersion ?? "2024-11-05", capabilities: { tools: {} }, serverInfo: { name: "governed-ad-platform-executor", version: "0.2.0" } };
  if (request.method === "notifications/initialized") return null;
  if (request.method === "ping") return {};
  if (request.method === "tools/list") return { tools };
  if (request.method !== "tools/call" || typeof request.params?.name !== "string") throw new ExecutorError("METHOD_NOT_FOUND", "unsupported MCP method");
  const ctx = options.context ?? contextFromEnv(options.env, options);
  const args = request.params.arguments ?? {};
  try {
    switch (request.params.name) {
      case "preview_ad_platform_change": return responseResult(await previewChange(ctx, args));
      case "execute_approved_change": return responseResult(await executeApprovedChange(ctx, args));
      case "reconcile_ambiguous_execution": return responseResult(await reconcileExecution(ctx, args));
      case "rollback_execution": return responseResult(await rollbackExecution(ctx, args));
      case "get_execution": return responseResult(await getExecution(ctx, args));
      default: throw new ExecutorError("TOOL_NOT_FOUND", "unknown executor tool");
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
    try { request = JSON.parse(line); }
    catch { output.write(`${JSON.stringify({ jsonrpc: "2.0", id: null, error: { code: -32700, message: "parse error" } })}\n`); continue; }
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
  if (process.argv.includes("--healthcheck")) { process.stdout.write(`${JSON.stringify({ status: "ok", server: "governed-ad-platform-executor" })}\n`); return; }
  if (process.argv.includes("--smoke")) {
    const initialized = await handleRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} });
    if (initialized.serverInfo?.name !== "governed-ad-platform-executor") throw new Error("MCP initialization failed");
    process.stdout.write(`${JSON.stringify({ ok: true, protocol: "mcp" })}\n`);
    return;
  }
  await runServer();
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? "").href) main().catch((error) => { process.stderr.write(`${JSON.stringify(publicError(error))}\n`); process.exitCode = 1; });
