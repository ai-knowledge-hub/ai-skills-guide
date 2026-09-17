# Remote package installation

`skills-hub install` has two intentionally separate source modes.

- `--source local` is the default development path. It reads registry metadata
  and package files from a repository checkout.
- `--source remote` installs a published release through an HTTPS registry. It
  does not read package source from the current checkout.

Remote mode is useful for clean clients, automation, and reproducible installs
that should consume a released artifact rather than mutable source files.

## Install a published release

Each module uses its own registry index:

```bash
./bin/skills-hub install \
  --source remote \
  --module skills \
  --registry-url \
    https://skills.ai-knowledge-hub.org/registry/skills-index.json \
  --entry engineering/implementation-strategy@latest \
  --runtime codex
```

Use `--module agents`, `tools`, or `plugins` with the corresponding published
index. Template-only entries are rejected from the operational install path.
Plugin releases must also declare a self-contained dependency closure.
For a self-contained plugin, the publisher copies every declared skill, agent,
and tool into `bundled/<module>/<id>/` inside the archive and verifies each
embedded manifest and package layout. Declared hooks must remain under the
plugin's own `hooks/` directory. Missing or undeclared bundled components fail
admission before runtime installation.

Local-only flags such as `--root` and `--registry` are rejected in remote mode.
Remote-only flags are rejected in local mode. This prevents an install from
silently mixing trusted release metadata with mutable checkout content.

## Execution compatibility

Before committing an executable or bundle, remote admission maps the local host
to the manifest platform vocabulary: Go `darwin` is `macos`, while `linux` and
`windows` retain their names. The host must appear in
`execution.supported_platforms` for the package and every bundled dependency.

The selected catalog adapter (`codex`, `claude`, or `generic`) and `native` are
the default execution capabilities. Additional environments must be selected
explicitly with a repeatable flag, for example:

```bash
./bin/skills-hub install \
  --source remote \
  --registry-url https://skills.example.test/registry/skills-index.json \
  --entry engineering/example@1.0.0 \
  --runtime generic \
  --target ./runtime/skills \
  --execution-runtime node22 \
  --execution-runtime container
```

At least one selected capability must appear in each executable manifest's
`execution.supported_runtimes`. The flag declares the environment selected by
the operator; it does not install or independently attest that runtime.

## Verification and commit sequence

Remote installation completes these stages in order:

1. Fetch and parse a supported registry index over HTTPS.
2. Resolve the requested package ID and version from a registry `1.3` release
   record carrying both artifact and manifest digests.
3. Download the archive from the registry origin. Additional HTTPS origins
   require an explicit, repeatable `--allow-origin` value.
4. Stream the archive through SHA-256 verification before caching it.
5. Extract into a temporary directory and reject unsafe paths, links, special
   files, duplicate paths, oversized content, and excessive entry counts.
6. Verify the archive manifest against the selected version's
   `manifest_sha256`, apply the embedded canonical module schema, then validate
   its admission metadata and on-disk layout, including the selected ID,
   version, runtime, and current evidence freshness. Version-specific package
   metadata comes from that manifest, while entry-level deprecation and
   replacement policy remains authoritative from the current registry. A
   deprecated entry is warned consistently with local installation and cannot
   retain `usable-now` or `setup-required` readiness at either the root or an
   executable-helper scope.
7. For plugins, validate every bundled dependency for the selected runtime and
   operational install path, then prepare runtime-specific package files in
   staging.
8. Add a non-secret install receipt binding the package identity, artifact
   digest, runtime contract, and normalized installed-tree digest.
9. Rename the complete staged tree into its destination. Plugin dependencies
   are activated in their sibling runtime module directories before the plugin
   becomes visible. The full closure shares a durable transaction, so a failed
   or interrupted activation restores every previous dependency before
   restoring the plugin. Because external runtimes do not participate in
   installer locks, in-place upgrades of installed multi-package closures are
   rejected until a runtime-wide atomic activation pointer is available.

No downloaded package byte reaches a final runtime directory before the plugin
and its complete dependency closure pass verification and validation.

`skills-hub doctor` and `skills-hub smoke` require this receipt and recompute
the installed-tree digest. Packages installed by an older CLI must be
reinstalled before those commands can produce current readiness evidence.

## Cache and offline operation

The default cache lives below the operating system user cache directory. Set a
different location with `--cache-dir`.

- Registry snapshots are keyed by registry URL and the package/version
  selection whose trust pipeline they passed.
- A downloaded registry becomes the newest cached snapshot only after its
  selected artifact and manifest pass the complete trust pipeline. The five
  latest successful snapshots are retained for last-known-good recovery.
- Artifacts are content-addressed by their expected SHA-256.
- A cached artifact is hashed again before every use.
- A corrupt cache entry is never installed and can be repaired by an online
  retry.
- `--offline` performs no network request. It succeeds only when both the
  registry snapshot and selected artifact are already cached and valid.

An online fetch is required once before an offline install:

```bash
./bin/skills-hub install \
  --source remote \
  --registry-url \
    https://skills.ai-knowledge-hub.org/registry/skills-index.json \
  --entry engineering/implementation-strategy@1.0.0 \
  --runtime generic \
  --target ./runtime/skills

./bin/skills-hub install \
  --source remote \
  --offline \
  --force \
  --registry-url \
    https://skills.ai-knowledge-hub.org/registry/skills-index.json \
  --entry engineering/implementation-strategy@1.0.0 \
  --runtime generic \
  --target ./runtime/skills
```

## Failure and recovery

Errors identify the failed stage, such as registry download, version
resolution, checksum validation, archive validation, package validation, or
install commit. Network, checksum, and validation failures leave the existing
runtime package unchanged. Interrupted downloads remain temporary and are
discarded.

If an install reports a cache error, remove only the named cache entry or use a
new `--cache-dir`, then retry online. If activation fails during `--force`, the
installer restores the prior package. A message naming a rollback directory
means automatic restoration also failed; move that exact directory back to the
reported destination before retrying.

Interrupted force upgrades are reconciled automatically on the next install of
the same package. If activation had not completed, the preserved package is
restored; if activation had completed, the stale backup and marker are removed.
Concurrent processes targeting the same package are serialized by a durable
package-scoped lock.

Plugin-closure recovery acquires every package lock recorded in its durable
transaction marker. If any lock is busy, installation stops with a retriable
error and leaves the marker and runtime trees untouched.

## Publisher checksum contract

The `sha256` in a remotely served registry index is the digest of the exact
`.tar.gz` response at that version's `artifact_url`; `manifest_sha256` binds the
selected release's admission metadata. The web publication step creates a
deterministic candidate, then compares it with the tracked, append-only
`releases/` store. Reusing an `(id, version)` with different bytes fails. Clean
builds restore every retained version, so older pinned releases remain
installable. Repository registry files are current source projections and must
not be served directly as release indexes.
