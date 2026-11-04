# Published registry installation verification

<!-- markdownlint-disable MD013 -->

This is a bounded first slice of #52: install the exact published instruction
release `engineering/implementation-strategy@0.1.0` through the public HTTPS
registry into clean Codex, Claude and generic **adapter targets**. It does not
launch the Codex or Claude applications, execute a skill, call a provider, prove
agent/plugin operation, or promote readiness. The website remains content-only.

This remains the default `--sample instructions` case. A separate, explicitly
pinned `--sample ga4` case adds published tool installation and credential-free
MCP startup; see [GA4 verification](published-ga4-verification.md). Its tool-aware
manifest, receipt and permission checks do not relax the instruction-only case.
The `bigquery` and `meta-ads` samples extend the same controls to two more exact
published releases; see [BigQuery and Meta Ads verification](published-tools-verification.md).
The separately pinned `code-maintenance` sample verifies a self-contained plugin
and its activated skill closure; see [published plugin verification](published-plugin-verification.md).
Released-client instruction dispatches also run on Windows AMD64; see
[Windows verification scope and evidence](published-windows-verification.md).

## Client provenance is explicit

The check has two mutually exclusive inputs:

- `--cli-release vX.Y.Z`: download the platform binary, `SHA256SUMS` and provenance
  bundle from that exact GitHub release. Check its SHA-256 and use `gh attestation
  verify` to verify the repository, release workflow, tag ref and GitHub-hosted
  builder **before** enabling execution. Missing assets, failed checksums or
  failed provenance stop the run. There is no source-build fallback.
- `--cli /absolute/path/to/binary`: copy the explicitly supplied candidate into
  the disposable workspace and label the report `candidate`. This verifies the
  published package delivery with that binary; it cannot certify a released CLI.
  The workflow builds `skills-hub-test` with the same installation-only
  profile as a release. It needs no model roots, but remains a source-built
  candidate without released-client provenance.

An exact package version is intentionally fixed for this narrow check. `latest`
does not silently change the subject. The sample must remain instruction-only,
without executable helpers. Additional packages and composed/executable products
need separately reviewed cases.

Both modes also use the repository's read-only Go manifest/receipt verifier.
It is a checker dependency, not a replacement client or a released-CLI fallback.
Go (the version in `go.mod`) is required to build this helper; package installation
still runs outside the checkout with binaries only.
Build the helper with `CGO_ENABLED=0`, as CI and the fixture tests do. It has no
C dependencies; this avoids the Go 1.22 C-linked helper's `missing LC_UUID`
startup failure on macOS 26.

## What is verified

The checker independently fetches the deployed registry, archive and manifest.
It binds both downloads to the selected release's digests and checks the served
manifest equals the manifest inside the archive. Registry/package redirects must
retain the HTTPS origin. It reads the archive without extracting it and rejects
links, traversal, duplicate members, executable instruction files and oversized
payloads.

The sample's usability and adapter declarations come from this digest-bound
historical manifest. The current registry entry only selects the exact release
record; declarations for a later release cannot reclassify the pinned subject.
The installing CLI still applies the current registry's lifecycle policy.
The helper uses the existing historical schema/parser and runtime contract digest
calculation, rather than a separate YAML subset or JSON normalization.

For each adapter, a new empty working directory, target and cache are created
outside the repository. The child receives no source path or adopting-client
credentials/runtime-home environment. All target/cache flags are explicit.
It then performs three observable transitions:

1. Cold remote install. Verify the exact file inventory and bytes against the
   independently downloaded archive, and receipt source, module, ID, version,
   runtime and artifact digest. The existing runtime receipt consumer also checks
   the tree digest against the installed files, the runtime-contract digest against
   the pinned manifest, timestamp parsing/nonzero value and strict receipt format.
   An exit code or receipt alone is insufficient.
2. Repeat without `--force`. Require rejection and an unchanged installed file
   inventory, bytes, receipt, file types, permissions, ownership, inode/link counts
   and modification/change timestamps. Links are inspected with `lstat` and
   `readlink`, without following their targets. Access time is excluded because
   verification reads can update it. A mutation fails before repair is attempted.
3. Deliberately damage the disposable `SKILL.md`, then reinstall with
   `--offline --force`. Require restoration of the published bytes. Ordinary
   HTTP(S) proxy traffic is directed at a bound, non-listening loopback socket
   during this step; this is not an OS-wide network sandbox. Installer transport
   behavior is also covered separately by existing Go tests.

