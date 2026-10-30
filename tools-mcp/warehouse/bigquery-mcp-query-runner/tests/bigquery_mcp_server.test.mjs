import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createHash, generateKeyPairSync } from "node:crypto";
import { once } from "node:events";
import { mkdtempSync, rmSync, symlinkSync } from "node:fs";
import { createInterface } from "node:readline";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  BigQueryError,
  getAccessToken,
  handleMCPRequest,
  parseCredentialEnvelope,
  runQuery,
  validateQueryInput,
} from "../scripts/bigquery_mcp_server.mjs";
import { runDriver } from "../scripts/bigquery_auth_driver.mjs";

const accessCredential = JSON.stringify({
  type: "access_token",
  access_token: "test-bigquery-access-token",
  expires_at: "2099-01-01T00:00:00Z",
  account: "billing-project",
});

function env(overrides = {}) {
  return {
    BQ_CREDENTIAL: accessCredential,
    BQ_POLICY: JSON.stringify({
      allowed_project_ids: ["data-project", "billing-project"],
      billing_project_id: "billing-project",
      allowed_datasets: ["data-project.analytics", "billing-project.scratch"],
      allowed_locations: ["EU", "europe-west2"],
      max_bytes_billed: 1000000,
      max_rows: 1000,
      max_timeout_ms: 120000,
      allow_user_oauth: true,
      environment_tier: "sandbox",
    }),
    ...overrides,
  };
}

function jsonResponse(payload, init = {}) {
  return new Response(JSON.stringify(payload), { status: init.status ?? 200, headers: { "content-type": "application/json", ...(init.headers ?? {}) } });
}

const baseInput = {
  query: "SELECT channel, SUM(spend) AS spend FROM `data-project.analytics.performance` WHERE event_date >= @start GROUP BY channel",
  parameters: [{ name: "start", type: "DATE", value: "2026-09-01" }],
  project_id: "data-project",
  billing_project_id: "billing-project",
  location: "EU",
  max_rows: 10,
  page_size: 2,
};

function dryRun(bytes = "100") { return { jobComplete: true, totalBytesProcessed: bytes }; }

const testJobID = "skills_hub_job-123";

function submittedJob(overrides = {}) {
  return {
    jobReference: { projectId: "billing-project", jobId: testJobID, location: "EU" },
    status: { state: "RUNNING" },
    ...overrides,
  };
}

function terminalJob(overrides = {}) {
  return {
    jobReference: { projectId: "billing-project", jobId: testJobID, location: "EU" },
    status: { state: "DONE" },
    statistics: { query: { totalBytesProcessed: "100", totalBytesBilled: "100", totalSlotMs: "4", cacheHit: false } },
    ...overrides,
  };
}

function resultPage(rows, overrides = {}) {
  return {
    jobComplete: true,
    jobReference: { projectId: "billing-project", jobId: testJobID, location: "EU" },
    schema: { fields: [{ name: "channel", type: "STRING" }, { name: "spend", type: "NUMERIC" }, { name: "ratio", type: "FLOAT64" }, { name: "active", type: "BOOL" }] },
    rows: rows.map(([channel, spend, ratio, active]) => ({ f: [{ v: channel }, { v: spend }, { v: ratio }, { v: active }] })),
    totalRows: String(rows.length),
    totalBytesProcessed: "100",
    cacheHit: false,
    ...overrides,
  };
}

function successfulFetch(rows = [["Direct", "1", "1", "true"]], { resultOverrides = {}, jobOverrides = {} } = {}) {
  return async (url, options) => {
    const endpoint = String(url);
    if (endpoint.endsWith("/queries") && options.method === "POST") return jsonResponse(dryRun());
    if (endpoint.endsWith("/jobs") && options.method === "POST") return jsonResponse(submittedJob());
    if (endpoint.includes(`/queries/${testJobID}`)) return jsonResponse(resultPage(rows, resultOverrides));
    if (endpoint.includes(`/jobs/${testJobID}`)) return jsonResponse(terminalJob(jobOverrides));
    throw new Error(`unexpected request ${endpoint}`);
  };
}

function runOptions(fetchImpl, overrides = {}) {
  return { env: env(), resolvedAuth: { token: "resolved-token" }, fetchImpl, jobIDFactory: () => "job-123", ...overrides };
}

