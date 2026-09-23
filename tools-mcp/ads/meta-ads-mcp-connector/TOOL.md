# Meta Ads MCP Connector

## Purpose

Run one bounded, account-scoped Meta Marketing API Insights read through a launchable stdio MCP server.

## Authority boundary

The runtime owner configures the allowed business and ad accounts, Graph API version, field allowlist, date span, row count, page count, timeout, and sandbox/live tier. Tool callers can only narrow those limits. The credential envelope supplies token material but cannot attest identity, permissions, business membership, or account access.

## Read lifecycle

1. Validate the complete tool request against packaged and runtime policy.
2. Ask Meta `debug_token` to attest application binding, token principal, permissions, validity, and expiry.
3. Read the selected ad-account object and require its account and owning business to match policy.
4. Read Insights with `use_account_attribution_setting=true`.
5. Reconstruct every page on the trusted Graph origin from a bounded cursor.
6. Return normalized rows with provider reporting dates, attribution, account timezone/currency, provider response dates, and retrieval time.

## Guardrails

- Only `GET` reads are implemented; there is no generic Graph request tool.
- Account, business, fields, breakdowns, date span, rows, pages, timeout, upstream body, and final MCP wire size are bounded.
- One absolute deadline governs all pages.
- Provider response messages and URLs are not returned or followed.
- Rate limit, expiry, revocation, permission, timeout, malformed response, and budget failures use stable redacted codes.

## Authentication

`META_ADS_CREDENTIAL` contains an access token plus the application identity and secret needed for Meta's token debugger. `META_ADS_POLICY` contains non-secret runtime authority. The packaged readiness driver verifies `ads_read` and `business_management`, the token principal and expiry, and the exact business/ad-account relationship. Neither binding is written into the package or readiness profile.

Deterministic tests are not provider evidence. Promotion requires a current sandbox smoke observation bound to the retained archive and private provider target, or a privacy-safe commitment verifiable by an approved audit authority.
