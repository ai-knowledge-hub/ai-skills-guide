# Test Prompts

1. Query sessions and conversions for an allowed property over an explicit seven-day range.
2. Compare two explicit weekly windows by `sessionDefaultChannelGroup` without exceeding 500 rows per call.
3. Reject a metric or dimension outside the packaged allowlist before contacting Google.
4. Reject an otherwise valid query for a property outside `GA4_ALLOWED_PROPERTY_IDS`.
5. Return normalized rows, pagination state, provider identity, API version, and retrieval freshness.
6. Reject a credential envelope that attempts to self-declare its principal or granted scopes.
7. Reject individually valid pages when the final JSON-RPC response, including text and structured MCP representations, would exceed 2 MiB.
