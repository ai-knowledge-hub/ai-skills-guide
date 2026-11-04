# Published BigQuery and Meta Ads installation and MCP startup

<!-- markdownlint-disable MD013 -->

This #52 extension verifies two exact public tools with the released installer
CLI, followed by credential-free local MCP startup. The website remains a
content-only publisher. These results describe installation into adapter targets
and process/protocol behaviour; provider access and actual Codex/Claude application
discovery remain separate.

| Sample | Pinned package | Expected MCP tool |
| --- | --- | --- |
| `bigquery` | `warehouse/bigquery-mcp-query-runner@0.2.0` | `bigquery_run_query` |
| `meta-ads` | `ads/meta-ads-mcp-connector@0.2.0` | `meta_ads_read_insights` |

## What is checked

Both cases share the [published-install trust pipeline](published-install-verification.md)
and [bounded MCP process controls introduced for GA4](published-ga4-verification.md):

1. Download the explicitly selected released CLI, initially `v0.4.0-alpha.1`.
   Verify its checksum and repository/workflow/tag-bound provenance before
   enabling execution. Candidate builds remain explicitly labelled and cannot
   replace released-client evidence.
2. Select the exact package release from the public tools index, even when its
   current entry points to a later release. Verify archive and manifest digests,
   the served manifest's equality to the archive manifest, and safe archive paths
   and types. Schema/layout validation uses the downloaded historical manifest;
   current lifecycle/deprecation policy still applies in the installing CLI.
3. Install into a fresh Codex, Claude or generic adapter target outside the source
   checkout, explicitly declaring the `node22` execution capability. Independently
   compare installed file bytes, complete inventory and normalized permissions.
   Verify receipt module, identity, version, runtime, artifact digest, historical
   runtime contract, tree digest, timestamp shape and strict receipt format.
4. Follow the installed CLI's emitted `skills-hub help setup` redirect. For these
   two tools its compatible setup guidance is prose. Require the exact pinned
   tool section, binding names and Node command, then validate the installed
   `.mcp.json` against that command. Reject alternate commands, paths, working
   directories, extra servers and embedded credentials.
5. Translate the package's documented relative registration to the absolute,
   symlink-safe form below. From an unrelated working directory, run healthcheck,
   then an actual ordered MCP `initialize` / initialized notification / `tools/list`
   exchange. Check the pinned server identity, protocol, response IDs, tool name,
   exact required inputs, property types and the selected input bounds. BigQuery
   and Meta Ads cannot substitute for each other merely because they share a version.
6. Require startup to preserve installed files, metadata and receipt. Require a
   second install without `--force` to fail without mutation. Damage the disposable
   server script, then require `--offline --force` to restore the published bytes
   and a valid receipt.

The read-only Go helper admits only the three explicitly pinned setup-required
MCP contracts (GA4, BigQuery and Meta Ads at `0.2.0`). It checks the expected Node
entrypoint, healthcheck, platform/runtime declarations, self-contained artifact,
credential bindings and scopes. Its default instruction-only checks stay strict.

## Direct registration used by the check

The retained packages register a relative script with `cwd: "."`. The released
CLI's setup guide directs users to those same scripts. This companion registration
adapts that documented command for clients whose working directory differs from
the package root; the archive's original `.mcp.json` and receipt are preserved.
Replace each placeholder with its absolute installed directory:

```json
{
  "mcpServers": {
    "bigquery": {
      "type": "stdio",
      "command": "node",
      "args": ["--preserve-symlinks-main", "<absolute-installed-bigquery-directory>/scripts/bigquery_mcp_server.mjs"]
    },
    "meta-ads": {
      "type": "stdio",
      "command": "node",
      "args": ["--preserve-symlinks-main", "<absolute-installed-meta-ads-directory>/scripts/meta_ads_mcp_server.mjs"]
    }
  }
}
```

Real queries/reports still require the package's documented runtime bindings:
`BQ_CREDENTIAL` and `BQ_POLICY`, or `META_ADS_CREDENTIAL` and `META_ADS_POLICY`.
Use the adopting application's secret/environment settings for those values.
The verification check supplies none of them and never issues `tools/call` or
the provider smoke command. Startup success leaves each tool `setup-required`.

## Isolation and independent oracles

The server receives a minimal PATH environment. Inherited HOME, Node options,
Google credential paths, credential envelopes and policy bindings are excluded.
A harness preload rejects and records `fetch` and ordinary Node HTTP/TLS/socket
connection attempts, even when the server catches the error. This checks the
retained JavaScript servers' known paths; it is not an OS-wide sandbox for native
code or spawned subprocesses. Each process has a 15-second deadline and 256 KiB
combined output budget, with cleanup for timeout, broken pipes and early exit.

