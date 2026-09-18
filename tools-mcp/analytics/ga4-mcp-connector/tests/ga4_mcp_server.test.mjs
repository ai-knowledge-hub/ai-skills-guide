import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { generateKeyPairSync } from "node:crypto";
import { once } from "node:events";
import { createInterface } from "node:readline";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  GA4Error,
  handleMCPRequest,
  parseCredentialEnvelope,
  runReport,
  validateRunReportInput,
} from "../scripts/ga4_mcp_server.mjs";
import { runDriver } from "../scripts/ga4_auth_driver.mjs";

const credential = JSON.stringify({
  type: "access_token",
  access_token: "test-access-token-value",
  expires_at: "2099-01-01T00:00:00Z",
  account: "123456789",
});

function env(overrides = {}) {
  return {
    GA4_CREDENTIAL: credential,
    GA4_ALLOWED_PROPERTY_IDS: "123456789",
    ...overrides,
  };
}

function jsonResponse(payload, init = {}) {
  return new Response(JSON.stringify(payload), {
    status: init.status ?? 200,
    headers: { "content-type": "application/json", ...(init.headers ?? {}) },
  });
}

function reportPage(values, rowCount = values.length) {
  return {
    rowCount,
    rows: values.map(([dimension, metric]) => ({
      dimensionValues: dimension === undefined ? [] : [{ value: dimension }],
      metricValues: [{ value: metric }],
    })),
    propertyQuota: {},
  };
}

function tokenInfo(overrides = {}) {
  return {
    email: "provider-verified@example.test",
    scope: "https://www.googleapis.com/auth/analytics.readonly",
    expires_in: "3600",
    ...overrides,
  };
}

const baseInput = {
  property_id: "123456789",
  start_date: "2026-01-01",
  end_date: "2026-01-07",
  dimensions: ["sessionDefaultChannelGroup"],
  metrics: ["sessions"],
};

test("validation rejects cross-property, unbounded, and unknown fields before fetch", () => {
  assert.throws(
    () => validateRunReportInput({ ...baseInput, property_id: "999999999" }, env()),
    (error) => error instanceof GA4Error && error.code === "PROPERTY_NOT_ALLOWED",
  );
  assert.throws(
    () => validateRunReportInput({ ...baseInput, start_date: "2020-01-01" }, env()),
    (error) => error.code === "DATE_RANGE_NOT_ALLOWED",
  );
  assert.throws(
    () => validateRunReportInput({ ...baseInput, metrics: ["customMetric"] }, env()),
    (error) => error.code === "INVALID_ARGUMENT",
  );
  assert.throws(
    () => validateRunReportInput({ ...baseInput, secret: "do-not-forward" }, env()),
    (error) => error.code === "INVALID_ARGUMENT",
  );
});

test("runReport paginates within budgets and returns normalized freshness", async () => {
  const requests = [];
  const pages = [
    reportPage([["Organic Search", "7"], ["Direct", "4"]], 3),
    reportPage([["Referral", "2"]], 3),
  ];
  const result = await runReport({ ...baseInput, page_size: 2, max_rows: 10 }, {
    env: env(),
    now: () => new Date("2026-09-17T10:00:00Z"),
    fetchImpl: async (url, options) => {
      requests.push({ url: String(url), body: JSON.parse(options.body), authorization: options.headers.authorization });
      return jsonResponse(pages.shift());
    },
  });
  assert.equal(requests.length, 2);
  assert.deepEqual(requests.map((request) => request.body.offset), ["0", "2"]);
  assert.ok(requests.every((request) => request.authorization === "Bearer test-access-token-value"));
  assert.equal(result.rows[2].sessions, "2");
  assert.equal(result.pagination.returned_rows, 3);
  assert.equal(result.source.retrieved_at, "2026-09-17T10:00:00.000Z");
});

test("runReport reports truncation at the caller row limit", async () => {
  const result = await runReport({ ...baseInput, page_size: 2, max_rows: 2 }, {
    env: env(),
    fetchImpl: async () => jsonResponse(reportPage([["Organic Search", "7"], ["Direct", "4"]], 20)),
  });
  assert.equal(result.rows.length, 2);
  assert.equal(result.pagination.truncated, true);
});

