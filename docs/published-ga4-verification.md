# Published GA4 installation and credential-free MCP startup

<!-- markdownlint-disable MD013 -->

This focused #52 slice checks `analytics/ga4-mcp-connector@0.2.0` from the public
tools registry using an actual released CLI. It verifies installation and local
MCP process startup. It does not authenticate to Google, retrieve reports, launch
the Codex/Claude applications, write readiness evidence or change catalogue claims.

## Pinned subjects and authority

- CLI: an explicitly selected GitHub release, initially `v0.4.0-alpha.1`.
  The shared checker verifies its checksum and repository/workflow/tag-bound
  provenance before enabling execution. There is no source-built fallback.
- Package: exact `analytics/ga4-mcp-connector@0.2.0`, regardless of a newer
  current release. The tools index selects the version-bound artifact and manifest
  digests; package classification and adapter declarations come from the downloaded
  historical manifest. Current lifecycle/deprecation policy still applies in the
  installing CLI.
- The read-only Go helper accepts this specific tool contract under `--module
  tools`: a self-contained, setup-required GA4 MCP server requiring `GA4_CREDENTIAL`
  and the read-only analytics scope, with the declared Node entrypoint and
  healthcheck. It retains its stricter instruction-only default.
- Neither the helper nor setup check can certify provider authentication, model
  capability or live readiness. Actual provider deadlines and authority checks
  remain separate from this content/distribution proof.

## Observable checks

For each Codex, Claude and generic adapter target, the checker creates a fresh
temporary work directory, target and cache outside the source checkout:

1. Install the public tool archive with `--module tools --execution-runtime node22`.
   Independently compare the served manifest to the archive manifest, reject
   unsafe archive members, and verify the installed file bytes, complete directory
   inventory, normalized permissions and install receipt. Tool scripts may be
   executable as declared in the archive; instruction-only files remain subject
   to their existing non-executable restriction.
2. Follow the install output's `skills-hub help setup` redirect using that same
   released binary. Parse and validate its exact GA4 stdio registration, including
   the absolute script path and `--preserve-symlinks-main`, through a temporary
   symlink to the installed package. The adopting application's working directory
   is outside the package. Node.js 22 or newer is selected explicitly; CI uses 22.
3. Run the registered command with `--healthcheck`. Then start a separate actual
   MCP process, await `initialize` for protocol `2025-06-18`, send the initialized
   notification, and request `tools/list`. Validate response IDs, JSON-RPC shape,
   pinned server identity, tools capability and the `ga4_run_report` input contract.
   A healthcheck or constant success alone cannot pass this check.
4. Require startup to leave installed files, permissions, metadata and receipt
   unchanged. Repeat installation without `--force` and require rejection without
   any mutation. Damage the disposable server script and require `--offline
   --force` to restore the published bytes and a valid receipt.

The server receives only a minimal PATH environment: no adopting-client HOME,
credential bindings, Google credential paths or inherited NODE_OPTIONS. A harness
preload rejects and records attempts through `fetch` and ordinary Node HTTP/TLS/
socket connection APIs, even when the server catches the exception. Any recorded
attempt fails the run. No `tools/call` or provider smoke command is issued.
This is a check of the retained server's known network paths, not an OS-wide
sandbox for arbitrary native code or spawned subprocesses.

Each server process has a 15-second deadline and a 256 KiB combined stdout/stderr
budget. Invalid/trailing output, process failure or a timeout cannot become a
successful report. Local fixture tests exercise these failure paths and child
cleanup. No credentials or provider response bodies are recorded.

## Run and inspect

```sh
CGO_ENABLED=0 go build -o bin/published-install-verifier ./cmd/published-install-verifier
python3 scripts/check-published-install.py \
  --sample ga4 --cli-release v0.4.0-alpha.1 \
  --report output/published-install/ga4/released-client.json
```

The default index for this case is
`https://skills.ai-knowledge-hub.org/registry/tools-index.json`.
An explicit `--registry-url` must still satisfy the HTTPS/origin restrictions.

Dispatch **Published registry installation** with `sample: ga4`,
`client_source: release`, `cli_release: v0.4.0-alpha.1`, and a blank `registry_url`
(or the tools URL above). Require Linux and macOS reports. GA4 artifact names have
the `-ga4` suffix; existing instruction artifact names remain unchanged.

Across the two hosts, require **18/18 installation transitions**, six successful
MCP sessions and six healthchecks. Reports must identify `sample: ga4`, `module:
tools`, the exact package version, `client_kind: release`, successful checksums and
provenance, and the real host/Node version. Every startup result must record all
four checks as passed, zero network attempts, `credentials_supplied: false` and
`provider_calls_verified: false`. Candidate mode remains explicitly labelled and
cannot substitute for released-client evidence.

## Deterministic failure coverage

```sh
python3 -m unittest discover -s scripts/tests -p 'test_published_install.py'
python3 -m unittest discover -s scripts/tests -p 'test_published_ga4.py'
```

The private failure-space test-design workflow models four independent boundaries:
published identity/digests, installed bytes/permissions/receipt, setup-to-process
registration, and process/protocol/network effects. Tests mutate module, identity,
version, runtime, source and integrity receipt fields; package bytes, links,
permissions and extra paths; setup commands/paths/embedded credentials; protocol
IDs/versions/tool contracts; swallowed network attempts; constant health success;
startup mutations; malformed/extra output; and deadline/output exhaustion.
The positive local process case uses retained package bytes, not a mock MCP result.
Public delivery and released-client provenance are established separately by the
live workflow, not by these local fixtures.

Windows, other CPU architectures, actual application discovery, BigQuery/Meta Ads,
plugin composition, live provider identity/account/scope/revocation checks and the
advanced readiness development profile remain outside this focused PR. #52 stays open.

## Observed on 3 November 2026

The [local macOS ARM64 released-client report](reviews/published-ga4-2026-11-03/local-macos-release.json)
records 9/9 installation transitions, three healthchecks and three actual MCP
sessions with the downloaded `v0.4.0-alpha.1` binary. Checksums/provenance passed;
all startup checks passed with zero network attempts and no supplied credentials.
This local host used Node.js `v25.8.2`; the workflow explicitly selects Node 22.
It is local evidence, not a substitute for the Linux/macOS workflow matrix.

All 32 deterministic checker tests passed (16 instruction/release-boundary cases
and 16 GA4 cases), as did `CGO_ENABLED=0 GOTOOLCHAIN=go1.22.12 go test ./...`,
`actionlint` v1.7.12 for both changed workflows (shellcheck/pyflakes integrations
disabled), and `git diff --check`.
