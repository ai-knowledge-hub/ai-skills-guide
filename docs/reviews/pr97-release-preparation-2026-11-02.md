# PR #97 review and CLI release preparation

<!-- markdownlint-disable MD013 -->

## Review frame

Reviewed PR #97 at `99e07fbb2507844841692eac0e9b477287a8b86d`, against its
merge-base `96cc5f9cb306a147acc7e2ae328de196bed138e6`. Current `dev` is
`2216296247911bd17e3691f11325d1f4ba8d3f16`, which additionally merges the separate
landing-page PR #98. PR #97 changes ten files; retained JSON reports are evidence,
not executable policy. No new actionable defect was found in this bounded review.
The PR was marked ready for review, without merging it.

## Reachable surface and authority

| Producer / authority | Commit or verification boundary | Consumer / recovery |
| --- | --- | --- |
| Public registry selects the pinned `0.1.0` release record | Downloaded manifest and archive must match recorded SHA-256, and archived/served manifests must agree | Historical manifest parser determines instruction-only usability and declared adapter targets; no implicit latest substitution |
| GitHub release assets and signed provenance | Verify checksum and repository/workflow/tag-bound binary attestation before chmod/execution | Release mode fails closed; candidate mode remains separately labelled |
| CLI runs outside the source checkout with explicit disposable target/cache | Files must match the separately downloaded archive; receipt must bind identity, runtime, source and artifact | Go receipt consumer checks tree/contract digests and strict metadata; temporary installation is removed afterward |
| Duplicate installation attempt | Nonzero exit plus unchanged lstat inventory/bytes/metadata before any repair | Mutation fails before the repair step can erase its evidence |
| Offline forced reinstall after deliberate damage | Cached install must restore the published payload and valid receipt | Bounded proxy rejection supplements installer transport tests; it is not an OS network sandbox |

Confirmed by source and focused execution: pinned-manifest selection, independent
downloaded-byte oracle, receipt verification and duplicate-before-repair ordering.
The Go contract/receipt routines are intentionally the actual consumer; they are
not an independent proof of their own digest algorithms. Python fixture tests
also supply independently represented tree metadata and a retained contract
digest. Existing installer tests cover the underlying parser/transport paths.

Counter-review examined newer current-entry declarations with unchanged pinned
history, coordinated receipt/payload substitutions, success-only clients,
same-byte filesystem mutations, malformed URLs/archives, and release provenance
failure before execution. The 16 focused tests invoke the actual Go consumer
where relevant; they passed again on 2 November. The installer suite also passed.
No previous inline review threads were returned by GitHub. All eight check runs
and the Vercel status were green for the reviewed head.

## Separate publication gap and preparation

Direct inspection of the existing `Cut Release` producer showed that it pushes
with `GITHUB_TOKEN` but expected a separate push-triggered publisher to run.
GitHub suppresses that chained push event. This is a release-path gap in existing
code, rather than an introduced #97 checker defect.

The separate preparation branch explicitly dispatches publication at the new tag
and preserves `refs/tags/<version>` for the #97 provenance policy. It adds exact
tag/source/main-ancestry/clean-checkout guards, serializes same-tag runs, and adds
seven deterministic release-boundary tests using isolated Git repositories and a
non-networking dispatch stub. Tests cover lightweight/annotated tags, malformed
and branch-only refs, unmerged source, source/checkout substitutions, absent main
authority, dirty trees, exact dispatch arguments and failed-dispatch propagation.

## Validation and remaining evidence

- PR #97: all 16 focused checker tests and the installer Go suite passed locally.
- Release preparation: all seven tests passed under Python 3.12 and 3.14; shell
  syntax and actionlint v1.7.7 passed for all three changed workflows
  (shellcheck/pyflakes integrations disabled).
- `go test ./...` passed on the preparation baseline with unchanged runtime code.
- No runtime implementation, registry, release-history, package manifest or
  website changes are needed for this preparation.
- Repository trust-root variable configuration remains unconfirmed: the local
  GitHub CLI account returned HTTP 403 / suspended-account text when asked for
  variable names only. No keys or secrets were read or created.
- Actual tag push/dispatch/OIDC publication, six governed cross-compiles, and
  Linux/macOS released-client proof have not been run. No new release was cut.
- Public release reruns can overwrite assets under the existing uploader; the
  preparation checklist requires inspecting partial publication and explicit
  recovery rather than silently replacing distributed binaries.

The [release checklist and draft notes](../cli-release-preparation.md) describe
the remaining merge, configuration, publication and acceptance sequence.