test("runReport enforces one aggregate response-size limit across pages", async () => {
  let calls = 0;
  const oversizedAcrossPages = "x".repeat(700_000);
  await assert.rejects(
    () => runReport({ ...baseInput, page_size: 1, max_rows: 3 }, {
      env: env(),
      fetchImpl: async () => {
        calls += 1;
        return jsonResponse(reportPage([[oversizedAcrossPages, "1"]], 3));
      },
    }),
    (error) => error.code === "RESULT_TOO_LARGE",
  );
  assert.equal(calls, 3);
});

test("runReport rejects an incomplete page sequence that contradicts rowCount", async () => {
  await assert.rejects(
    () => runReport({ ...baseInput, page_size: 2, max_rows: 10 }, {
      env: env(), fetchImpl: async () => jsonResponse(reportPage([], 3)),
    }),
    (error) => error.code === "UPSTREAM_MALFORMED",
  );
});

test("quota, revoked credentials, and malformed responses are normalized", async (t) => {
  const cases = [
    ["quota", jsonResponse({ error: { status: "RESOURCE_EXHAUSTED" } }, { status: 429 }), "QUOTA_EXHAUSTED", true],
    ["revoked", jsonResponse({ error: { status: "UNAUTHENTICATED" } }, { status: 401 }), "AUTH_REVOKED", false],
    ["malformed", new Response("not-json", { status: 200 }), "UPSTREAM_MALFORMED", false],
  ];
  for (const [name, response, code, retryable] of cases) {
    await t.test(name, async () => {
      await assert.rejects(
        () => runReport(baseInput, { env: env(), fetchImpl: async () => response }),
        (error) => error.code === code && error.retryable === retryable && !error.message.includes("test-access-token-value"),
      );
    });
  }
});

test("request timeout is bounded and normalized", async () => {
  await assert.rejects(
    () => runReport(baseInput, {
      env: env({ GA4_REQUEST_TIMEOUT_MS: "10" }),
      fetchImpl: (_url, options) => new Promise((_resolve, reject) => {
        options.signal.addEventListener("abort", () => reject(Object.assign(new Error("aborted"), { name: "AbortError" })));
      }),
    }),
    (error) => error.code === "UPSTREAM_TIMEOUT" && error.retryable,
  );
});

test("MCP initialize, list, and call preserve protocol shape", async () => {
  const initialize = await handleMCPRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05" } });
  assert.equal(initialize.serverInfo.name, "ga4-mcp-connector");
  assert.equal(initialize.protocolVersion, "2024-11-05");
  const listed = await handleMCPRequest({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} });
  assert.deepEqual(listed.tools.map((tool) => tool.name), ["ga4_run_report"]);
  const called = await handleMCPRequest({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "ga4_run_report", arguments: baseInput } }, {
    env: env(), fetchImpl: async () => jsonResponse(reportPage([["Organic Search", "7"]], 1)),
  });
  assert.equal(called.isError, false);
  assert.equal(called.structuredContent.rows[0].sessions, "7");
  await assert.rejects(
    () => handleMCPRequest({ jsonrpc: "2.0", id: "x".repeat(257), method: "ping" }),
    (error) => error.code === "INVALID_REQUEST",
  );
});

test("MCP wire response enforces one budget across text and structured content", async () => {
  const request = {
    jsonrpc: "2.0",
    id: "wire-budget-regression",
    method: "tools/call",
    params: { name: "ga4_run_report", arguments: { ...baseInput, page_size: 1, max_rows: 1 } },
  };
  const invoke = (dimensionSize) => handleMCPRequest(request, {
    env: env(),
    fetchImpl: async () => jsonResponse(reportPage([["x".repeat(dimensionSize), "1"]], 1)),
  });

  const accepted = await invoke(700_000);
  assert.equal(accepted.isError, false);
  const acceptedWire = `${JSON.stringify({ jsonrpc: "2.0", id: request.id, result: accepted })}\n`;
  assert.ok(Buffer.byteLength(acceptedWire, "utf8") <= 2 * 1024 * 1024);

  const rejected = await invoke(1_050_000);
  assert.equal(rejected.isError, true);
  assert.equal(rejected.structuredContent.error.code, "RESULT_TOO_LARGE");
  const rejectedWire = `${JSON.stringify({ jsonrpc: "2.0", id: request.id, result: rejected })}\n`;
  assert.ok(Buffer.byteLength(rejectedWire, "utf8") <= 2 * 1024 * 1024);
});

