import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { createInterface } from "node:readline";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  MetaAdsError,
  handleMCPRequest,
  parseCredentialEnvelope,
  parsePolicy,
  runInsights,
  validateInsightsInput,
  verifyMetaAuthority,
} from "../scripts/meta_ads_mcp_server.mjs";
import { runDriver } from "../scripts/meta_ads_auth_driver.mjs";

const ACCESS_TOKEN = "test-meta-access-token-value-123456789";
const APP_SECRET = "test-meta-app-secret-value-123456789";
const ACCOUNT_ID = "act_123456789";
const BUSINESS_ID = "987654321";
const APP_ID = "123456789";
const credential = JSON.stringify({ type: "access_token", access_token: ACCESS_TOKEN, app_id: APP_ID, app_secret: APP_SECRET });
const policyDocument = {
  allowed_ad_account_ids: [ACCOUNT_ID],
  allowed_business_ids: [BUSINESS_ID],
  max_date_days: 90,
  max_rows: 500,
  max_pages: 10,
  max_timeout_ms: 30_000,
  graph_api_version: "v24.0",
  environment_tier: "sandbox",
};

function env(overrides = {}) {
  return { META_ADS_CREDENTIAL: credential, META_ADS_POLICY: JSON.stringify(policyDocument), ...overrides };
}

function jsonResponse(payload, init = {}) {
  return new Response(JSON.stringify(payload), {
    status: init.status ?? 200,
    headers: { "content-type": "application/json", ...(init.headers ?? {}) },
  });
}

function delayedJSONResponse(payload, { stall = false, delayMs = 1_000 } = {}) {
  const bytes = new TextEncoder().encode(JSON.stringify(payload));
  let timer;
  return new Response(new ReadableStream({
    start(controller) {
      if (stall) return;
      const split = Math.max(1, Math.floor(bytes.length / 2));
      controller.enqueue(bytes.slice(0, split));
      timer = setTimeout(() => {
        try {
          controller.enqueue(bytes.slice(split));
          controller.close();
        } catch {}
      }, delayMs);
    },
    cancel() { if (timer) clearTimeout(timer); },
  }), { headers: { "content-type": "application/json" } });
}

function debugToken(overrides = {}) {
  return {
    data: {
      app_id: APP_ID,
      type: "SYSTEM_USER",
      application: "Sandbox reporting app",
      data_access_expires_at: 4_102_444_800,
      expires_at: 4_102_444_800,
      is_valid: true,
      scopes: ["ads_read", "business_management"],
      user_id: "1111122222",
      ...overrides,
    },
  };
}

function account(overrides = {}) {
  return {
    id: ACCOUNT_ID,
    account_id: ACCOUNT_ID.slice(4),
    name: "Sandbox account",
    account_status: 1,
    business: { id: BUSINESS_ID, name: "Sandbox business" },
    currency: "GBP",
    timezone_name: "Europe/London",
    timezone_offset_hours_utc: 1,
    attribution_spec: [{ event_type: "CLICK_THROUGH", window_days: 7 }, { event_type: "VIEW_THROUGH", window_days: 1 }],
    ...overrides,
  };
}

function verified(overrides = {}) {
  return {
    token: ACCESS_TOKEN,
    appSecretProof: "b".repeat(64),
    policy: parsePolicy(JSON.stringify(policyDocument)),
    account: account(),
    accountID: ACCOUNT_ID,
    businessID: BUSINESS_ID,
    principal: "meta-user:1111122222",
    scopes: ["ads_read", "business_management"],
    providerExpiresAt: "2099-01-01T00:00:00.000Z",
    generation: "test-generation",
    attestationReference: "meta-live:" + "a".repeat(64),
    ...overrides,
  };
}

const baseInput = {
  ad_account_id: ACCOUNT_ID,
  level: "campaign",
  start_date: "2026-09-01",
  end_date: "2026-09-07",
  fields: ["campaign_id", "campaign_name", "impressions", "clicks", "spend", "actions", "action_values"],
};

