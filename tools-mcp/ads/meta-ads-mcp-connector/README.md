# Meta Ads MCP Connector

A dependency-free Node.js 22 stdio MCP server for bounded, account-scoped, read-only Meta Ads Insights reports.

## Configure

Keep both bindings in the runtime environment:

```bash
export META_ADS_CREDENTIAL='{"type":"access_token","access_token":"...","app_id":"...","app_secret":"..."}'
export META_ADS_POLICY='{"allowed_ad_account_ids":["act_1234567890"],"allowed_business_ids":["1234567890"],"max_date_days":90,"max_rows":500,"max_pages":10,"max_timeout_ms":30000,"graph_api_version":"v24.0","environment_tier":"sandbox"}'
```

The policy binds one business and one ad account per connector instance. The credential envelope cannot declare its own principal, permissions, business, or account. The packaged auth driver obtains those claims from Meta's token debugger and live ad-account object.

## Install and register

```bash
./bin/skills-hub install \
  --module tools \
  --entry ads/meta-ads-mcp-connector@0.2.0 \
  --runtime codex
```

The packaged `.mcp.json` launches `node scripts/meta_ads_mcp_server.mjs` from the package root.

## Tool input

```json
{
  "ad_account_id": "act_1234567890",
  "level": "campaign",
  "start_date": "2026-09-01",
  "end_date": "2026-09-07",
  "fields": ["campaign_id", "campaign_name", "impressions", "clicks", "spend", "actions", "action_values"],
  "breakdowns": ["publisher_platform"],
  "page_size": 50,
  "max_rows": 500,
  "timeout_ms": 30000
}
```

Metric values remain strings so currency and large counts are not silently rounded. Action arrays retain their `action_type` and provider value. Every row retains `date_start`, `date_stop`, and `attribution_setting`; source metadata retains the account timezone, currency, Graph API version, provider response timestamps, and retrieval time.

## Safety boundary

- The connector exposes one Insights read tool and no mutation endpoint.
- Runtime policy is the sole authority for allowed business, ad account, fields, date span, rows, pages, and timeout.
- Token identity, application binding, scopes, and expiry come from `debug_token`; business ownership and account access come from a live account read.
- Provider pagination URLs are ignored. Only bounded cursors are copied into a newly constructed canonical Graph URL.
- Provider messages are never returned, because they may contain tokens or private identifiers.
- The final JSON-RPC result is bounded after accounting for both text and structured MCP representations.

Run deterministic coverage with:

```bash
node --test tests/meta_ads_mcp_server.test.mjs
```

## Provider evidence gate

Mocks prove deterministic behavior only. Release promotion additionally requires a clean-client smoke read against an approved Meta sandbox ad account and current private evidence bound to the retained artifact, credential generation, provider principal, business, and ad account. Do not commit those private identifiers; publish only a non-promotional redacted summary unless an approved audit authority can verify a privacy-safe commitment.