test("credential envelopes cannot redirect token exchange to an arbitrary host", async () => {
  let contacted = false;
  const malicious = JSON.stringify({
    type: "authorized_user", client_id: "example-client-id", client_secret: "example-client-secret",
    refresh_token: "example-refresh-token", token_uri: "https://attacker.example/token",
    account: "123456789",
  });
  await assert.rejects(
    () => runReport(baseInput, { env: env({ GA4_CREDENTIAL: malicious }), fetchImpl: async () => { contacted = true; return jsonResponse({}); } }),
    (error) => error.code === "CONFIGURATION_ERROR" && !error.message.includes("attacker.example"),
  );
  assert.equal(contacted, false);
});

test("authorized-user refresh and service-account assertion flows use canonical token exchange", async (t) => {
  const scenarios = [];
  scenarios.push(["authorized user", {
    type: "authorized_user", client_id: "example-client-id", client_secret: "example-client-secret",
    refresh_token: "example-refresh-token", account: "123456789",
  }, "refresh_token"]);
  const { privateKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
  scenarios.push(["service account", {
    type: "service_account", client_email: "reader@example-project.iam.gserviceaccount.com",
    private_key: privateKey.export({ type: "pkcs8", format: "pem" }), private_key_id: "example-key-id",
    project_id: "example-project", account: "123456789",
  }, "urn:ietf:params:oauth:grant-type:jwt-bearer"]);
  for (const [name, envelope, grantType] of scenarios) {
    await t.test(name, async () => {
      const calls = [];
      const result = await runReport(baseInput, {
        env: env({ GA4_CREDENTIAL: JSON.stringify(envelope) }),
        now: () => new Date("2026-09-17T10:00:00Z"),
        fetchImpl: async (url, options) => {
          calls.push({ url: String(url), options });
          if (calls.length === 1) return jsonResponse({ access_token: "exchanged-access-token", expires_in: 3600 });
          return jsonResponse(reportPage([["Direct", "2"]], 1));
        },
      });
      assert.equal(calls[0].url, "https://oauth2.googleapis.com/token");
      assert.equal(calls[0].options.body.get("grant_type"), grantType);
      assert.equal(calls[1].options.headers.authorization, "Bearer exchanged-access-token");
      assert.equal(result.rows[0].sessions, "2");
    });
  }
});

test("MCP tool failures are structured and do not expose credentials", async () => {
  const called = await handleMCPRequest({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "ga4_run_report", arguments: baseInput } }, {
    env: env(), fetchImpl: async () => jsonResponse({ error: { message: "token invalid", status: "UNAUTHENTICATED" } }, { status: 401 }),
  });
  assert.equal(called.isError, true);
  assert.equal(called.structuredContent.error.code, "AUTH_REVOKED");
  assert.ok(!JSON.stringify(called).includes("test-access-token-value"));
});

test("packaged auth driver live-verifies bearer identity without storing it", async () => {
  const options = {
    env: env(),
    fetchImpl: async (url) => String(url).startsWith("https://oauth2.googleapis.com/tokeninfo?")
      ? jsonResponse(tokenInfo())
      : jsonResponse(reportPage([], 0)),
    now: () => new Date("2026-09-17T10:00:00Z"),
  };
    const bootstrap = await runDriver("bearer-token", "bootstrap", {
      method: "bearer-token",
      flow: "credential-binding",
      requested_scopes: ["https://www.googleapis.com/auth/analytics.readonly"],
      credential_bindings: ["GA4_CREDENTIAL"],
      expected_account: "123456789",
  }, options);
  assert.equal(bootstrap.completed, true);
  assert.equal(bootstrap.account, "123456789");
  const status = await runDriver("bearer-token", "status", { method: "bearer-token" }, options);
  assert.equal(status.authenticated, true);
  assert.equal(status.principal, "provider-verified@example.test");
  assert.deepEqual(status.scopes, ["https://www.googleapis.com/auth/analytics.readonly"]);
  assert.ok(!JSON.stringify(status).includes("test-access-token-value"));
});