test("policy and request validation reject scope expansion before fetch", () => {
  const policy = parsePolicy(JSON.stringify(policyDocument));
  assert.throws(() => validateInsightsInput({ ...baseInput, ad_account_id: "act_999999999" }, policy), (error) => error.code === "ACCOUNT_NOT_ALLOWED");
  assert.throws(() => validateInsightsInput({ ...baseInput, start_date: "2025-01-01" }, policy), (error) => error.code === "DATE_RANGE_NOT_ALLOWED");
  assert.throws(() => validateInsightsInput({ ...baseInput, fields: ["effective_status"] }, policy), (error) => error.code === "INVALID_ARGUMENT");
  assert.throws(() => validateInsightsInput({ ...baseInput, mutation: "pause" }, policy), (error) => error.code === "INVALID_ARGUMENT");
  assert.throws(() => parsePolicy(JSON.stringify({ ...policyDocument, allowed_ad_account_ids: [] })), (error) => error.code === "CONFIGURATION_ERROR");
  assert.throws(() => parsePolicy(JSON.stringify({ ...policyDocument, allowed_ad_account_ids: [ACCOUNT_ID, "act_999999999"] })), (error) => error.code === "CONFIGURATION_ERROR");
});

test("the executable path rejects a wrong account before any provider request", async () => {
  let contacted = false;
  await assert.rejects(
    () => runInsights({ ...baseInput, ad_account_id: "act_999999999" }, { env: env(), fetchImpl: async () => { contacted = true; return jsonResponse({}); } }),
    (error) => error.code === "ACCOUNT_NOT_ALLOWED",
  );
  assert.equal(contacted, false);
});

test("credential envelopes cannot self-attest identity, scopes, or account", () => {
  for (const injected of [
    { principal: "forged-user" },
    { scopes: ["ads_read", "business_management"] },
    { ad_account_id: ACCOUNT_ID },
    { graph_origin: "https://attacker.example" },
  ]) {
    assert.throws(
      () => parseCredentialEnvelope(JSON.stringify({ type: "access_token", access_token: ACCESS_TOKEN, app_id: APP_ID, app_secret: APP_SECRET, ...injected })),
      (error) => error instanceof MetaAdsError && error.code === "INVALID_ARGUMENT",
    );
  }
});

test("provider verification binds app, principal, scopes, business, account, and expiry", async () => {
  const calls = [];
  const auth = await verifyMetaAuthority({
    env: env(),
    now: () => new Date("2026-09-19T12:00:00Z"),
    fetchImpl: async (url, options) => {
      calls.push({ url: new URL(url), options });
      return calls.length === 1 ? jsonResponse(debugToken()) : jsonResponse(account());
    },
  });
  assert.equal(calls.length, 2);
  assert.equal(calls[0].url.origin, "https://graph.facebook.com");
  assert.equal(calls[0].url.pathname, "/v24.0/debug_token");
  assert.equal(calls[1].url.pathname, `/v24.0/${ACCOUNT_ID}`);
  assert.match(calls[1].url.searchParams.get("appsecret_proof"), /^[a-f0-9]{64}$/);
  assert.equal(calls[1].options.headers.authorization, `Bearer ${ACCESS_TOKEN}`);
  assert.equal(auth.principal, "meta-user:1111122222");
  assert.equal(auth.accountID, ACCOUNT_ID);
  assert.equal(auth.businessID, BUSINESS_ID);
  assert.deepEqual(auth.scopes, ["ads_read", "business_management"]);
  assert.match(auth.attestationReference, /^meta-live:[a-f0-9]{64}$/);
  assert.ok(!JSON.stringify(auth.account).includes(ACCESS_TOKEN));
});

test("provider-declared non-expiring system-user tokens receive bounded readiness evidence", async () => {
  const auth = await verifyMetaAuthority({
    env: env(),
    now: () => new Date("2026-09-19T12:00:00Z"),
    fetchImpl: async (url) => String(url).includes("/debug_token?")
      ? jsonResponse(debugToken({ expires_at: 0, data_access_expires_at: 0 }))
      : jsonResponse(account()),
  });
  assert.equal(auth.providerExpiresAt, null);
});

test("provider lifetime metadata is explicit and remains absolute through account verification", async (t) => {
  await t.test("missing expiry metadata is not treated as non-expiring", async () => {
    const missing = debugToken();
    delete missing.data.expires_at;
    delete missing.data.data_access_expires_at;
    let calls = 0;
    await assert.rejects(
      () => verifyMetaAuthority({ env: env(), now: () => new Date("2026-09-19T12:00:00Z"), fetchImpl: async () => { calls += 1; return jsonResponse(missing); } }),
      (error) => error.code === "AUTH_IDENTITY_UNVERIFIED",
    );
    assert.equal(calls, 1);
  });

  await t.test("expiry crossing during account verification fails closed", async () => {
    const times = [new Date("2026-09-19T12:00:00Z"), new Date("2026-09-19T12:00:10Z")];
    let calls = 0;
    const expiresAt = Math.floor(new Date("2026-09-19T12:00:05Z").valueOf() / 1_000);
    await assert.rejects(
      () => verifyMetaAuthority({
        env: env(), now: () => times.shift(),
        fetchImpl: async () => (++calls === 1 ? jsonResponse(debugToken({ expires_at: expiresAt, data_access_expires_at: expiresAt })) : jsonResponse(account())),
      }),
      (error) => error.code === "AUTH_EXPIRED",
    );
    assert.equal(calls, 2);
  });
});

