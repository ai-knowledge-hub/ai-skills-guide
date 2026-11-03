# Installation-only CLI release

<!-- markdownlint-disable MD013 -->

Decision: 3 November 2026. The first public CLI installs and inspects packages.
The adopting application owns execution. The content-only website publishes
registry metadata and immutable package files; it does not become a provider
verification service.

## Build and command boundary

| Build | Commands | Authority |
| --- | --- | --- |
| `make cli-build` → `bin/skills-hub` | `list`, `search`, `info`, `validate`, `install`, `help` | Installer; no model roots |
| `make cli-build-test` → `bin/skills-hub-test` | Same installation-only profile | Unreleased candidate; no model roots |
| `make cli-build-runtime-dev` → `bin/skills-hub-runtime-dev` | Installer plus `auth`, `doctor`, `smoke`, `run-agent` and local readiness options | Explicit development profile; requires model roots |

The default Go build excludes readiness command handlers. Installer build and
release targets clear ambient `GOFLAGS`, disable workspace overrides and select
`installer_release`. Combining that tag with `runtime_readiness` fails compilation.
A command-line argument or environment variable cannot activate the deferred
profile in a released binary. The model verifier itself remains fail-closed.

`list` and `info` display static metadata; their readiness-specific `--runtime`,
`--target` and `--state-dir` options are absent from the installer. `search
--runtime` remains a declaration filter; `install --runtime` selects an adapter.
Unresolved positive executable claims are displayed conservatively, without
modifying the metadata used by package admission. Compatibility notes remain
maintainer declarations, not live provider observations.

## Installation invariant

Success means that the selected package files were installed for the selected
adapter. It does not establish application discovery, provider authentication,
model capabilities, live readiness or workflow execution. Setup guidance points
to `skills-hub help setup`, an offline guide embedded in the installer binary.
The [embedded guide](../cmd/skills-hub/setup-guide.md) supersedes retained README
readiness commands and gives a direct GA4 MCP
configuration/launch path without modifying package contents or receipts.

Installation retains HTTPS-origin restrictions, exact artifact/manifest
checksums, schemas, safe extraction, eligibility and platform checks, dependency
closure validation, staging, locks, receipts, cache verification and recovery.
Declared model and authentication requirements remain in manifests and generated
configuration. They are not converted into verified runtime facts. Existing
package eligibility restrictions remain in force.

## Preserved advanced work

Authentication drivers, doctor/smoke evidence, model-attestation verification,
agent preflight and their internal tests remain in the repository. The explicit
`runtime_readiness` development profile also retains CLI integration, including
local catalogue derivation and genuinely signed model-attestation acceptance.
Run `make cli-runtime-test` to exercise it with isolated test authority. It is
not an alternate public release artifact or a user-facing readiness guarantee.
Issuer ownership and deployment remain prerequisites for a future readiness
release, independently of the installer.

## Release and acceptance

The release retains exact tagged-source/main-ancestry checks, clean checkout,
six cross-compiles, `SHA256SUMS`, GitHub build provenance and create-only release
publication. Model-adapter keys are not release inputs. Candidate verification
remains labelled `candidate`; released verification requires actual downloaded
binaries and provenance, with no source-build fallback.

The public installation check remains bounded to the exact instruction-only
`engineering/implementation-strategy@0.1.0` package, disposable targets and
cold install / duplicate rejection / offline repair. Require both Linux and
macOS released-client reports after publication: 18 transitions across two
hosts and three adapters. This does not establish all six binary targets or
executable/composed package operation and does not close the broader #52 work.

## Verification model

Confirmed boundaries: registry metadata selects immutable release bytes;
package admission controls filesystem activation; a receipt records installed
identity/integrity; runtime evidence is a separate authority. Installation may
write only its selected targets/cache and must not invoke provider validators.

The new executable tests observe five installer commands, rejection of four
deferred commands, rejection of three readiness flags on each of two catalogue
commands, three adapter installs without model keys, static-claim projection
without metadata mutation, and rejection of conflicting build profiles.
Existing archive, receipt, recovery and release-publication tests supply the
unchanged integrity oracles. A successful mock or exit status alone is not
published-client proof; the existing independent file/receipt checker remains
required.

The retained-package setup regression is run separately with Go and Node.js 22+
(`python3 -m unittest discover -s scripts/tests -p 'test_installer_setup.py'`).
It follows a real installation's emitted help command, resolves the embedded MCP
registration to the retained GA4 package, and executes its credential-free
healthcheck from outside the checkout with provider fetches blocked. CI selects
Node 22 for this test. It verifies launch and immutable bytes, not provider access.
