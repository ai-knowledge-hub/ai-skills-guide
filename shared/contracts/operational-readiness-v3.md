# Operational readiness: evidence and expiry policy v3

<!-- markdownlint-disable MD013 -->

| Field | Value |
| --- | --- |
| Contract ID | `operational-readiness` |
| Contract version | `3.0.0` |
| Status | Normative for updated runtime consumers |
| Effective date | 2026-11-01 |
| Policy owner | Repository maintainers, through reviewed changes |

## Scope and precedence

This major revision replaces v1's section 9 and its age-based interpretation of
package observations. All other [v1 requirements](operational-readiness-v1.md)
are incorporated, subject to the overrides below. In particular, artifact
integrity, exact target identity, credential generation, provider verification,
revocation, approval and terminal deprecation rules remain in force. The
withdrawn v2 prototype is not a prerequisite or an active policy.

[Catalogue compatibility v1](catalogue-compatibility-v1.md) continues to govern
the content-only website. It requires no service, provider calls, credentials,
signed observations or new compatibility evaluator.

## Decision table

| Claim | Validity rule | Invalidation / outcome |
| --- | --- | --- |
| Package and dependency identity | Exact version, digest and dependency closure | A changed artifact is a different subject; existing evidence does not transfer |
| API, SDK, protocol, runtime or capability support | Relevant maintainer declarations and exact-release reviewed notes, with sources and limitations | Relevant breaking changes, retirement, known failures or a new package release require a content review; age alone does not invalidate the note |
| Legacy `last_verified_at` | Historical observation only, with evidence references | No 30-, 90- or 180-day package admission limit; no authentication or compatibility authority |
| Current authentication | Fresh authoritative provider check with matching target, principal, scopes and credential generation | Earliest actual credential expiry (when present) or mandatory provider-attestation expiry; also revocation, rotation, target or scope changes |
| Setup / doctor | Point-in-time inspection of the current installation and configuration | Rerun for each readiness evaluation; serialized `expires_at` equals `observed_at` and grants no reusable setup lease |
| Persisted local executable smoke result | Exact installed target match plus explicit expiry; maximum 90 days from the run's start, bounded further by authentication deadlines | Failure, changed target or expiry removes local readiness; it does not declare an API version incompatible |
| Security review and accepted risk | Explicit authorized deadline; existing 365-day ceiling retained | Relevant security changes, findings, withdrawal or deadline remove assurance; never grants execution authority |
| Delivery / deprecation | Identity and lifecycle rules | No age expiry; deprecation remains terminal |

The old manifest 180/90/30-day ceilings are removed because elapsed package age
cannot establish instruction relevance, version support or present credentials.
The old 30-day authentication maximum is also removed: the provider supplies a
mandatory attestation deadline and any credential deadline, and consumers check
the provider again rather than treating a persisted success as authority. A
missing attestation deadline fails closed; it MUST NOT mean unlimited validity.

The local smoke 90-day ceiling is retained as a bounded reuse policy for a local
execution result: configuration or environment state outside the recorded
fingerprints may change. It requires a new smoke run, not a new package release
or a rewritten review date. It is not a compatibility or authentication cap.
The security 365-day ceiling likewise remains a separate assurance policy;
this change does not implement or invent security-review evidence. Maintainers
own both retained ceilings; they may be revised independently with evidence.

The five-minute future-observation tolerance is retained solely as package
metadata clock-skew protection. It does not add five minutes to any credential,
attestation, smoke or security deadline. Actual deadlines use UTC half-open
validity: `observed_at <= now < expires_at`, including rejection at exact expiry.
A setup snapshot has an empty reusable interval and must be recomputed.

## Compatibility vocabulary and examples

- **Declared supported**: a maintainer specifies relevant supported versions or
  capabilities; this is not an independently tested result.
- **Reviewed version notes**: an exact-release editorial inspection with sources,
  date and limitations. It is not a live API test.
- **Untested**: the combination has not been exercised by an applicable test.
- **Unknown**: relevant scope or version information is missing or unmatched.
- **Unsupported**: the maintainer explicitly excludes the combination.
- **Known incompatible**: a documented breaking change or applicable reproducible
  failure establishes a mismatch. Transient outages and authentication failures
  alone do not establish version incompatibility.

An unchanged integration with a supported SDK and a review older than 30 days
keeps its version note and remains installable, subject to integrity, lifecycle
and runtime checks. It is not automatically ready for live use.

An integration reviewed yesterday against an API retired today needs a reviewed
correction describing that incompatibility. A recent date cannot override the
breaking change. The static site does not discover vendor changes automatically.

A compatible integration with expired or revoked credentials still fails live
authentication. A 45-day provider deadline is accepted when genuinely supplied
by the provider, but a credential expiring tomorrow shortens it to tomorrow.
Versions, editorial dates and package observations cannot renew either deadline.

## Static publication and authority

A static registry projection MUST NOT promote opaque manifest evidence
references, positive capability declarations or a recent timestamp into live
`usable-now` or target-ready `setup-required` capability claims. Preserve those
original declarations in immutable release history, but project unresolved
positive capability and executable-helper claims as `not-verified`, including
helpers declared by active instruction-primary packages. Instruction delivery
does not establish a helper's executable readiness. Preserve helper paths,
setup instructions and limitations; mark the projected classification inferred
when a helper is demoted. An aggregate `setup-required`
classification may still explain required setup without asserting target success.
Lifecycle demotion still takes precedence. Current local readiness continues to
require actual installation, provider and execution checks; version notes are
not an input to those authority decisions.

## Migration and rollback

1. Updated consumers apply this policy to both legacy and current manifests.
   No manifest schema field, timestamp, digest, archive or release sidecar is
   rewritten. Schema 2.2 compatibility declarations stay optional; missing
   versions remain unknown, never inferred support.
2. New doctor and smoke receipts identify contract `3.0.0`. Their JSON shapes
   remain unchanged; updated evidence schemas accept `1.0.3` and `3.0.0`.
   Composite readers accept either version while still comparing exact current
   child expectations. No new composite producer is introduced.
3. Existing `1.0.3` evidence keeps its original explicit deadline; the new reader
   MUST NOT extend a stored receipt just because the implicit package gate was
   removed. Stored successes still require current target/authentication checks.
   Authentication status itself is recomputed, not loaded as a reusable success.
4. Older clients may continue rejecting packages after the old age caps and
   reject new contract-version receipts. This is a conservative interoperability
   limitation, not a reason to alter retained releases. Upgrade the consumer.
5. Rollback restores the older consumer and its stricter gates. New receipts may
   no longer qualify; rerun checks with that consumer. Never backdate receipts,
   relabel their policy version, disable authentication, or refresh release dates
   to accommodate rollback.

These are consumer policy changes, not claims that old evidence now proves
compatibility or authority. The immutable package is obtainable; live operations
remain independently gated.
