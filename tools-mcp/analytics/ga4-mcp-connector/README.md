# GA4 MCP Connector

A launchable, read-only MCP server for the Google Analytics Data API. It exposes one bounded tool, `ga4_run_report`, and ships without third-party runtime dependencies.

## Safety boundary

- Only `properties/*:runReport` is called; no GA4 Admin or mutation API is exposed.
- Every property must appear in `GA4_ALLOWED_PROPERTY_IDS`.
- Dimensions and metrics come from packaged allowlists.
- Date ranges are limited to 366 days, pages to 250 rows, results to 1,000 rows, and each call to 10 upstream requests.
- Requests time out after 15 seconds by default; each upstream page and the complete JSON-RPC response are limited to 2 MiB.
- Provider errors are normalized and never include credential payloads.

## Requirements

- Node.js 22 or newer.
- The Google Analytics Data API enabled in the selected Google Cloud project.
- Viewer access to every configured GA4 property.
- OAuth scope `https://www.googleapis.com/auth/analytics.readonly`.

## Credential envelope

Store one JSON envelope in a runtime-owned environment source and bind it as `GA4_CREDENTIAL`. Do not add this value to MCP configuration, shell history, source files, or the package.

For an existing short-lived OAuth access token:

```json
{"type":"access_token","access_token":"<runtime-provided>","expires_at":"2026-09-17T12:00:00Z","account":"123456789"}
```

For a user OAuth grant, complete Google's installed-application authorization flow with offline access and store the resulting client and refresh-token fields in an `authorized_user` envelope. The connector exchanges the refresh token at runtime; it never writes the refreshed access token:

```json
{"type":"authorized_user","client_id":"<runtime-provided>","client_secret":"<runtime-provided>","refresh_token":"<runtime-provided>","account":"123456789"}
```

For server workloads, use a dedicated service account with Viewer access to only the required properties. Add only the non-secret `account` property ID to the downloaded fields accepted below:

```json
{"type":"service_account","client_email":"ga4-reader@example-project.iam.gserviceaccount.com","private_key":"<runtime-provided>","private_key_id":"<runtime-provided>","project_id":"example-project","account":"123456789"}
```

The envelope cannot declare its own principal or granted scopes. During readiness checks the driver exchanges or resolves the access token, asks Google's token-info endpoint to attest its principal, scopes, and lifetime, and uses that same token for the live property-access query. User tokens require an attested email or subject; normally scoped service-account tokens use their attested, non-ambiguous `azp`/`aud` identifier.

Rotate service-account keys and revoke user grants at Google. Increment a non-secret generation value whenever the credential changes.

## Install and configure readiness

```bash
./bin/skills-hub install --module tools --entry analytics/ga4-mcp-connector@0.2.0 --runtime codex

export GA4_CREDENTIAL_SOURCE='<credential-envelope>'
export GA4_CREDENTIAL_GENERATION='generation-1'

./bin/skills-hub auth configure analytics/ga4-mcp-connector@0.2.0 \
  --module tools --runtime codex --method bearer-token --account 123456789 \
  --binding GA4_CREDENTIAL=env:GA4_CREDENTIAL_SOURCE \
  --generation GA4_CREDENTIAL=env:GA4_CREDENTIAL_GENERATION
```

Use `--method service-account` for its envelope. The packaged driver performs a live one-row query during bootstrap and status checks. The readiness profile stores only environment variable names and a generation reference.

## Register and run

The package includes `.mcp.json`. Ensure the MCP runtime supplies these environment values to the server process:

```bash
export GA4_CREDENTIAL="$GA4_CREDENTIAL_SOURCE"
export GA4_ALLOWED_PROPERTY_IDS='123456789,987654321'
node scripts/ga4_mcp_server.mjs --healthcheck
node scripts/ga4_mcp_server.mjs --smoke
```

Run the deterministic suite with `node --test tests/ga4_mcp_server.test.mjs`. It covers MCP initialize/list/call, process launch, pagination, truncation, quotas, timeouts, malformed responses, revoked credentials, property isolation, and auth-driver semantics against a mocked provider.
