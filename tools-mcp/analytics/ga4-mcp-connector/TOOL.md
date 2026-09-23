# GA4 MCP Connector

## Purpose

Provide normalized, read-only Google Analytics 4 reporting through a launchable local MCP server.

## Capabilities

- Run reports for explicitly allowed GA4 properties and fields.
- Page within fixed request and result budgets.
- Return structured rows, provider identity, and retrieval freshness.

## Guardrails

- Reject cross-property, unknown-field, future-date, oversized, and over-budget requests before returning data; bound the complete JSON-RPC response across both MCP result representations.
- Use the Analytics Data API read-only scope and expose no administration or mutation methods.
- Derive readiness principal and granted scopes from Google's token-info response, never from caller-supplied credential metadata; accept `azp`/`aud` identity only for the selected service-account credential type.
- Normalize provider failures without copying credential or upstream error payloads.

## Tool

`ga4_run_report` executes a bounded Google Analytics Data API `runReport` request.

Required arguments:

- `property_id`: numeric GA4 property ID; must be in `GA4_ALLOWED_PROPERTY_IDS`.
- `start_date`, `end_date`: inclusive `YYYY-MM-DD` dates, at most 366 days and not in the future.
- `metrics`: 1–10 packaged metric names.

Optional arguments:

- `dimensions`: up to 10 packaged dimension names.
- `page_size`: 1–250, default 100.
- `max_rows`: 1–1,000, default 500.

## Supported fields

Dimensions: `city`, `country`, `date`, `deviceCategory`, `eventName`, `firstUserDefaultChannelGroup`, `firstUserSourceMedium`, `landingPagePlusQueryString`, `newVsReturning`, `pagePath`, `sessionDefaultChannelGroup`, `sessionSourceMedium`.

Metrics: `activeUsers`, `advertiserAdCost`, `conversions`, `eventCount`, `keyEvents`, `newUsers`, `purchaseRevenue`, `screenPageViews`, `sessions`, `totalRevenue`, `totalUsers`.

## Output and failures

Successful calls return `skills-hub.ga4-report/v1` with the property, requested date range, ordered columns, normalized string-valued rows, pagination state, provider/API identity, retrieval timestamp, and freshness classification.

Failures return a stable code, safe message, and retryability flag. Expected codes include `INVALID_ARGUMENT`, `PROPERTY_NOT_ALLOWED`, `DATE_RANGE_NOT_ALLOWED`, `AUTH_MISSING`, `AUTH_INVALID`, `AUTH_EXPIRED`, `AUTH_REVOKED`, `AUTH_IDENTITY_UNVERIFIED`, `AUTH_SCOPE_UNVERIFIED`, `AUTH_SCOPE_INVALID`, `PROPERTY_ACCESS_DENIED`, `QUOTA_EXHAUSTED`, `UPSTREAM_TIMEOUT`, `UPSTREAM_MALFORMED`, `UPSTREAM_RESPONSE_TOO_LARGE`, `RESULT_TOO_LARGE`, and `QUERY_BUDGET_EXCEEDED`.

## Authority and effects

This connector has read authority only. It cannot create or update GA4 properties, audiences, events, annotations, access policy, or account configuration. A successful report is observed provider data, not approval to trigger a downstream write.