test("service-account readiness accepts provider-attested aud and azp identity", async () => {
  const { privateKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
  const serviceCredential = JSON.stringify({
    type: "service_account",
    client_email: "reader@example-project.iam.gserviceaccount.com",
    private_key: privateKey.export({ type: "pkcs8", format: "pem" }),
    private_key_id: "example-key-id",
    project_id: "example-project",
    account: "123456789",
  });
  const calls = [];
  const status = await runDriver("service-account", "status", {}, {
    env: env({ GA4_CREDENTIAL: serviceCredential }),
    fetchImpl: async (url) => {
      calls.push(String(url));
      if (String(url) === "https://oauth2.googleapis.com/token") {
        return jsonResponse({ access_token: "service-access-token", expires_in: 3600 });
      }
      if (String(url).startsWith("https://oauth2.googleapis.com/tokeninfo?")) {
        return jsonResponse(tokenInfo({
          email: undefined,
          aud: "service-client-123.apps.googleusercontent.com",
          azp: "service-client-123.apps.googleusercontent.com",
        }));
      }
      return jsonResponse(reportPage([], 0));
    },
    now: () => new Date("2026-09-17T10:00:00Z"),
  });
  assert.deepEqual(calls.map((url) => new URL(url).pathname), ["/token", "/tokeninfo", "/v1beta/properties/123456789:runReport"]);
  assert.equal(status.authenticated, true);
  assert.equal(status.principal, "google-service-account:service-client-123.apps.googleusercontent.com");
  assert.deepEqual(status.scopes, ["https://www.googleapis.com/auth/analytics.readonly"]);
});

test("bearer readiness does not reinterpret client audience as user identity", async () => {
  await assert.rejects(
    () => runDriver("bearer-token", "status", {}, {
      env: env(),
      fetchImpl: async (url) => String(url).startsWith("https://oauth2.googleapis.com/tokeninfo?")
        ? jsonResponse(tokenInfo({ email: undefined, aud: "oauth-client.apps.googleusercontent.com", azp: "oauth-client.apps.googleusercontent.com" }))
        : jsonResponse(reportPage([], 0)),
    }),
    (error) => error.code === "AUTH_IDENTITY_UNVERIFIED",
  );
});

test("credential envelope cannot self-attest principal or scopes", () => {
  for (const untrusted of [
    { principal: "impersonated@example.test" },
    { scopes: ["https://www.googleapis.com/auth/analytics.readonly"] },
  ]) {
    assert.throws(
      () => parseCredentialEnvelope(JSON.stringify({
        type: "access_token",
        access_token: "test-access-token-value",
        account: "123456789",
        ...untrusted,
      })),
      (error) => error instanceof GA4Error && error.code === "INVALID_ARGUMENT",
    );
  }
});

test("auth driver rejects a token whose provider-attested scopes omit GA4 read access", async () => {
  let reportCalled = false;
  await assert.rejects(
    () => runDriver("bearer-token", "status", {}, {
      env: env(),
      fetchImpl: async (url) => {
        if (String(url).startsWith("https://oauth2.googleapis.com/tokeninfo?")) {
          return jsonResponse(tokenInfo({ scope: "openid email" }));
        }
        reportCalled = true;
        return jsonResponse(reportPage([], 0));
      },
    }),
    (error) => error.code === "AUTH_SCOPE_INVALID",
  );
  assert.equal(reportCalled, false);
});

test("launchable stdio process completes initialize and tools/list", async () => {
  const child = spawn(process.execPath, [fileURLToPath(new URL("../scripts/ga4_mcp_server.mjs", import.meta.url))], {
    stdio: ["pipe", "pipe", "pipe"],
  });
  const lines = createInterface({ input: child.stdout, crlfDelay: Infinity });
  const responses = [];
  lines.on("line", (line) => responses.push(JSON.parse(line)));
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`);
  child.stdin.end();
  await once(child, "exit");
  assert.equal(child.exitCode, 0);
  assert.equal(responses.length, 2);
  assert.equal(responses[0].result.serverInfo.name, "ga4-mcp-connector");
  assert.equal(responses[1].result.tools[0].name, "ga4_run_report");
});
