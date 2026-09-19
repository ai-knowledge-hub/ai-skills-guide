# BigQuery MCP Query Runner

## Purpose

Run one parameterized, read-only GoogleSQL query through a launchable stdio MCP server without allowing the caller to choose its own security or cost boundary.

## Authority boundary

The runtime owner configures allowed data projects, one billing project, datasets, locations, maximum bytes billed, row count, timeout, and an explicit `sandbox` or `live` provider tier. Tool arguments can only narrow those limits. Every external table must use a backtick-qualified `dataset.table` or `project.dataset.table` name in the configured allowlist.

## Query lifecycle

1. Lex SQL independently of strings and comments. Reject scripts, terminators, DDL, DML, external queries, and unsupported table references.
2. Require declared named parameters to exactly match `@name` placeholders.
3. Submit a BigQuery dry run and reject estimates above `BQ_MAX_BYTES_BILLED`.
4. Execute through `jobs.insert` with a caller-known job ID, `maximumBytesBilled`, and provider-side `jobTimeoutMs`.
5. Poll and paginate inside timeout, page, row, upstream-body, and final MCP-wire limits; treat nonterminal `errors` as warnings until the terminal job receipt is checked.
6. Request cancellation by the known job identity after timeout, ambiguous submission, or polling failure.
7. Return typed rows plus job, cost, cache, pagination, source, and freshness diagnostics.

## Guardrails

- Accept only a single `SELECT` or `WITH` statement after lexical analysis outside strings and comments.
- Require exact named parameter declarations and qualified external table identifiers.
- Treat `BQ_POLICY` as the sole authority for project, dataset, location, cost, row, timeout, and user-OAuth permissions.
- Reject destinations, DDL, DML, scripts, external queries, response overflows, and terminal jobs whose authoritative `status.errorResult` reports failure; retain safe reason codes from nonterminal warning arrays as diagnostics.
- Redact provider failures and never return SQL parameters, access tokens, private keys, or workload subject tokens.

## Authentication

- `service_account`: signed JWT exchange for the BigQuery scope.
- `external_account`: workload subject token from an explicit environment binding, Google STS exchange, and service-account impersonation.
- `application_default`: Google metadata-server access token.
- `authorized_user` or `access_token`: disabled unless the runtime owner explicitly sets `BQ_ALLOW_USER_OAUTH=1`.

Readiness never trusts identity or scopes written into the credential envelope. It uses Google's token-info response and a live bounded BigQuery probe.

Mocks prove deterministic behavior only. Release completion additionally requires a clean-client sandbox smoke run and current artifact-, credential-generation-, and provider-target-bound evidence produced by the repository readiness workflow.
