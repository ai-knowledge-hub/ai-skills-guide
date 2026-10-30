# Immutable release store

This directory is the append-only source for published package releases. Each
release is stored below:

`releases/<module>/<category>/<package>/<version>/`

The directory contains the exact versioned manifest, `package.tar.gz` bytes,
and immutable `registry-entry.json` projection. The projection lets the
publisher reconstruct a catalog entry if its mutable source directory is later
removed. `apps/web/scripts/prepare-public-assets.mjs` restores these retained
files into the web build and rejects an existing `(module, id, version)` whose
manifest, archive, or projection differs from the deterministic candidate.

Before indexing any retained release, the publisher runs it through the Go
installer's bounded extraction, path hardening, canonical module schema,
semantic admission, and package-layout checks. Historical evidence timestamps
are preserved as release records and do not become publication failures merely
because time passes. Current evidence freshness is still enforced when a client
selects and installs that version. The published catalog also derives a current
projection: an expired historical `usable-now` claim is shown as
`not-verified`, while its immutable evidence remains in the release record.
Retained versions use SemVer precedence, with deterministic ordering for build
metadata that has equal precedence. Go is therefore required alongside Node
when generating the web assets.

To publish changed package content, increment the manifest version, regenerate
the registry indexes, run `pnpm prepare:assets` in `apps/web`, and commit the
new release directory. Do not edit or remove an existing version. CI runs
`make release-assets-check` so new releases cannot exist only in an ephemeral
build workspace, and compares the complete pull-request or push commit range so
an earlier commit cannot conceal a mutation behind an unrelated final commit.
