# Test Prompts

1. Read last-week campaign performance for the configured sandbox ad account and preserve its reporting dates and account attribution setting.
2. Return spend, impressions, clicks, purchases, and purchase value by campaign without converting provider decimal strings to binary floating point.
3. Reject an ad account outside `META_ADS_POLICY` before issuing an Insights request.
4. Explain a revoked or expired token using a stable redacted diagnostic without returning Meta's raw provider message.
5. Page through a two-page report using only Meta's bounded cursor while ignoring the provider-supplied next URL.
6. Stop when the configured page, row, timeout, upstream-body, or final MCP-wire budget is reached.
7. Reject a token that lacks `ads_read` or `business_management`, belongs to another app, or resolves the account to another business.
