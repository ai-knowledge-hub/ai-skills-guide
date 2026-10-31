// Host-owned fixture oracle. This file is not loaded from the tested package.
import assert from "node:assert/strict";
import { pathToFileURL } from "node:url";
const nodeVersion = process.versions.node;
const connector = await import(pathToFileURL(process.argv[2]).href);
try {
  for (const version of ["2024-11-05", "2025-03-26", "2025-06-18"]) {
    const initialized = await connector.handleMCPRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: version } });
    assert.equal(initialized.protocolVersion, version);
  }
  const tools = await connector.handleMCPRequest({ jsonrpc: "2.0", id: 2, method: "tools/list" });
  assert.equal(tools.tools[0].name, "ga4_run_report");
  const env = { GA4_ALLOWED_PROPERTY_IDS: "123456789" };
  const args = { property_id: "123456789", start_date: "2026-01-01", end_date: "2026-01-07", dimensions: ["sessionDefaultChannelGroup"], metrics: ["sessions"], page_size: 1, max_rows: 2 };
  let calls = 0;
  const options = {
    env,
    now: () => new Date("2026-01-08T00:00:00Z"),
    // Deliberate fixture injection: this check does not verify authentication.
    resolvedAuth: { token: "host-owned-fixture-token" },
    fetchImpl: async (url, request) => {
      assert.equal(String(url), "https://analyticsdata.googleapis.com/v1beta/properties/123456789:runReport");
      assert.equal(request.method, "POST");
      const body = JSON.parse(request.body);
      assert.deepEqual(body.dateRanges, [{ startDate: args.start_date, endDate: args.end_date }]);
      assert.deepEqual(body.dimensions, [{ name: "sessionDefaultChannelGroup" }]);
      assert.deepEqual(body.metrics, [{ name: "sessions" }]);
      assert.equal(body.limit, "1");
      assert.equal(body.offset, String(calls));
      assert.ok(calls < 2);
      const value = ["Organic Search", "Direct"][calls++];
      return new Response(JSON.stringify({ rowCount: 2, rows: [{ dimensionValues: [{ value }], metricValues: [{ value: "7" }] }] }), { headers: { "content-type": "application/json" } });
    }
  };
  const result = await connector.handleMCPRequest({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "ga4_run_report", arguments: args } }, options);
  assert.equal(result.isError, false);
  assert.equal(calls, 2);
  assert.deepEqual(result.structuredContent.rows, [{ sessionDefaultChannelGroup: "Organic Search", sessions: "7" }, { sessionDefaultChannelGroup: "Direct", sessions: "7" }]);
  assert.equal(result.structuredContent.source.api_version, "v1beta");
  assert.deepEqual(result.structuredContent.pagination, { returned_rows: 2, total_rows: 2, truncated: false, requests: 2 });
  await assert.rejects(() => connector.runReport({ ...args, property_id: "999999999" }, options), error => error.code === "PROPERTY_NOT_ALLOWED");
  assert.equal(calls, 2);
  console.log(JSON.stringify({ result: "passed", node_version: nodeVersion }));
} catch (error) {
  console.log(JSON.stringify({ result: error instanceof assert.AssertionError ? "incompatible" : "inconclusive", node_version: nodeVersion }));
}