test("provider verification fails closed before insights for invalid authority", async (t) => {
  const cases = [
    ["revoked", debugToken({ is_valid: false }), account(), "AUTH_REVOKED"],
    ["wrong app", debugToken({ app_id: "777777777" }), account(), "AUTH_APP_MISMATCH"],
    ["missing scope", debugToken({ scopes: ["ads_read"] }), account(), "AUTH_SCOPE_INVALID"],
    ["expired", debugToken({ expires_at: 1, data_access_expires_at: 1 }), account(), "AUTH_EXPIRED"],
    ["wrong business", debugToken(), account({ business: { id: "555555555" } }), "BUSINESS_MISMATCH"],
    ["wrong account", debugToken(), account({ id: "act_555555555", account_id: "555555555" }), "ACCOUNT_MISMATCH"],
    ["inactive account", debugToken(), account({ account_status: 2 }), "ACCOUNT_INACTIVE"],
  ];
  for (const [name, debug, profile, code] of cases) {
    await t.test(name, async () => {
      let calls = 0;
      await assert.rejects(
        () => verifyMetaAuthority({ env: env(), now: () => new Date("2026-09-19T12:00:00Z"), fetchImpl: async () => (++calls === 1 ? jsonResponse(debug) : jsonResponse(profile)) }),
        (error) => error.code === code && !error.message.includes(ACCESS_TOKEN) && !error.message.includes(APP_SECRET),
      );
      assert.ok(calls <= 2);
    });
  }
});

test("insights pagination is canonical, bounded, and preserves provider time and attribution", async () => {
  const calls = [];
  const pages = [
    {
      data: [{ account_id: ACCOUNT_ID.slice(4), campaign_id: "1", campaign_name: "A", impressions: "10", clicks: "2", spend: "3.50", actions: [{ action_type: "purchase", value: "1" }], action_values: [{ action_type: "purchase", value: "9.99" }], date_start: "2026-09-01", date_stop: "2026-09-07", attribution_setting: "7d_click,1d_view" }],
      paging: { cursors: { after: "cursor-2" }, next: `https://graph.facebook.com/v24.0/${ACCOUNT_ID}/insights?access_token=${ACCESS_TOKEN}&after=cursor-2` },
    },
    { data: [{ account_id: ACCOUNT_ID.slice(4), campaign_id: "2", campaign_name: "B", impressions: "20", clicks: "3", spend: "4.75", date_start: "2026-09-01", date_stop: "2026-09-07", attribution_setting: "7d_click,1d_view" }] },
  ];
  const result = await runInsights({ ...baseInput, page_size: 1, max_rows: 10 }, {
    env: env(), resolvedAuth: verified(), now: () => new Date("2026-09-19T12:00:00Z"),
    fetchImpl: async (url, options) => {
      calls.push({ url: new URL(url), options });
      return jsonResponse(pages.shift(), { headers: { date: "Sat, 19 Sep 2026 12:00:00 GMT" } });
    },
  });
  assert.equal(calls.length, 2);
  assert.ok(calls.every((call) => call.url.origin === "https://graph.facebook.com" && !call.url.searchParams.has("access_token")));
  assert.ok(calls.every((call) => call.url.searchParams.get("appsecret_proof") === "b".repeat(64)));
  assert.equal(calls[1].url.searchParams.get("after"), "cursor-2");
  assert.equal(calls[0].url.searchParams.get("use_account_attribution_setting"), "true");
  assert.equal(result.rows[0].actions[0].value, "1");
  assert.equal(result.rows[1].spend, "4.75");
  assert.deepEqual(result.attribution.account_spec, account().attribution_spec);
  assert.deepEqual(result.source.provider_response_dates, ["2026-09-19T12:00:00.000Z"]);
  assert.equal(result.source.account_timezone, "Europe/London");
});