test("validation rejects mutation, scripts, scope escape, and parameter mismatch before fetch", () => {
  for (const query of [
    "DELETE FROM `data-project.analytics.performance` WHERE TRUE",
    "SELECT 1; DROP TABLE `data-project.analytics.performance`",
    "SELECT * FROM EXTERNAL_QUERY('connection', 'SELECT 1')",
    "SELECT `data-project.analytics.remote_enricher`(channel) FROM `data-project.analytics.performance`",
  ]) {
    assert.throws(() => validateQueryInput({ ...baseInput, query, parameters: [] }, env()), (error) => error instanceof BigQueryError && error.code === "QUERY_NOT_ALLOWED");
  }
  assert.throws(
    () => validateQueryInput({ ...baseInput, query: "SELECT * FROM `other-project.secret.data`", parameters: [] }, env()),
    (error) => error.code === "DATASET_NOT_ALLOWED",
  );
  assert.throws(
    () => validateQueryInput({ ...baseInput, parameters: [] }, env()),
    (error) => error.code === "PARAMETER_MISMATCH",
  );
  for (const query of [
    "SELECT * FROM `data-project.analytics.performance` a JOIN secret_table b ON TRUE",
    "SELECT * FROM `data-project.analytics.performance`, secret_table",
    "SELECT * FROM @dataset",
  ]) {
    assert.throws(() => validateQueryInput({ ...baseInput, query, parameters: [] }, env()), (error) => error.code === "DATASET_NOT_ALLOWED");
  }
});

test("SQL lexer does not interpret comments, literals, or quoted names as executable mutation", () => {
  const validated = validateQueryInput({
    ...baseInput,
    query: "SELECT 'DROP TABLE x; -- inert' AS note, channel FROM `data-project.analytics.performance` WHERE channel = @channel /* DELETE */",
    parameters: [{ name: "channel", type: "STRING", value: "Direct" }],
  }, env());
  assert.deepEqual(validated.tableReferences, ["data-project.analytics.performance"]);
});

test("dry-run estimate blocks execution before chargeable request", async () => {
  let calls = 0;
  await assert.rejects(
    () => runQuery(baseInput, { env: env(), resolvedAuth: { token: "resolved-token" }, fetchImpl: async () => { calls += 1; return jsonResponse(dryRun("1000001")); } }),
    (error) => error.code === "BYTE_BUDGET_EXCEEDED",
  );
  assert.equal(calls, 1);
});

test("execution repeats maximumBytesBilled and returns typed rows and diagnostics", async () => {
  const requests = [];
  const responses = [dryRun("900"), submittedJob(), resultPage([["Direct", "12.50", "0.25", "true"]]), terminalJob()];
  const result = await runQuery(baseInput, {
    env: env(), resolvedAuth: { token: "resolved-token" }, now: () => new Date("2026-09-18T10:00:00Z"),
    jobIDFactory: () => "job-123",
    fetchImpl: async (url, options) => { requests.push({ url: String(url), body: options.body ? JSON.parse(options.body) : undefined }); return jsonResponse(responses.shift()); },
  });
  assert.equal(requests[0].body.dryRun, true);
  assert.equal(requests[0].body.jobTimeoutMs, "30000");
  assert.equal(requests[1].url.endsWith("/jobs"), true);
  assert.equal(requests[1].body.configuration.jobTimeoutMs, "30000");
  assert.equal(requests[1].body.configuration.query.maximumBytesBilled, "1000000");
  assert.equal(requests[1].body.jobReference.jobId, testJobID);
  assert.equal(result.rows[0].spend, "12.50");
  assert.equal(result.rows[0].ratio, 0.25);
  assert.equal(result.rows[0].active, true);
  assert.equal(result.diagnostics.dry_run_bytes_processed, "900");
  assert.deepEqual(result.diagnostics.table_references, ["data-project.analytics.performance"]);
});

