# BigQuery MCP Query Runner

A dependency-free Node.js 22 stdio MCP server for bounded, parameterized, read-only BigQuery queries.

## Configure

Set all execution authority in the runtime environment:

```bash
export BQ_CREDENTIAL='{"type":"application_default","source":"metadata","account":"billing-project"}'
export BQ_POLICY='{"allowed_project_ids":["data-project","billing-project"],"billing_project_id":"billing-project","allowed_datasets":["data-project.analytics","data-project.reporting"],"allowed_locations":["EU","europe-west2"],"max_bytes_billed":1000000000,"max_rows":500,"max_timeout_ms":30000,"allow_user_oauth":false,"environment_tier":"sandbox"}'
```

Service-account, workload-identity, metadata application-default, and explicitly approved user OAuth envelopes are documented in [TOOL.md](TOOL.md). Secrets stay in runtime bindings; none are copied into the package.

## Install and register

```bash
./bin/skills-hub install \
  --module tools \
  --entry warehouse/bigquery-mcp-query-runner@latest \
  --runtime codex
```

The packaged `.mcp.json` launches `node scripts/bigquery_mcp_server.mjs` from the package root.

## Tool input

```json
{
  "query": "SELECT channel, SUM(spend) AS spend FROM `data-project.analytics.performance` WHERE event_date BETWEEN @start AND @end GROUP BY channel",
  "parameters": [
    {"name": "start", "type": "DATE", "value": "2026-09-01"},
    {"name": "end", "type": "DATE", "value": "2026-09-07"}
  ],
  "project_id": "data-project",
  "billing_project_id": "billing-project",
  "location": "EU",
  "max_rows": 100,
  "page_size": 100,
  "timeout_ms": 30000
}
```

Integer, numeric, temporal, geography, bytes, and JSON-like warehouse values remain strings when JavaScript conversion would lose information. `FLOAT64` becomes a finite number and `BOOL` a boolean; the accompanying schema records every BigQuery type.

## Safety notes

- `LIMIT` is not a cost control. The connector always dry-runs and repeats `maximumBytesBilled` on execution.
- Every execution uses a caller-known BigQuery job ID and provider-side `jobTimeoutMs`; ambiguous submissions are cancelled by that identity and reported without automatic replay.
- Dataset and project policy comes only from runtime configuration.
- Query and parameter values are not placed in returned diagnostics.
- Quota, authentication, permission, timeout, malformed response, pagination, and result-size failures use stable redacted error codes.

Run deterministic coverage with:

```bash
node --test tests/bigquery_mcp_server.test.mjs
```

## Provider evidence gate

Deterministic tests and mocked Google responses do not count as provider evidence. Before publishing this release as verified usable, install the retained artifact on a clean client, configure `environment_tier: "sandbox"`, run authentication status and the packaged smoke command against the sandbox billing project, and publish the resulting artifact-, credential-generation-, and provider-target-bound readiness evidence. No live observation is claimed by this source tree.