test("provider-supplied next URLs are never followed", async () => {
  const urls = [];
  const result = await runInsights({ ...baseInput, page_size: 1, max_rows: 2 }, {
    env: env(), resolvedAuth: verified(),
    fetchImpl: async (url) => {
      urls.push(String(url));
      if (urls.length === 1) return jsonResponse({ data: [{ account_id: ACCOUNT_ID.slice(4), campaign_id: "1" }], paging: { cursors: { after: "safe-cursor" }, next: "https://attacker.example/steal" } });
      return jsonResponse({ data: [] });
    },
  });
  assert.equal(result.rows.length, 1);
  assert.ok(urls.every((url) => url.startsWith("https://graph.facebook.com/")));
});

test("one absolute deadline governs every insights page", async () => {
  let current = 1_000;
  let calls = 0;
  await assert.rejects(
    () => runInsights({ ...baseInput, page_size: 1, max_rows: 2, timeout_ms: 10 }, {
      env: env(), resolvedAuth: verified(), clock: () => current,
      fetchImpl: async () => {
        calls += 1;
        current = 1_011;
        return jsonResponse({ data: [{ account_id: ACCOUNT_ID.slice(4), campaign_id: "1" }], paging: { cursors: { after: "next" } } });
      },
    }),
    (error) => error.code === "UPSTREAM_TIMEOUT" && error.retryable,
  );
  assert.equal(calls, 1);
});

test("the caller deadline also governs token and account verification", async () => {
  let current = 1_000;
  let calls = 0;
  await assert.rejects(
    () => runInsights({ ...baseInput, timeout_ms: 10 }, {
      env: env(), clock: () => current,
      fetchImpl: async () => {
        calls += 1;
        current += calls === 1 ? 5 : 6;
        return calls === 1 ? jsonResponse(debugToken()) : jsonResponse(account());
      },
    }),
    (error) => error.code === "UPSTREAM_TIMEOUT" && error.retryable,
  );
  assert.equal(calls, 2);
});

test("one deadline governs stalled and trickling response bodies at every provider stage", async (t) => {
  for (const mode of ["stalled", "trickling"]) {
    const responseFor = (payload) => delayedJSONResponse(payload, { stall: mode === "stalled" });
    for (const stage of ["debugger", "account", "insights"]) {
      await t.test(`${mode} ${stage} body`, async () => {
        let calls = 0;
        const fetchImpl = async () => {
          calls += 1;
          if (stage === "debugger") return responseFor(debugToken());
          if (stage === "account") return calls === 1 ? jsonResponse(debugToken()) : responseFor(account());
          return responseFor({ data: [] });
        };
        const operation = stage === "insights"
          ? runInsights({ ...baseInput, timeout_ms: 25 }, { env: env(), resolvedAuth: verified(), fetchImpl })
          : verifyMetaAuthority({ env: env(), fetchImpl, deadline: Date.now() + 25 });
        await assert.rejects(operation, (error) => error.code === "UPSTREAM_TIMEOUT" && error.retryable);
        assert.equal(calls, stage === "account" ? 2 : 1);
      });
    }
  }
});

test("rate limits, expiry, permissions, and malformed responses are normalized without secrets", async (t) => {
  const cases = [
    ["rate", jsonResponse({ error: { code: 613, message: ACCESS_TOKEN } }, { status: 429, headers: { "retry-after": "60" } }), "RATE_LIMITED", true, 60],
    ["expired", jsonResponse({ error: { code: 190, error_subcode: 463, message: ACCESS_TOKEN } }, { status: 401 }), "AUTH_EXPIRED", false, 0],
    ["permission", jsonResponse({ error: { code: 200, message: ACCESS_TOKEN } }, { status: 403 }), "ACCOUNT_ACCESS_DENIED", false, 0],
    ["malformed", new Response("not-json", { status: 200 }), "UPSTREAM_MALFORMED", false, 0],
  ];
  for (const [name, response, code, retryable, retryAfter] of cases) {
    await t.test(name, async () => {
      await assert.rejects(
        () => runInsights(baseInput, { env: env(), resolvedAuth: verified(), fetchImpl: async () => response }),
        (error) => error.code === code && error.retryable === retryable && error.retryAfterSeconds === retryAfter && !error.message.includes(ACCESS_TOKEN),
      );
    });
  }
});

