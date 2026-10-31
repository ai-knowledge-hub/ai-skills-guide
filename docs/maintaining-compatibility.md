# Maintaining catalogue compatibility information

The website is a static catalogue. Maintainers record relevant version details
in `apps/web/lib/catalogueCompatibility.ts` through a normal reviewed change.
The static build displays an annotation only when its module, package ID and
exact release version match the catalogue entry's latest release.

For each annotation, record only details useful to someone choosing or setting
up the package. Include the reviewed date, a source link to the exact packaged
release or authoritative documentation, and important limitations. A date
records an editorial review; it is not an automatic expiry or a live check.

When a package publishes a different version, the old annotation does not carry
forward automatically. Review its implementation and dependencies before adding
an annotation for the new version. Missing annotations remain unknown. Skills
without relevant API or SDK dependencies do not need invented versions.

This data is website content. It does not modify immutable package manifests,
installation behavior, credential checks, or runtime readiness. The existing
manifest-declared compatibility is still displayed when no editorial annotation
exists. No provider API calls, signing keys, background checks, or new servers
are needed to display catalogue information.

The offline evaluator from #88 remains in the repository for now. It is not a
prerequisite for maintaining or publishing these annotations. A separate audit
should identify its consumers before any removal. The operational readiness
policy and runtime gates are tracked separately in #83.