test("query results paginate with stable schema and row limits", async () => {
  const responses = [
    dryRun(),
    submittedJob(),
    resultPage([["Direct", "1", "1", "true"]], { totalRows: "2", pageToken: "page-2" }),
    terminalJob(),
    resultPage([["Organic", "2", "2", "false"]], { totalRows: "2", pageToken: undefined }),
  ];
  const result = await runQuery(baseInput, runOptions(async () => jsonResponse(responses.shift())));
  assert.equal(result.rows.length, 2);
  assert.equal(result.rows[1].channel, "Organic");
  assert.equal(result.pagination.pages, 2);
});

test("one absolute deadline governs terminal receipt and every result page", async (t) => {
  await t.test("terminal receipt crossing the deadline is rejected", async () => {
    let clock = 0;
    const urls = [];
    await assert.rejects(
      () => runQuery({ ...baseInput, timeout_ms: 10 }, runOptions(async (url, options) => {
        const endpoint = String(url);
        urls.push(endpoint);
        if (endpoint.endsWith("/queries") && options.method === "POST") return jsonResponse(dryRun());
        if (endpoint.endsWith("/jobs") && options.method === "POST") return jsonResponse(submittedJob());
        if (endpoint.includes(`/queries/${testJobID}`)) { clock = 4; return jsonResponse(resultPage([["Direct", "1", "1", "true"]])); }
        if (endpoint.includes(`/jobs/${testJobID}/cancel`)) return jsonResponse({ job: { id: testJobID } });
        if (endpoint.includes(`/jobs/${testJobID}`)) { clock = 10; return jsonResponse(terminalJob()); }
        throw new Error(`unexpected request ${endpoint}`);
      }, { now: () => new Date(clock) })),
      (error) => error.code === "QUERY_TIMEOUT" && error.details.cancellation_requested === true,
    );
    assert.ok(urls.some((url) => url.includes(`/jobs/${testJobID}/cancel`)));
  });

  await t.test("later page crossing the deadline is rejected", async () => {
    let clock = 0;
    const urls = [];
    await assert.rejects(
      () => runQuery({ ...baseInput, timeout_ms: 10 }, runOptions(async (url, options) => {
        const endpoint = String(url);
        urls.push(endpoint);
        if (endpoint.endsWith("/queries") && options.method === "POST") return jsonResponse(dryRun());
        if (endpoint.endsWith("/jobs") && options.method === "POST") return jsonResponse(submittedJob());
        if (endpoint.includes(`/jobs/${testJobID}/cancel`)) return jsonResponse({ job: { id: testJobID } });
        if (endpoint.includes(`/jobs/${testJobID}`)) { clock = 6; return jsonResponse(terminalJob()); }
        if (endpoint.includes(`/queries/${testJobID}`) && new URL(endpoint).searchParams.has("pageToken")) {
          clock = 10;
          return jsonResponse(resultPage([["Organic", "2", "2", "false"]], { totalRows: "2" }));
        }
        if (endpoint.includes(`/queries/${testJobID}`)) {
          clock = 3;
          return jsonResponse(resultPage([["Direct", "1", "1", "true"]], { totalRows: "2", pageToken: "page-2" }));
        }
        throw new Error(`unexpected request ${endpoint}`);
      }, { now: () => new Date(clock) })),
      (error) => error.code === "QUERY_TIMEOUT" && error.details.cancellation_requested === true,
    );
    assert.ok(urls.some((url) => new URL(url).searchParams.has("pageToken")));
    assert.ok(urls.some((url) => url.includes(`/jobs/${testJobID}/cancel`)));
  });
});

test("normalized rows are bounded for the duplicated MCP wire representation", async () => {
  const huge = "x".repeat(1_100_000);
  const called = await handleMCPRequest({ jsonrpc: "2.0", id: 4, method: "tools/call", params: { name: "bigquery_run_query", arguments: { ...baseInput, max_rows: 1, page_size: 1 } } }, {
    ...runOptions(successfulFetch([[huge, "1", "1", "true"]])),
  });
  assert.equal(called.isError, true);
  assert.equal(called.structuredContent.error.code, "RESULT_TOO_LARGE");
});

