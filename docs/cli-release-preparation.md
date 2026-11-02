# CLI release preparation

<!-- markdownlint-disable MD013 -->

Prepared on 2 November 2026. This is a reviewable publication plan, not a record
that a new tag, release, deployment or live-runtime proof has been published.
The content-only website remains independent of the adopting client's installer.

## Proposed milestone

Use a new alpha release for the accumulated CLI and catalogue changes. With the
currently published latest tag `v0.3.0-alpha.1`, the existing `minor-alpha` cut
computes **`v0.4.0-alpha.1`**. Confirm the dry-run output immediately before the
cut; this document does not reserve a tag or change package versions.

The older GitHub releases have no attached CLI binaries. Do not retrofit them
with binaries built from today's source: a release binary must match its tag.
Prepare a new tag after the reviewed work is integrated into `main`.

## Merge and configuration prerequisites

1. Merge [PR #97](https://github.com/ai-knowledge-hub/ai-skills-guide/pull/97) and
   this release-preparation change into `dev`; require green checks.
2. Review and merge the intended `dev` state into `main`, confirm its CI and
   intended production deployment, and record the exact commit.
3. A release maintainer must confirm these repository Actions variables contain
   the approved runtime-adapter **public** Ed25519 keys, in canonical standard
   base64, each decoding to 32 bytes:
   - `CODEX_MODEL_ATTESTATION_PUBLIC_KEY`
   - `CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY`
   - `GENERIC_MODEL_ATTESTATION_PUBLIC_KEY`
4. Confirm the corresponding signing adapters and key ownership. Key-shaped data
   alone does not establish an issuer. Keep private signing keys in the adapter
   deployment, outside this repository. Do not substitute the CI test-vector key
   or generate a new issuer just to make the release build pass.

The existing `make check-runtime-trust` validates representation; `make cli-build`
and the release builder require all three roots. The issuer contract is in
[runtime model readiness attestation](model-readiness-attestation.md). These
runtime model roots are separate from GitHub's binary build-provenance signature.
They do not require the static website to host an issuer or call a provider.

Repository-variable configuration could not be confirmed in this preparation:
the local `gh` account returned HTTP 403 with “Your account was suspended” when
listing variable names. No variable values or secrets were retrieved. GitHub
connector reads still worked. Treat the configuration as **unconfirmed**, rather
than missing or approved; use an authorized maintainer account for this check.

## Cut and observe

Run **Cut Release** from `main`, first with `release_type: minor-alpha` and
`dry_run: true`. It validates the catalogue, manifests, governed CLI build and
web build without pushing a tag or dispatching publication. The dry run does not
exercise GitHub OIDC attestation or all six cross-compiles.

After reviewing that run, repeat with `dry_run: false` on the intended source.
The workflow checks out `main` again, so confirm it has not advanced to an
unreviewed commit between runs. It pushes the computed tag and explicitly calls
`Release On Tag` **using that tag as the workflow ref**. Do not dispatch at `main`
and pass a tag merely as an input: the attestation must identify the tag ref.

This explicit call is necessary because a `GITHUB_TOKEN`-authenticated push does
not trigger a second push workflow. [GitHub's workflow-trigger documentation](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow)
describes the exception for explicit dispatches. `Cut Release` now has the
`actions: write` permission needed to request that dispatch.

Wait for **Release On Tag** itself to succeed; an accepted dispatch is not
publication success. It rejects a branch ref, malformed/missing tag, mismatched
source or checkout, unmerged tagged source, or dirty checkout. It runs the Go
suite, builds governed binaries, attests their digests, and uploads the assets.
Per-tag concurrency serializes publication runs.

For `v0.4.0-alpha.1`, expect these eight release assets:

| Asset | Intended target |
| --- | --- |
| `skills-hub_v0.4.0-alpha.1_linux_amd64` | Linux x86-64 |
| `skills-hub_v0.4.0-alpha.1_linux_arm64` | Linux ARM64 |
| `skills-hub_v0.4.0-alpha.1_darwin_amd64` | macOS Intel |
| `skills-hub_v0.4.0-alpha.1_darwin_arm64` | macOS Apple Silicon |
| `skills-hub_v0.4.0-alpha.1_windows_amd64.exe` | Windows x86-64 |
| `skills-hub_v0.4.0-alpha.1_windows_arm64.exe` | Windows ARM64 |
| `SHA256SUMS` | Binary checksums |
| `skills-hub_v0.4.0-alpha.1_provenance.json` | Signed binary provenance bundle |

The bundle attests the binaries listed in `SHA256SUMS`; the checksum file itself
is not a separate signed subject. Binary checksum and signed-subject verification
are both required. Cross-compiling a target is not evidence of running on it.

## Published-client acceptance

After publication, dispatch **Published registry installation** from the
integrated workflow with `client_source: release`, the exact new `cli_release`
tag, and the intended public HTTPS registry URL. This workflow is supplied by
PR #97 and must be on the default branch before dispatch.

Require **both** Linux and macOS reports to identify `client_kind: release`,
pass repository/workflow/tag-bound provenance and checksums, and pass all three
installation transitions for each of Codex, Claude and generic adapter targets.
Retain the workflow links, source commit, binary digests and JSON reports.

That is 18 installation transitions across two hosts. The reports must also
identify the actual host architecture; this does not prove every cross-compiled
architecture. Missing assets or failed provenance must stop verification, with
no source-built fallback. Follow PR #97's documented commands and limitations.

Do not close #52 on this result alone. Application startup, executable/composed
packages, upgrade/rollback/uninstall, provider authentication and bounded live
sandbox operations remain separate work. Adapter directory installation does
not establish live readiness.

## Recovery

If the tag push succeeds but dispatch fails, keep the tag and inspect the run.
For a prepared tag containing these workflow changes, use an authorized account:

```sh
gh workflow run release-on-tag.yml \
  --repo ai-knowledge-hub/ai-skills-guide --ref v0.4.0-alpha.1
```

The tag must still point to the reviewed commit integrated into `main`. Do not
move a public tag, use a branch ref, or rebuild an old tag from newer source.
Before retrying publication, inspect existing assets: the upload action can
overwrite same-name assets. If a public release is partial or incorrect, stop
and choose an explicit recovery or a new version; do not silently replace
already distributed binaries. Correct code/workflow changes require a new tag.

If published-install verification fails after publication, retain its report,
keep #52 open, and investigate the failed stage before making installability
claims. A website deployment or candidate-client pass does not replace this
released-client acceptance.

## Draft release notes

Suggested title: **v0.4.0-alpha.1 — Governed CLI distribution and practical catalogue**.

- Publish platform CLI binaries with checksums and tag-bound build provenance.
- Verify published instruction-package delivery with an isolated client and
  explicit release-versus-candidate evidence.
- Keep catalogue compatibility as maintainer-reviewed exact-release API/SDK
  information; elapsed review age alone does not establish incompatibility.
- Present the catalogue as the publication's practical companion, with the
  narrower folded-field artwork and curated article/resource links.
- Preserve runtime checks for actual authority, provider/credential deadlines,
  revocation and target scope on the adopting client.

Before publishing these notes, replace proposed outcomes with the actual source
commit, asset list and verification results. Describe adapter-target proof
precisely; do not claim actual Codex/Claude launches, provider success, Windows
execution, or complete #52 coverage without separate evidence.
