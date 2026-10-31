# Scoped compatibility evaluation

This is the second implementation step for #84, following #87. The evaluator
implements the compatibility decision table in
[readiness v2](../shared/contracts/operational-readiness-v2.md).
It reports compatibility only; installation, authentication, live availability
and effect authority continue through the existing runtime paths.

## Inputs and authority

`internal/compatibility.Context` receives the selected package/artifact,
dependency-lock and configuration identities, operation, provider target,
runtime/platform and actual API/SDK/capability dimensions. These are independent
expectations from the caller; an observation cannot define its own target or
reduce its own required membership.

Maintainer requirements are conjunctive. Exact versions within a single
requirement are alternatives; listed capabilities are all required. Missing
required dimensions mean unknown. A version outside a declared list means
untested/unknown, not a proven incompatibility. Ranges remain unsupported.

The separate trust store maps SHA256 of the exact raw observation bytes to an
approved producer. Matching a producer name inside the observation is not
sufficient. Formatting or content changes require approval of the new bytes.
The current caller supplies this trust policy; the evaluator does not create
producer approval, verify signatures or contact a provider. A production
producer/verifier integration must establish this authority independently before
these records can be used for release or runtime decisions.

The standalone command is an offline diagnostic: an operator can supply their
own data and trust policy, but its output is not ingested as authoritative
runtime evidence. It always reports `operational_readiness: not_evaluated`.

## Authority ledger

| Protected fact | Authority | Evidence status |
| --- | --- | --- |
| Current subject and requirements | Selected artifact/lock and actual target supplied independently by caller | Confirmed interface; production discovery integration pending |
| Accepted producer observation | Independent approval of exact record bytes and producer | Confirmed evaluation rule; producer verification/distribution pending |
| Required child membership | Root lock and expected child subject/requirement digests | Confirmed invariant; actual lock adapter pending |
| Compatibility semantics | Readiness v2 decision table and scoped changes | Confirmed contract and finite regression cases |
| Authentication/live authority | Existing runtime/provider validation | Separate authority, not established by this diagnostic |

## Observation record

The strict canonical schema is
[compatibility-evidence.schema.json](../shared/schemas/compatibility-evidence.schema.json).
An observation contains schema and policy versions, the complete subject,
current requirements digest, producer, test ID/version, observation time,
result and reason. Results are `passed`, `incompatible` (an applicable
compatibility failure), or `inconclusive` (including transient failures).
There is no artificial expiry or authenticated/ready field.

`RequirementsDigest` canonically binds every required dimension, allowed exact
version and required capability. Requirement ordering, capability ordering and
explanatory notes do not change the binding. Removing or changing a requirement
does change it. `SubjectDigest` binds the complete actual combination; separate
passing API and SDK tests cannot be combined into untested joint coverage.
Provider targets must use non-secret identifiers, never credentials.

The latest approved observation for the exact subject and requirement set wins.
A newer incompatible result replaces an older pass; a newer inconclusive result
reports unknown. At equal instants, incompatible precedes inconclusive, then
passed; a deterministic digest tie-break removes input-order dependence.
Future observations are ignored. Old observations do not expire solely by age.
Removing trust or changing a scoped input prevents the old record from matching.
For a known provider change that exposes no new version, the caller supplies
`invalidated_at` as an independent event barrier. Prior and simultaneous
observations cannot establish compatibility; only strictly newer approved
verification can clear that uncertainty. A declaration alone cannot erase this
barrier. The producer integration must retain such invalidations when trust is
withdrawn so older passing observations cannot silently restore coverage.
These rules retain historical check times and do not renew observations.

The four outputs are `tested-compatible`, `declared-supported`, `unknown` and
`known-incompatible`. Lack of a matching approved observation can establish only
declared support if the actual dimensions match the maintainer requirements.
A matching approved failure can establish known incompatibility. A passing
observation that contradicts the declared constraints cannot promote support.

## Dependency composition

`ComposeDependencyCoverage` requires the exact independently selected locked
member set. Every child report must match the expected subject and requirements
digests. Omitted members, same-count substitutions and swapped child reports
are rejected. The parent caller must derive these expectations from the real
lock and selected target rather than from the submitted child reports.

Any required known incompatibility makes required coverage incompatible.
Required unknown coverage prevents tested coverage. Otherwise a required
untested declaration leaves declared coverage; all required tested observations
establish tested child coverage. Optional failures do not elevate or block
required coverage, but optional reports must still be present and correctly
bound. With no required members the result is unknown. Child coverage alone
does not establish root workflow compatibility or live readiness.

## Try the offline diagnostic

The checked-in request uses synthetic identifiers and placeholder digests:

```bash
go run ./cmd/compatibility-evaluator \
  --request examples/compatibility/request.json
```

This reports declared support with operational readiness not evaluated.
To evaluate separately approved observation files:

```bash
go run ./cmd/compatibility-evaluator \
  --request /path/to/current-target.json \
  --evidence /path/to/observation.json \
  --trust-store /path/to/operator-approved-evidence.json
```

The trust store is a JSON object mapping raw record SHA256 to producer name.
It must come from the operator/verifier authority, not the observation itself.
Inputs are limited to 1 MiB per file and fixed-shape records reject unknown
fields. No sample observation or provider verification has been fabricated.

## Verification and remaining work

The focused tests use independent expected statuses and a 4-by-4 decision table
for required dependency coverage. They exercise nine subject-identity
substitutions, changed/deleted requirements, untrusted bytes/producer names,
missing dimensions, API/SDK combination mixing, rolling capability loss,
chronology, conservative ties, order independence, explicit change-event
invalidation and immutable caller inputs.
The command tests exercise supplied files through JSON output and verify that
compatibility never grants operational readiness.

This is bounded evaluation coverage, not proof that external providers remain
compatible. Real connector version inventory, separately produced test records,
producer verification/trust distribution, current target discovery, live/runtime
integration and CLI/catalogue coverage display remain under #84/#52. Existing
v1 age gates are still active until that coordinated migration is ready.

Validation: the complete Go suite and focused race tests passed. Four temporary
source-overlay mutations were detected (4/4 selected mutations): removing
independent trust, ignoring target scope, adding a blanket 30-day age limit, and
accepting swapped child reports. Required-child result coverage is 16/16 cases
in the declared four-state matrix. These counts apply only to this finite model.
