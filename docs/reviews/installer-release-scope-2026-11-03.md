# Installer release scope verification — 3 November 2026

<!-- markdownlint-disable MD013 -->

## Subject and outcome

Implementation source: `6f2b4bd819c9eabd981b831ec521ae25d9968c7f`, based on
merged `dev` at `aaa2073bc5f869f8d85e468616af11aae8f6c055`.
The normal CLI and release builder produce the installation-only profile without
model keys. The advanced commands and signed-model verification remain in the
explicit development profile. No public tag or GitHub release was created by
this verification.

## Evidence and authority map

| Boundary | Authority and commit | Observed result |
| --- | --- | --- |
| Installer command surface | Compiled profile, before argument effects | Four deferred commands rejected; six readiness flag/command combinations rejected without target/state creation |
| Declared status | Read-only catalogue projection, separate from admission input | Root, helper and capability claims demoted without changing the selected metadata; active/deprecated cases covered |
| Local package installation | Existing manifest admission and receipt commit | Three instruction adapter installs and one authenticated GA4 tool install pass without model/provider credentials; interpreter traps remain untouched |
| Advanced runtime authority | Explicit development profile and signed attestation | Preserved genuine-signature binary integration and negative authority tests pass |
| Release build | Exact local tag, clean checkout, explicit installer profile | Six binaries and six checksums pass without model roots, despite ambient readiness tags/workspace overrides |
| Published delivery | Independently fetched public archive/manifest plus installed bytes and receipt | Native release-builder candidate passes all nine cold/duplicate/offline transitions |
| GitHub publication | Workflow source guards, provenance and create-only publisher | 25 deterministic release-boundary tests pass; actual GitHub dispatch/attestation/publication remains unexecuted |

The source tree must remain clean while building. The isolated build exposed
that generated `dist/` files marked subsequent Go binaries as dirty; `/dist/`
is now ignored like `bin/`. After that correction, all six build-info records
identify the same source revision, `installer_release` tag, `CGO_ENABLED=0`, and
`vcs.modified=false`. Non-empty output rejection and source guards remain intact.

## Commands and results

- `go test ./...` — passes, including existing archive/admission/receipt/recovery and internal readiness suites.
- `make cli-runtime-test` — passes, including the built-binary signed-model-attestation integration test.
- `make cli-build cli-build-test` with all three model-root variables unset and ambient `GOFLAGS=-tags=runtime_readiness` — passes with installer command surface.
- `python3 -m unittest discover -s scripts/tests -p 'test_cli_release.py'` — 25 pass.
- `python3 -m unittest discover -s scripts/tests -p 'test_published_install.py'` — 16 pass.
- Actionlint v1.7.7 on all four changed workflows — passes (shellcheck/pyflakes integrations disabled); changed shell scripts pass `bash -n`.
- Changed Markdown and `git diff --check` — pass.

The actual release builder was run in a disposable local clone with a
local-only `v0.0.0-installer-check` tag. All three model-root variables were
removed; `GOFLAGS=-tags=runtime_readiness` and a nonexistent `GOWORK` path were
supplied. The builder explicitly selects the installer profile and clears those
ambient build overrides. Six cross-compiles and SHA-256 verification pass.
Only the native macOS ARM64 binary was executed.

That stripped, CGO-disabled native artifact was then supplied to the existing
published-install checker as `--cli`, so the retained report correctly says
`client_kind: candidate`. The checker runs outside the checkout with disposable
runtime directories, independently checks published bytes/receipts and performs
cold install, duplicate rejection without mutation, and offline repair for
Codex, Claude and generic targets: **9/9 transitions passed**.

[Candidate report](installer-release-scope-2026-11-03/candidate-macos-arm64.json) · [Six binary build-info records](installer-release-scope-2026-11-03/binary-build-info.txt)

## Bounded counter-review and remaining evidence

The selected failure space covers four unavailable commands, three unavailable
flags on two catalogue commands, three adapter destinations, one credentialed
tool with subprocess traps, active/deprecated projection isolation, conflicting
build profiles, missing roots and ambient build-profile substitution. Existing
integrity and release suites retain their separate failure oracles; passing
these bounded cases is not exhaustive system proof.

No source/archive/manifest format or installed receipt migration is introduced.
No provider, model or application live-readiness claim is promoted. Future
issuer ownership remains unresolved and belongs to the deferred readiness
release. Linux/macOS verification of an actual downloaded and provenance-verified
public CLI still follows the approved release cut. Broader #52 application,
provider, executable/composed-package and lifecycle work remains open.
