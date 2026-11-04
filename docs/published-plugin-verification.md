# Published code-maintenance plugin verification

<!-- markdownlint-disable MD013 -->

This #52 slice verifies `engineering/code-maintenance-plugin@0.2.0` from the
public plugins registry with the actual `v0.4.0-alpha.1` installer. It checks a
bundle, generated adapter registration and a local first-use plan. The website
remains a content publisher; actual Codex/Claude application discovery and native
hook enforcement require separate evidence.

## Independent subject and closure

The archive SHA-256 is fixed to the reviewed immutable release:
`7d170bcd18bb15c58eacd99fc8a0def78498ca795e62eae9fc951301b308f93a`.
The shared checker verifies the selected released CLI's checksum and signed
repository/workflow/tag-bound provenance before executing it. It independently
downloads the plugins index, exact archive and manifest over the intended HTTPS
origin. Candidate builds remain labelled and cannot replace released-client proof.

Five skills at `0.1.0` are required independently of the submitted dependency
lock, receipts, registration and filesystem contents:

- `engineering/code-change-verification`
- `engineering/coverage-gap-reporter`
- `engineering/implementation-strategy`
- `engineering/pr-review-and-draft`
- `engineering/test-gap-analyzer`

Before installation, require this exact membership, canonical manifest paths,
empty transitive dependency edges, the reviewed component content hashes and a
complete checksum inventory. Reject links, special files, injected receipts,
undeclared components and unexpected empty directories. The historical Go
manifest consumer also validates schemas/layouts, dependency locks, packaging
checksums, SBOM and packaging provenance. Packaging provenance is archive data;
it is distinct from the signed GitHub build provenance for the CLI binary.

## Installed layout and receipt checks

Each Codex, Claude and generic case runs in a new temporary target/cache outside
the checkout. Installation explicitly selects `--module plugins` and the
`node22` execution capability. The check compares:

- Original archive payload bytes and normalized permissions.
- All five separately activated sibling skill trees against their archive
  copies, including complete file/directory inventories and permissions.
- Materialized `skills/<name>` copies inside the plugin against those same
  archive bytes. No repository sibling source directories are available to the CLI.
- The root remote receipt, five bundled receipts and five activated receipts.
  Require exact identities/versions/runtimes, parent artifact bindings, known
  runtime-contract digests and independently recomputed tree hashes. The Go
  receipt/closure consumer must also accept the complete installation.

The released installer normalizes delivered non-executable files, including
non-secret receipts, to mode `0644`. Receipt comparison is strict about role:
bundled receipts bind to the parent archive and cannot claim an independent
artifact or another closure. Root receipt membership cannot define its own
completeness. Installer lock files are permitted at the six known package lock
paths; they remain outside the activated-package mutation snapshots.

Compilation legitimately adds registration files and skill copies, and rewrites
the Codex root `plugin.json`. Those are explicit exceptions to raw byte equality,
validated by an independent semantic model rather than by re-running the compiler:

| Adapter | Generated registration |
| --- | --- |
| Codex | Portable root `plugin.json`, five discoverable skill directories, `.runtime/codex.json` |
| Claude | `.claude-plugin/plugin.json` with `skills: "./skills/"`, five skill directories, `.runtime/claude.json` |
| Generic | Five skill directories and `.runtime/generic.json` |

Every runtime contract must name the five pinned skills and no agents, MCP
servers or extra configuration. The Markdown completion hook must remain
**advisory** and must not appear as an executable native hook. Registration file
validation establishes the delivered layout, not that a runtime application has
loaded or enforced it.

## First use and installation transitions

Follow the published README's documented command:

```sh
node scripts/first_use.mjs examples/first-use-input.json
```

The checker invokes absolute script/input paths from an unrelated working
directory with a minimal environment and no credentials, inherited Node options
or runtime HOME. It uses the shared 15-second/256 KiB process limits and Node
network-attempt guard. This guard covers the reviewed JavaScript paths and is not
an OS-wide sandbox for arbitrary native code or spawned processes.

