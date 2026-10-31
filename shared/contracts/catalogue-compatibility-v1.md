# Catalogue compatibility information

| Field | Value |
| --- | --- |
| Contract ID | `catalogue-compatibility` |
| Contract version | `1.0.0` |
| Status | Normative for website content |
| Effective date | 2026-10-31 |

## Scope

The content-only catalogue presents maintainer-provided version information.
This contract governs that content, not operational installation, authentication
or execution. Runtime policy remains in
[operational readiness v1](operational-readiness-v1.md).

## Content and release identity

Maintainers SHOULD document relevant API, SDK, protocol and runtime versions,
important limitations and a source supporting the declaration. Packages MUST
NOT invent a dependency or version merely to populate a field. A skill that
has no relevant platform dependency does not require compatibility metadata.

Editorial annotations MUST identify the module, package ID and exact release
version. They MUST be displayed only for a matching release. A new release
requires its own review; an older annotation MUST NOT carry forward implicitly.

A review date records a maintainer's inspection. It MUST NOT assert a live API
check, independently verified execution or current authentication. Elapsed age
alone MUST NOT expire an editorial compatibility statement or declare a package
irrelevant. Relevant platform or package changes SHOULD trigger a reviewed
content update through the normal repository workflow.

## Display and authority

A matching editorial annotation takes precedence over the release's optional
manifest declarations for this display. Otherwise show existing declared
compatibility, or unknown when neither source provides it. The source and date
of an annotation SHOULD remain visible so readers can assess it.

These notes MUST NOT modify immutable release manifests or archives, installed
runtime contracts, credential authority, scope or revocation. A version claim
MUST NOT imply that a provider is reachable or credentials are valid.

The static build MUST NOT require provider credentials, signed observations,
a compatibility evaluator, a live API service or background polling to publish
these notes. Routine validation SHOULD check the data's package/version matching
and truthful display, alongside existing site build checks.