Fixture tests use retained package archives and the actual Node server processes.
Runtime-contract goldens come from the frozen released CLI's public installation
results, rather than the Go verifier under test. Mutations cover receipt identity
and integrity, modules and permissions, other valid package substitution, altered
prose/registration, wrong server/tool/required inputs, weakened input bounds,
swallowed network attempts, inherited credentials and startup writes. The shared
instruction/GA4 suites cover malformed output, time/output exhaustion, early-exit
cleanup, source-mode/provenance failures and GitHub token origin restrictions.

## Run locally or on GitHub

```sh
CGO_ENABLED=0 go build -o bin/published-install-verifier ./cmd/published-install-verifier
python3 scripts/check-published-install.py \
  --sample bigquery --cli-release v0.4.0-alpha.1 \
  --report output/published-install/tools/bigquery.json
python3 scripts/check-published-install.py \
  --sample meta-ads --cli-release v0.4.0-alpha.1 \
  --report output/published-install/tools/meta-ads.json
```

Both default to `https://skills.ai-knowledge-hub.org/registry/tools-index.json`.
Run **Published registry installation** separately with `sample: bigquery` and
`sample: meta-ads`, `client_source: release`, exact `cli_release`, and a blank
`registry_url` (or the tools index URL). CI selects Node 22 on Linux and macOS.
Artifact names end in `-bigquery` or `-meta-ads`; instruction and GA4 names stay
compatible with earlier runs.

Across both tools and both hosts, require **36/36 installation transitions**, 12
MCP sessions and 12 healthchecks. Every report must identify the exact tool,
`module: tools`, `client_kind: release`, the actual host/Node version, passed
checksum/provenance and receipt verification, all four startup checks as passed,
zero network attempts, no supplied provider credentials and no provider-call proof.

Run deterministic checks with:

```sh
python3 -m unittest discover -s scripts/tests -p 'test_published_*.py'
```

Windows/other architectures, actual application use, composed plugin activation,
upgrade/rollback/uninstall features and live provider identity/account/scope/
revocation verification remain outside this slice. The wider #52 and epic #33
remain open.

## Observed on 4 November 2026

PR #106 is merged. The [GA4 workflow on main](https://github.com/ai-knowledge-hub/ai-skills-guide/actions/runs/41792155134)
at `277116c8fac904d2ac6d720a8a7d9b3261f1d8a3` passed on Linux x86-64 and macOS
ARM64: 18/18 installation transitions, six MCP sessions and six healthchecks with
the released CLI. Main CI and Vercel also passed.

Both new tools passed local macOS ARM64 released-client checks: nine installation
transitions, three MCP sessions and three healthchecks per tool, with no supplied
credentials or detected startup network attempts. The Linux/macOS Node 22 workflow
matrix supplies the separate CI evidence.

The final matrix passed at implementation commit
`c56b9498bcb46ffc025d19ef062d26de8add5423`, using Node.js `v22.23.4` and the actual
released `v0.4.0-alpha.1` client:

| Tool | Workflow | Linux x86-64 report | macOS ARM64 report |
| --- | --- | --- | --- |
| BigQuery | [Run 41793607555](https://github.com/ai-knowledge-hub/ai-skills-guide/actions/runs/41793607555) | [JSON](reviews/published-tools-2026-11-04/ci-bigquery-ubuntu.json) | [JSON](reviews/published-tools-2026-11-04/ci-bigquery-macos.json) |
| Meta Ads | [Run 41793611156](https://github.com/ai-knowledge-hub/ai-skills-guide/actions/runs/41793611156) | [JSON](reviews/published-tools-2026-11-04/ci-meta-ads-ubuntu.json) | [JSON](reviews/published-tools-2026-11-04/ci-meta-ads-macos.json) |

All four reports passed exact CLI/package identity, checksum/provenance and
receipt checks. They confirm **36/36 installation transitions**, **12 actual MCP
sessions** and **12 healthchecks**, with no supplied provider credentials and
zero detected startup network attempts. Candidate builds were skipped in all
four jobs. These results do not promote Google or Meta provider readiness.

All 45 deterministic checker tests passed (17 instruction/client cases, 16 GA4
cases and 12 parameterized BigQuery/Meta Ads cases), as did
`CGO_ENABLED=0 GOTOOLCHAIN=go1.22.12 go test ./...`, `actionlint` v1.7.12 for both
changed workflows (shellcheck/pyflakes integrations disabled), and `git diff --check`.