test("incomplete jobs are polled and timeout requests cancellation", async () => {
  const urls = [];
  let clock = 0;
  await assert.rejects(
    () => runQuery({ ...baseInput, timeout_ms: 5 }, {
      ...runOptions(async (url, options) => {
        const endpoint = String(url);
        urls.push(endpoint);
        if (endpoint.endsWith("/queries") && options.method === "POST") return jsonResponse(dryRun());
        if (endpoint.endsWith("/jobs") && options.method === "POST") return jsonResponse(submittedJob());
        if (endpoint.includes(`/queries/${testJobID}`)) { clock = 5; return jsonResponse({ jobComplete: false, jobReference: { projectId: "billing-project", jobId: testJobID, location: "EU" } }); }
        if (endpoint.includes("/cancel")) return jsonResponse({ job: { id: testJobID } });
        throw new Error(`unexpected request ${endpoint}`);
      }),
      now: () => new Date(clock), sleep: async () => {},
    }),
    (error) => error.code === "QUERY_TIMEOUT" && error.details.cancellation_requested === true && error.details.job_id === testJobID,
  );
  assert.ok(urls.some((url) => url.includes(`/queries/${testJobID}`)));
  assert.ok(urls.some((url) => url.includes(`/jobs/${testJobID}/cancel`)));
});

test("ambiguous job submission retains identity, provider timeout, and cancellation", async () => {
  const requests = [];
  await assert.rejects(
    () => runQuery(baseInput, runOptions(async (url, options) => {
      requests.push({ url: String(url), body: options.body ? JSON.parse(options.body) : undefined });
      if (String(url).endsWith("/queries")) return jsonResponse(dryRun());
      if (String(url).endsWith("/jobs")) throw Object.assign(new Error("lost response"), { name: "AbortError" });
      if (String(url).includes("/cancel")) return jsonResponse({ job: { id: testJobID } });
      throw new Error("unexpected request");
    })),
    (error) => error.code === "QUERY_SUBMISSION_AMBIGUOUS" && error.details.job_id === testJobID && error.details.cancellation_requested === true,
  );
  assert.equal(requests[1].body.configuration.jobTimeoutMs, "30000");
  assert.ok(requests[2].url.includes(`/jobs/${testJobID}/cancel`));
});

test("nonterminal warnings preserve job ownership until terminal status is confirmed", async () => {
  const responses = [
    dryRun(),
    submittedJob(),
    { jobComplete: false, jobReference: { projectId: "billing-project", jobId: testJobID, location: "EU" }, errors: [{ reason: "backendWarning", message: "do not expose" }] },
    resultPage([["Direct", "1", "1", "true"]], { errors: [{ reason: "partialWarning", message: "do not expose" }] }),
    terminalJob({ status: { state: "DONE", errors: [{ reason: "jobWarning", message: "do not expose" }] } }),
  ];
  const result = await runQuery(baseInput, runOptions(async () => jsonResponse(responses.shift()), { sleep: async () => {} }));
  assert.equal(result.rows[0].channel, "Direct");
  assert.deepEqual(result.diagnostics.warning_reasons, ["backendWarning", "partialWarning", "jobWarning"]);
  assert.ok(!JSON.stringify(result).includes("do not expose"));
});

test("terminal job errorResult fails even when earlier errors were only warnings", async () => {
  const responses = [
    dryRun(),
    submittedJob(),
    resultPage([["Direct", "1", "1", "true"]], { errors: [{ reason: "partialWarning", message: "do not expose" }] }),
    terminalJob({ status: { state: "DONE", errors: [{ reason: "backendError", message: "do not expose" }], errorResult: { reason: "backendError", message: "do not expose" } } }),
  ];
  await assert.rejects(
    () => runQuery(baseInput, runOptions(async () => jsonResponse(responses.shift()))),
    (error) => error.code === "QUERY_FAILED" && error.details.job_id === testJobID && !JSON.stringify(error).includes("do not expose"),
  );
});

test("quota and malformed provider responses are normalized without credential leakage", async (t) => {
  await t.test("quota", async () => {
    const responses = [dryRun(), { error: { errors: [{ reason: "quotaExceeded" }] } }];
    await assert.rejects(
      () => runQuery(baseInput, runOptions(async () => { const payload = responses.shift(); return jsonResponse(payload, payload.error ? { status: 429 } : {}); })),
      (error) => error.code === "QUOTA_EXHAUSTED" && error.retryable,
    );
  });
  await t.test("malformed", async () => {
    await assert.rejects(
      () => runQuery(baseInput, { env: env(), resolvedAuth: { token: "resolved-token" }, fetchImpl: async () => new Response("not-json") }),
      (error) => error.code === "UPSTREAM_MALFORMED",
    );
  });
});

