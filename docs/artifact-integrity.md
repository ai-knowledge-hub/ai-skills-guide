# Artifact locks, SBOMs, and provenance

Self-contained plugin releases carry the complete local dependency closure
needed by the installer. The release publisher resolves direct and transitive
skill, agent, and tool references from the generated registry and embeds each
package under `bundled/<module>/<id>/`.

Every self-contained release declares and contains:

- `dependencies.lock.json`, which pins the root identity and every reachable
  component's module, ID, version, manifest path, dependency edges, and
  canonical SHA-256 content digest;
- `checksums.txt`, which covers every regular archive member except the
  checksum file itself;
- `sbom.cdx.json`, a CycloneDX 1.5 component inventory whose component hashes
  must match the dependency lock;
- `provenance.json`, which records the deterministic publisher, closure digest,
  and locked materials used to construct the release. It is checksum-bound but
  unsigned, so it is not a cryptographic release signature.

The release validator does not trust these files merely because they are
present. It verifies the checksum inventory against the extracted archive,
recomputes every component tree digest, matches component identities and
versions to their manifests, rejects missing, unlocked, or unreachable archive
members, and cross-checks the SBOM and provenance against the lock. The outer
registry additionally binds the complete archive and root manifest with SHA-256
digests.

## Reproducibility

Archive paths, modes, ownership, timestamps, member order, metadata JSON, and
checksum order are canonical. Transient local files such as Python bytecode,
tool caches, `node_modules`, and operating-system metadata are excluded. An
existing release is compared by its canonical uncompressed tar payload so a
different zlib implementation cannot create a false collision.

## External binaries and services

The publisher bundles repository-owned skills, agents, tools, hooks, config,
and templates. It does not download or vendor large binaries, APIs, hosted MCP
servers, credentials, or provider assets during a build. Such dependencies
remain external authority boundaries and must be resolved by a future contract
that supplies an immutable HTTPS coordinate, SHA-256 digest, platform, license,
and provenance. Until that contract exists, an external binary cannot be
claimed as part of a self-contained release.

## Release lifecycle

Changing any locked component or generated integrity record requires a new
root plugin version. Published release directories are append-only; the
publisher rejects deletion or replacement of an existing version.
