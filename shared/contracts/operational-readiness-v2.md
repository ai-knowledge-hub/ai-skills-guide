# Operational Readiness and Compatibility Contract v2

<!-- markdownlint-disable MD013 -->

| Field | Value |
| --- | --- |
| Contract ID | `operational-readiness` |
| Contract version | `2.0.0` |
| Status | Normative target; staged implementation |
| Decision date | 2026-10-30 |
| Tracking | #83, #84, #85, #86 |

## Authority and adoption

This major version changes the validity rules in v1. Producers and consumers
MUST explicitly identify the contract version they implement. Existing v1
artifacts and evidence retain v1 semantics. Publication of this contract does
not upgrade existing runtime enforcement or renew any evidence.

The actors, artifact identity, authority boundaries, delivery classifications,
deprecation, target identity, secret handling and effect authorization rules in
[contract v1](operational-readiness-v1.md) remain normative foundations. This
version supersedes v1 sections 7–9 wherever they require a universal age limit
or an expiry field for compatibility, instruction relevance or package setup
requirements. Other protections remain applicable. A consumer MUST NOT claim v2
conformance by selectively disabling a v1 age check.

## Independent judgments

| Judgment | Required authority | Validity rule |
| --- | --- | --- |
| Artifact integrity | Released bytes, digest, lock and installed receipt | Changes in identity/content invalidate applicable observations |
| Instruction applicability | Referenced product/contract and relevant stack assumptions | Relevant content, API, runtime or workflow changes trigger reassessment |
| Declared support | Maintainer requirements for API, SDK, protocol, runtime/platform | Explicit version/capability matching; a declaration is not a passing test |
| Tested compatibility | Independently produced test observation bound to artifact, operation and dependency/target combination | Known relevant changes or applicable failures invalidate the affected scope |
| Live availability | Provider response for the selected target/operation | Report observation and its time; do not promise ongoing availability |
| Authentication | Current credential/provider validation, principal, scope, target and generation | Actual expiry, revocation or changed identity/binding invalidates authority |
| Security assurance | Scope-specific review and authorized acceptance policy | Its explicit authorized review/acceptance deadline remains binding |

Package age and last-tested age MUST NOT by themselves establish compatibility,
incompatibility or irrelevance. There is no blanket 30-, 90- or 180-day validity
ceiling for compatibility or instruction applicability under v2. Setup
requirements describe an implementation's requirements; current setup checks
are evaluated against the selected environment rather than a package date.

Authentication MUST continue respecting actual credential and provider-attestation
expiry, revocation, scope, principal and generation. V2 removes the generic
30-day manifest timestamp as a substitute for those checks. Where authoritative
validation is unavailable, current authentication is unknown; compatibility
MUST NOT bypass it. Time-limited approvals and explicit assurance deadlines
also remain enforceable. The existing security-review ceiling and five-minute
clock tolerance require separate policy decisions; neither is silently widened
by this compatibility change.

## Compatibility scope and results

A scoped combination includes the released artifact, operation, actual pinned
dependency closure, relevant provider/API identifier and version, SDK package
and version, runtime/platform, protocol and required capabilities. Record only
relevant dimensions. Missing information MUST NOT be invented from a check date.

| Result | Meaning |
| --- | --- |
| Tested compatible | Applicable independently produced test evidence passed for this combination, with no known relevant invalidating change |
| Declared supported | Maintainer explicitly supports this combination; applicable passing evidence has not established tested compatibility |
| Untested / unknown | Target dimensions or applicable evidence cannot be matched |
| Known incompatible | An explicit unsupported combination, breaking change or applicable reproducible compatibility failure establishes incompatibility |

A transient outage or failed credential does not alone establish version
incompatibility. Untested is not a proven failure. A recent timestamp does not
establish compatibility. An old passing observation remains historical evidence
for its recorded combination, and never guarantees a live service is available.

Provider versions may be semantic, date-based or opaque identifiers. The first
implementation uses exact identifiers only. Ranges require a defined comparison
scheme and an explicit support authority before implementation. Semantic-version
ordering MUST NOT be applied to arbitrary provider identifiers. An unversioned
API uses contract/capability probes and known provider changes; lack of a stable
version MUST NOT be interpreted as permanent compatibility.

Each required plugin/agent dependency is evaluated for its actual pinned version
and selected target. A known incompatible required dependency prevents a
compatible composition. Unknown required coverage prevents a tested-compatible
aggregate. Optional components are reported independently. Observations for one
provider, operation or dependency cannot promote unrelated combinations.

## Invalidation and observations

Relevant artifact/dependency changes, API retirement or breaking changes, loss
of required capabilities, runtime contract changes and newer applicable
compatibility failures invalidate or narrow the affected compatibility claim.
Preserve prior observations, versions and check dates for audit. A metadata-only
change may retain an observation only when its scoped inputs remain identical.
Absence of a detected provider change is not proof that no change occurred;
unknown target contracts or capabilities remain unknown.

Compatibility records carry observation time, result, producer, test identity,
artifact/closure identity and the tested combination. They need no artificial
expiry. Time-bound authentication, authorization or assurance records still
carry their authoritative expiry, evaluated as `observed_at <= now < expires_at`.
Future observations MUST NOT extend authority. All comparisons use UTC instants.
Maintainer declarations alone MUST NOT be promoted to independently tested
compatibility. Last-tested dates remain useful context, not a readiness score.

## Migration and implementation gates

1. Schema 2.2 adds optional compatibility declarations using exact version IDs
   and/or required capabilities. Registry projections preserve declarations;
   CLI and web label them as declared support. Legacy absence means unknown.
2. Independent scoped test records and an evaluator must establish tested,
   unknown and incompatible results. Bind them to policy, artifact, operation,
   target and dependency identity; do not infer results from author metadata.
3. Runtime migration must replace coarse manifest-age admission with the new
   compatibility evaluator plus current authentication, target and authority
   checks. Until this replacement is implemented, existing v1 gates stay active.
4. Update release gates, catalog views and #52's evidence/demotion criteria
   together. New producers must not advertise v2 operational readiness through
   v1 records. Older clients may reject schema 2.2 or enforce old age limits;
   document upgrade requirements and rollback behavior.
5. Preserve immutable archives and sidecars. New metadata requires a new release;
   no historical date, digest or evidence record may be rewritten or relabelled
   as a fresh observation. Rollback restores the old consumer/policy without
   altering retained bytes or treating unrecognized evidence as success.

Required acceptance cases include unchanged supported contracts after 30 days,
a breaking change immediately after verification, unknown/missing versions,
exact API/SDK/runtime combinations, rolling API capability loss, mixed required
and optional dependency coverage, and expired/revoked/wrong-target credentials
despite a compatible version. Validate old/new producer-consumer behavior before
claiming runtime v2 conformance.

## Worked decisions

- A connector tested against API `v1` and SDK `3.2.1` does not lose that scoped
  compatibility observation after 120 days solely because time passed. Current
  service availability and authentication still require their own observations.
- A connector checked yesterday is not automatically compatible with API `v2`
  released today. Missing coverage means unknown; an explicit unsupported
  version or applicable breaking failure establishes known incompatibility.
- A tested-compatible API/SDK combination with a revoked credential remains
  compatible for its recorded scope, but authentication fails and operational
  use requiring that credential is blocked.