test("page and aggregate MCP wire budgets are enforced", async () => {
  const request = { jsonrpc: "2.0", id: "wire", method: "tools/call", params: { name: "meta_ads_read_insights", arguments: { ...baseInput, page_size: 1, max_rows: 1 } } };
  const invoke = (size) => handleMCPRequest(request, {
    env: env(), resolvedAuth: verified(),
    fetchImpl: async () => jsonResponse({ data: [{ account_id: ACCOUNT_ID.slice(4), campaign_id: "1", campaign_name: "x".repeat(size) }] }),
  });
  const accepted = await invoke(700_000);
  assert.equal(accepted.isError, false);
  assert.ok(Buffer.byteLength(JSON.stringify({ jsonrpc: "2.0", id: request.id, result: accepted }), "utf8") <= 2 * 1024 * 1024);
  const rejected = await invoke(1_050_000);
  assert.equal(rejected.isError, true);
  assert.equal(rejected.structuredContent.error.code, "RESULT_TOO_LARGE");
  assert.ok(!JSON.stringify(rejected).includes(ACCESS_TOKEN));
});

test("MCP initialize, list, call, and identifier bounds preserve protocol shape", async () => {
  const initialized = await handleMCPRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05" } });
  assert.equal(initialized.serverInfo.name, "meta-ads-mcp-connector");
  assert.equal(initialized.protocolVersion, "2024-11-05");
  const listed = await handleMCPRequest({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} });
  assert.deepEqual(listed.tools.map((tool) => tool.name), ["meta_ads_read_insights"]);
  const called = await handleMCPRequest({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "meta_ads_read_insights", arguments: baseInput } }, {
    env: env(), resolvedAuth: verified(), fetchImpl: async () => jsonResponse({ data: [] }),
  });
  assert.equal(called.isError, false);
  await assert.rejects(() => handleMCPRequest({ jsonrpc: "2.0", id: "x".repeat(257), method: "ping" }), (error) => error.code === "INVALID_REQUEST");
});

test("auth driver exposes only provider-verified identity and target", async () => {
  const options = {
    env: env(), now: () => new Date("2026-09-19T12:00:00Z"),
    fetchImpl: async (url) => String(url).includes("/debug_token?") ? jsonResponse(debugToken()) : jsonResponse(account()),
  };
  const bootstrap = await runDriver("bearer-token", "bootstrap", {
    method: "bearer-token", flow: "credential-binding",
    requested_scopes: ["ads_read", "business_management"],
    credential_bindings: ["META_ADS_CREDENTIAL", "META_ADS_POLICY"], expected_account: ACCOUNT_ID,
  }, options);
  assert.deepEqual(bootstrap, { completed: true, account: ACCOUNT_ID, scopes: ["ads_read", "business_management"] });
  const status = await runDriver("bearer-token", "status", {}, options);
  assert.equal(status.authenticated, true);
  assert.equal(status.principal, "meta-user:1111122222");
  assert.equal(status.account, ACCOUNT_ID);
  assert.equal(status.tier, "sandbox");
  assert.match(status.attestation_reference, /^meta-live:[a-f0-9]{64}$/);
  assert.ok(!JSON.stringify(status).includes(ACCESS_TOKEN));
  assert.ok(!JSON.stringify(status).includes(APP_SECRET));
});

test("auth driver never re-anchors an absolute provider expiry", async () => {
  const providerExpiry = new Date("2026-09-19T12:00:05Z");
  const times = [
    new Date("2026-09-19T12:00:00Z"),
    new Date("2026-09-19T12:00:01Z"),
    new Date("2026-09-19T12:00:02Z"),
  ];
  let calls = 0;
  const status = await runDriver("bearer-token", "status", {}, {
    env: env(), now: () => times.shift(),
    fetchImpl: async () => (++calls === 1
      ? jsonResponse(debugToken({ expires_at: providerExpiry.valueOf() / 1_000, data_access_expires_at: providerExpiry.valueOf() / 1_000 }))
      : jsonResponse(account())),
  });
  assert.equal(status.expires_at, providerExpiry.toISOString());
  assert.equal(status.attestation_expires_at, providerExpiry.toISOString());
});

test("launchable stdio process completes initialize and tools/list", async () => {
  const child = spawn(process.execPath, [fileURLToPath(new URL("../scripts/meta_ads_mcp_server.mjs", import.meta.url))], { stdio: ["pipe", "pipe", "pipe"] });
  const responses = [];
  const lines = createInterface({ input: child.stdout, crlfDelay: Infinity });
  lines.on("line", (line) => responses.push(JSON.parse(line)));
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized", params: {} })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`);
  child.stdin.end();
  await once(child, "exit");
  assert.equal(child.exitCode, 0);
  assert.equal(responses.length, 2);
  assert.equal(responses[0].result.serverInfo.name, "meta-ads-mcp-connector");
  assert.equal(responses[1].result.tools[0].name, "meta_ads_read_insights");
});