test("MCP initialize, list, call, and aggregate wire budget are enforced", async () => {
  const initialized = await handleMCPRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2025-06-18" } });
  assert.equal(initialized.serverInfo.name, "bigquery-mcp-query-runner");
  const listed = await handleMCPRequest({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} });
  assert.deepEqual(listed.tools.map((tool) => tool.name), ["bigquery_run_query"]);
  const called = await handleMCPRequest({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "bigquery_run_query", arguments: baseInput } }, {
    ...runOptions(successfulFetch()),
  });
  assert.equal(called.isError, false);
  assert.equal(called.structuredContent.rows[0].channel, "Direct");
});

test("credential parsing and user OAuth are fail-closed", async () => {
  assert.throws(() => parseCredentialEnvelope(JSON.stringify({ ...JSON.parse(accessCredential), principal: "spoofed@example.test" })), (error) => error.code === "INVALID_ARGUMENT");
  const deniedPolicy = JSON.stringify({ ...JSON.parse(env().BQ_POLICY), allow_user_oauth: false });
  await assert.rejects(() => getAccessToken(env({ BQ_POLICY: deniedPolicy })), (error) => error.code === "AUTH_METHOD_NOT_ALLOWED");
});

test("service-account and workload-identity token exchanges stay on canonical endpoints", async (t) => {
  const { privateKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
  await t.test("service account", async () => {
    const calls = [];
    const credential = JSON.stringify({ type: "service_account", client_email: "reader@example-project.iam.gserviceaccount.com", private_key: privateKey.export({ type: "pkcs8", format: "pem" }), private_key_id: "key-id", project_id: "example-project", account: "billing-project" });
    const token = await getAccessToken(env({ BQ_CREDENTIAL: credential }), async (url, options) => { calls.push({ url: String(url), options }); return jsonResponse({ access_token: "service-account-token", expires_in: 3600 }); });
    assert.equal(calls[0].url, "https://oauth2.googleapis.com/token");
    assert.equal(calls[0].options.body.get("grant_type"), "urn:ietf:params:oauth:grant-type:jwt-bearer");
    assert.equal(token.token, "service-account-token");
  });
  await t.test("workload identity with impersonation", async () => {
    const calls = [];
    const credential = JSON.stringify({
      type: "external_account", audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
      subject_token_type: "urn:ietf:params:oauth:token-type:jwt", token_url: "https://sts.googleapis.com/v1/token",
      service_account_impersonation_url: "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/reader%40example-project.iam.gserviceaccount.com:generateAccessToken",
      subject_token: "signed-subject-token", account: "billing-project",
    });
    const token = await getAccessToken(env({ BQ_CREDENTIAL: credential }), async (url, options) => {
      calls.push({ url: String(url), options });
      return calls.length === 1 ? jsonResponse({ access_token: "federated-token", expires_in: 600 }) : jsonResponse({ accessToken: "impersonated-token", expireTime: "2099-01-01T00:00:00Z" });
    });
    assert.deepEqual(calls.map((call) => new URL(call.url).hostname), ["sts.googleapis.com", "iamcredentials.googleapis.com"]);
    assert.equal(calls[1].options.headers.authorization, "Bearer federated-token");
    assert.equal(token.token, "impersonated-token");
  });
});

test("auth driver provider-verifies identity and executes a live read-only probe", async () => {
  const calls = [];
  const options = {
    env: env(), now: () => new Date("2026-09-18T10:00:00Z"),
    fetchImpl: async (url, request) => {
      calls.push(String(url));
      if (String(url).startsWith("https://oauth2.googleapis.com/tokeninfo?")) return jsonResponse({ email: "analyst@example.test", scope: "https://www.googleapis.com/auth/bigquery", expires_in: "3600" });
      if (String(url).endsWith("/queries") && request.method === "POST") return jsonResponse(dryRun("0"));
      if (String(url).endsWith("/jobs") && request.method === "POST") {
        const requested = JSON.parse(request.body).jobReference;
        return jsonResponse({ jobReference: requested, status: { state: "RUNNING" } });
      }
      if (String(url).includes("/queries/skills_hub_")) return jsonResponse(resultPage([["readiness", "0", "0", "true"]], { jobReference: { projectId: "billing-project", jobId: new URL(String(url)).pathname.split("/").at(-1), location: "EU" }, schema: { fields: [{ name: "probe", type: "STRING" }] }, rows: [{ f: [{ v: "readiness" }] }], totalRows: "1" }));
      if (String(url).includes("/jobs/skills_hub_")) return jsonResponse({ jobReference: { projectId: "billing-project", jobId: new URL(String(url)).pathname.split("/").at(-1), location: "EU" }, status: { state: "DONE" }, statistics: { query: { totalBytesProcessed: "0", totalBytesBilled: "0", totalSlotMs: "0" } } });
      throw new Error(`unexpected request ${url}`);
    },
  };
  const status = await runDriver("bearer-token", "status", {}, options);
  assert.equal(status.principal, `google-user-sha256:${createHash("sha256").update("analyst@example.test").digest("hex")}`);
  assert.equal(status.account, "billing-project");
  assert.equal(status.tier, "sandbox");
  assert.equal(status.endpoint, "https://bigquery.googleapis.com");
  assert.equal(status.region, "EU");
  assert.match(status.attestation_reference, /^bigquery-job:skills_hub_/);
  assert.ok(calls.some((url) => url.includes("/bigquery/v2/projects/billing-project/queries")));
  assert.ok(!JSON.stringify(status).includes("test-bigquery-access-token"));
});

test("auth readiness fails closed unless the provider attests a zero-byte probe", async () => {
  await assert.rejects(
    () => runDriver("bearer-token", "status", {}, {
      env: env(), now: () => new Date("2026-09-18T10:00:00Z"),
      fetchImpl: async (url, request) => {
        if (String(url).startsWith("https://oauth2.googleapis.com/tokeninfo?")) return jsonResponse({ email: "analyst@example.test", scope: "https://www.googleapis.com/auth/bigquery", expires_in: "3600" });
        if (String(url).endsWith("/queries") && request.method === "POST") return jsonResponse(dryRun("0"));
        if (String(url).endsWith("/jobs") && request.method === "POST") {
          const requested = JSON.parse(request.body).jobReference;
          return jsonResponse({ jobReference: requested, status: { state: "RUNNING" } });
        }
        if (String(url).includes("/queries/skills_hub_")) return jsonResponse(resultPage([["readiness", "0", "0", "true"]], { jobReference: { projectId: "billing-project", jobId: new URL(String(url)).pathname.split("/").at(-1), location: "EU" }, schema: { fields: [{ name: "probe", type: "STRING" }] }, rows: [{ f: [{ v: "readiness" }] }], totalRows: "1", totalBytesProcessed: "1" }));
        if (String(url).includes("/jobs/skills_hub_")) return jsonResponse({ jobReference: { projectId: "billing-project", jobId: new URL(String(url)).pathname.split("/").at(-1), location: "EU" }, status: { state: "DONE" }, statistics: { query: { totalBytesProcessed: "1", totalBytesBilled: "1", totalSlotMs: "0" } } });
        throw new Error(`unexpected request ${url}`);
      },
    }),
    (error) => error.code === "AUTH_PROBE_COST_NONZERO",
  );
});

test("metadata application-default auth recognizes cloud-platform as effective BigQuery scope", async () => {
  const metadataCredential = JSON.stringify({ type: "application_default", source: "metadata", account: "billing-project" });
  const calls = [];
  const status = await runDriver("workload-identity", "status", {}, {
    env: env({ BQ_CREDENTIAL: metadataCredential }),
    now: () => new Date("2026-09-18T10:00:00Z"),
    fetchImpl: async (url, request) => {
      calls.push(String(url));
      if (String(url).startsWith("http://metadata.google.internal/")) return jsonResponse({ access_token: "metadata-access-token", expires_in: 3600 });
      if (String(url).startsWith("https://oauth2.googleapis.com/tokeninfo?")) return jsonResponse({ aud: "metadata-service-account-client", azp: "metadata-service-account-client", scope: "https://www.googleapis.com/auth/cloud-platform", expires_in: "3600" });
      if (String(url).endsWith("/queries") && request.method === "POST") return jsonResponse(dryRun("0"));
      if (String(url).endsWith("/jobs") && request.method === "POST") {
        const requested = JSON.parse(request.body).jobReference;
        return jsonResponse({ jobReference: requested, status: { state: "RUNNING" } });
      }
      if (String(url).includes("/queries/skills_hub_")) return jsonResponse(resultPage([["readiness", "0", "0", "true"]], { jobReference: { projectId: "billing-project", jobId: new URL(String(url)).pathname.split("/").at(-1), location: "EU" }, schema: { fields: [{ name: "probe", type: "STRING" }] }, rows: [{ f: [{ v: "readiness" }] }], totalRows: "1" }));
      if (String(url).includes("/jobs/skills_hub_")) return jsonResponse({ jobReference: { projectId: "billing-project", jobId: new URL(String(url)).pathname.split("/").at(-1), location: "EU" }, status: { state: "DONE" }, statistics: { query: { totalBytesProcessed: "0", totalBytesBilled: "0", totalSlotMs: "0" } } });
      throw new Error(`unexpected request ${url}`);
    },
  });
  assert.equal(status.principal, `google-workload-sha256:${createHash("sha256").update("metadata-service-account-client").digest("hex")}`);
  assert.equal(status.tier, "sandbox");
  assert.ok(status.scopes.includes("https://www.googleapis.com/auth/bigquery"));
  assert.ok(calls[0].includes("metadata.google.internal"));
});

test("launchable stdio process bounds request lines and continues with the next request", async () => {
  const child = spawn(process.execPath, [fileURLToPath(new URL("../scripts/bigquery_mcp_server.mjs", import.meta.url))], { stdio: ["pipe", "pipe", "pipe"] });
  const lines = createInterface({ input: child.stdout, crlfDelay: Infinity });
  const responses = [];
  lines.on("line", (line) => responses.push(JSON.parse(line)));
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 99, method: "unknown", padding: "x".repeat(300_000) })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`);
  child.stdin.end();
  await once(child, "exit");
  assert.equal(child.exitCode, 0);
  assert.equal(responses[0].result.serverInfo.name, "bigquery-mcp-query-runner");
  assert.equal(responses[1].error.data.code, "REQUEST_TOO_LARGE");
  assert.equal(responses[2].result.tools[0].name, "bigquery_run_query");
});

test("packaged entrypoints execute through their direct shebang path", async () => {
  const serverPath = fileURLToPath(new URL("../scripts/bigquery_mcp_server.mjs", import.meta.url));
  const health = spawn(serverPath, ["--healthcheck"], { stdio: ["ignore", "pipe", "pipe"] });
  let healthOutput = "";
  health.stdout.on("data", (chunk) => { healthOutput += chunk; });
  await once(health, "exit");
  assert.equal(health.exitCode, 0);
  assert.match(healthOutput, /"status":"ok"/);

  const driverPath = fileURLToPath(new URL("../scripts/bigquery_auth_driver.mjs", import.meta.url));
  const driver = spawn(driverPath, ["bearer-token", "status"], { env: { PATH: process.env.PATH }, stdio: ["pipe", "pipe", "pipe"] });
  let driverError = "";
  driver.stderr.on("data", (chunk) => { driverError += chunk; });
  driver.stdin.end("{}");
  await once(driver, "exit");
  assert.equal(driver.exitCode, 1);
  assert.equal(JSON.parse(driverError).code, "AUTH_MISSING");

  const linkedDirectory = mkdtempSync("/tmp/bigquery-entrypoint-");
  try {
    const linkedDriver = `${linkedDirectory}/bigquery_auth_driver.mjs`;
    symlinkSync(driverPath, linkedDriver);
    const linked = spawn(linkedDriver, ["bearer-token", "status"], { env: { PATH: process.env.PATH }, stdio: ["pipe", "pipe", "pipe"] });
    let linkedError = "";
    linked.stderr.on("data", (chunk) => { linkedError += chunk; });
    linked.stdin.end("{}");
    await once(linked, "exit");
    assert.equal(linked.exitCode, 1);
    assert.equal(JSON.parse(linkedError).code, "AUTH_MISSING");
  } finally {
    rmSync(linkedDirectory, { recursive: true, force: true });
  }
});