The report records the host, binary digest, registry digest, selected release,
published file hashes, per-adapter progress and any failure stage. Temporary
runtime/cache trees are removed on exit. No real runtime directory is changed.

## Run locally

Released client (Python 3.12+ and GitHub CLI with attestation support):

Build the read-only checker helper once, for either client mode:

```sh
CGO_ENABLED=0 go build -o bin/published-install-verifier ./cmd/published-install-verifier
```

```sh
python3 scripts/check-published-install.py \
  --cli-release vX.Y.Z \
  --report output/published-install/released-client.json
```

Candidate client:

```sh
make cli-build-test
python3 scripts/check-published-install.py \
  --cli bin/skills-hub-test \
  --report output/published-install/candidate.json
```

Both default to the public skills registry and all three adapter targets.
Use `--registry-url` with a reviewed HTTPS registry URL
to choose a different intended deployment explicitly. Use `--runtimes generic`
for one adapter. These operations download published data; they do not publish
a release or deploy a website.

## CI and failure-space checks

The **Published registry installation** workflow is manually dispatched and
records Linux/macOS results independently, plus a Windows job for released-client
instruction checks. `release` is the default mode and
requires an exact tag. A maintainer can explicitly choose `candidate` to check
delivery before publishing an installer CLI. Evidence is uploaded even for a
failed prerequisite. Public-network availability does not gate every PR.
The workflow must first be integrated onto the repository's default branch to
appear as a dispatchable workflow in GitHub Actions.
The `sample` input defaults to `instructions`; `ga4`, `bigquery` and `meta-ads`
select the tools index and Node.js 22. Leaving `registry_url` blank selects the
sample's module index.
`code-maintenance` selects the plugins index and checks bundle registration and
first use with Node 22. Its offline check installs into a fresh target, while
requiring the existing multi-package in-place upgrade restriction to hold.

Normal PR CI runs the deterministic harness tests:

```sh
python3 -m unittest discover -s scripts/tests -p 'test_published_install.py'
```

The test model separates public metadata, downloaded bytes, the installed tree
and receipt, and released versus candidate clients. Mutations include substituted
receipt identities/digests, missing/changed/extra files, links and executable
files, constant-success CLI behavior, missing release assets, checksum mismatch,
invalid provenance, unsafe URLs, unpinned selection and malformed endpoints.
The review regressions invoke the actual Go receipt consumer for invalid/missing
integrity fields and timestamps, stop before repair on duplicate metadata
mutations, and simulate newer current-entry declarations beside an unchanged
pinned manifest. Tests build the helper from this checkout; endpoint calls use
local fixtures.
These tests validate the evidence checker; the live run supplies the separate
public-delivery evidence.

## Observed on 1 November 2026

- Candidate built from the PR #97 baseline with checker-only review fixes and explicit test-only trust
  roots: all 9 installation transitions passed on macOS arm64 (3 adapters ×
  cold install / safe duplicate rejection / offline repair). The downloaded
  archive SHA-256 was `6e0946ab1e76fc8769171fa874f8cd4373e47655559a26c90cca4914efe5ab72`.
- Released-client prerequisite: the public `v0.3.0-alpha.1` release has no attached
  assets. The check correctly failed at the client stage before execution.
  The older `v0.2.0-alpha.1` release also had no attached assets when inspected.
- All 16 deterministic checker tests pass. The installer Go suite also passes.
  Both changed workflows pass
  `actionlint` v1.7.7 (shellcheck/pyflakes integrations disabled); YAML syntax
  and `git diff --check` also pass.
- [Candidate report](reviews/published-install-2026-11-01/candidate-macos-arm64.json)
  and [released-client prerequisite report](reviews/published-install-2026-11-01/missing-release-assets.json)
  retain the measured evidence. Linux execution and successful released-binary
  provenance/download remain unproven locally; the workflow enables those checks.

The candidate report was refreshed after all three review fixes. It now records
the read-only verifier's digest, pinned-manifest runtime contract and successful
runtime receipt checks. The earlier ten-test, payload-only pass remains in Git
history at `c928fe8`; it did not cover the complete receipt or duplicate metadata.

The next release step is to publish an installer CLI through the existing release
workflow, then run `release` mode against the intended registry deployment.
This change does not manufacture that release or close #52. Its broader provider,
execution, platform and lifecycle requirements remain separate.