Require the exact plan schema, plugin identity, expected ordered steps, echoed
input and a Boolean `merge_authorized: false`. Two probes are used: the packaged
example and a controlled input containing a sentinel verification command.
Neither the proposed command nor a merge is authorized or performed. A constant
success document, extra output, changed installed files or a caught network
attempt fails the check.

For each adapter and host, require four transitions:

1. Cold remote installation of the plugin and all five skills.
2. Offline duplicate rejection without mutation of any of the six package trees.
3. Offline `--force` rejection with the explicit unsupported in-place closure
   upgrade diagnostic and no mutation of the six trees.
4. Offline installation from the validated cache into a separate empty runtime
   root, followed by complete registration/payload/receipt verification. The
   original installation must remain unchanged.

The installer deliberately rejects in-place replacement of an installed
multi-package closure until a runtime-wide atomic activation pointer is available.
An isolated fresh installation tests offline delivery without bypassing that
restriction or pretending to prove bundle upgrades/rollback.

## Run and inspect

```sh
CGO_ENABLED=0 go build -o bin/published-install-verifier ./cmd/published-install-verifier
python3 scripts/check-published-install.py \
  --sample code-maintenance --cli-release v0.4.0-alpha.1 \
  --report output/published-install/plugin/released-client.json
```

The case defaults to
`https://skills.ai-knowledge-hub.org/registry/plugins-index.json`.
Dispatch **Published registry installation** with `sample: code-maintenance`,
`client_source: release`, the exact CLI tag and a blank registry URL (or that
plugins index). CI uses Node 22 and retains artifacts ending in
`-code-maintenance`.

Across Linux and macOS, require **24/24 installation-policy transitions**, six
complete cold closures, six complete offline closures and 12 first-use probes.
Every report must identify the exact published CLI/plugin, pass checksum/
provenance and receipt/closure checks, record five dependencies per adapter,
validate registration and advisory hook labelling, and report zero network
attempts, no executed verification command and `merge_authorized: false`.

## Failure evidence and scope

The [frozen client fixtures](../scripts/tests/fixtures/code-maintenance-0.2.0/README.md)
capture registration and receipt templates from the published CLI. The test
builder recomputes integrity hashes independently; it does not use the current
compiler or checker as an expected-output calculator.

The private failure-space test-design workflow exercises coordinated omissions
of lock/receipt/registration members; wrong dependency source/parent/version/
integrity; sibling-only tampering; links and extra discovery paths; advisory-hook
promotion; constant or malformed first-use output; unexpected merge authority;
executed sentinel commands; caught network attempts; and mutations before
duplicate rejection, force rejection or offline activation. Existing instruction/
MCP suites retain their identity, provenance, process-bound and environment tests.

```sh
python3 -m unittest discover -s scripts/tests -p 'test_published_*.py'
```

Actual application use, live provider/model readiness, native enforcement of the
Markdown hook, Windows execution, general plugin coverage and user-facing bundle
upgrade/rollback/uninstall remain separate. #52 and epic #33 remain open.

## Observed on 4 November 2026

PR #107 is merged into main at `df77cca41ba0245cc0d3be1096377ec453dcd4ba`.
The [BigQuery main run](https://github.com/ai-knowledge-hub/ai-skills-guide/actions/runs/41801932633)
and [Meta Ads main run](https://github.com/ai-knowledge-hub/ai-skills-guide/actions/runs/41801937155)
passed all four host jobs: 36/36 installation transitions, 12 MCP sessions and
12 healthchecks using the released client. Main CI and Vercel passed.

The plugin passed a local macOS ARM64 released-client run for all three adapters:
12/12 installation-policy transitions, six first-use probes and complete
verification of all five dependencies in both cold and offline clean installs.
The local host used Node.js `v25.8.2`; the workflow selects Node 22.

All 62 checker tests passed (45 instruction/MCP cases and 17 plugin cases), as
did `CGO_ENABLED=0 GOTOOLCHAIN=go1.22.12 go test ./...`, `actionlint` v1.7.12 for
both changed workflows (shellcheck/pyflakes integrations disabled), and
`git diff --check`.
