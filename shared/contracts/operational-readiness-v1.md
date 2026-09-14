# Operational Readiness and Promotion Contract

<!-- markdownlint-disable MD013 -->

| Field | Value |
| --- | --- |
| Contract ID | `operational-readiness` |
| Contract version | `1.0.3` |
| Status | Normative |
| Effective date | 2026-09-13 |

## 1. Purpose and scope

This contract defines how the catalog describes delivery, operational
readiness, security assurance, lifecycle, and usability. It is authoritative
for skills, agents, plugins, tools and MCP packages, and documentation packs.
Registry builders, schemas, validators, installers, and user interfaces MUST
derive their behavior from this contract and MUST NOT infer stronger readiness
from a module name, folder, prose claim, or the presence of a manifest.

The key words MUST, MUST NOT, REQUIRED, SHOULD, SHOULD NOT, and MAY are to be
interpreted as described by RFC 2119 and RFC 8174 when they appear in capitals.

This contract defines claims and evidence, not the storage schema that carries
them. A future schema MAY normalize the representation, but it MUST preserve
the meanings and derivation rules below.

## 2. State model

The named states are not one linear maturity ladder. They belong to four
independent axes so that, for example, a security review cannot imply a working
installation and an instruction-only skill need not pretend to be executable.

| Axis | Exactly one or independent | States |
| --- | --- | --- |
| Delivery | Exactly one | `documentation-only`, `template-only`, `implemented` |
| Operational readiness | At most one per usability scope and target key | `setup-required`, `verified-usable` |
| Security assurance | Independent boolean state | `security-reviewed` |
| Lifecycle | Independent terminal state | `deprecated` |

Every published entry MUST declare or deterministically derive one delivery
state. An entry without sufficient readiness evidence has no operational
readiness state; consumers MUST render it as `not-verified`, not silently as
`setup-required`. `deprecated` overrides every positive discovery or usability
claim, though prior states and evidence MAY be retained for audit history.
Readiness is evaluated separately for `instruction` and `executable` scope and
for each target key. A target key contains the entry version and artifact
digest, outcome scope, runtime and version, platform, configuration class,
capability or operation set, and a stable environment identifier for a concrete
environment evaluation.

When an outcome contacts an external provider, the target key MUST also contain
a provider-target fingerprint built from the provider/service identifier,
sandbox or live tier, canonical endpoint and region, and every provider-native
tenant, organization, account, project, subscription, or resource identifier
that bounds the claimed operation. It MUST include the provider API/protocol
version, authentication method, verified principal fingerprint, and granted
scope set. The authoritative provider MUST attest the native identifiers,
principal, and scopes through a non-destructive identity operation. The
fingerprint MUST be deterministic and MUST NOT contain credentials or secrets.
User-entered display names, aliases, URLs, or configuration text are not
authoritative provider identity.

The target key MUST additionally include a non-secret credential-binding key
for every authentication requirement. It consists of an opaque binding ID plus
an opaque generation or revision, or an equivalently safe credential-generation
fingerprint supplied by the credential manager or authoritative provider. A
binding ID without a generation is insufficient. If the credential system does
not expose a safe generation identifier, the runtime MUST maintain a monotonic
binding revision and increment it atomically before any credential replacement,
refresh, renewal, revocation, or source rebinding becomes usable. The key MUST
NOT be derived by hashing, truncating, encoding, logging, or otherwise
transforming secret material.

Authentication verification MUST record the credential-binding key observed
before the non-destructive provider check and confirm that the same key remains
current after the check. A changed or unreadable binding key makes the result
ineligible. Consumers MUST compare the current binding key with the evidence at
readiness derivation and immediately before live use. A mismatch invalidates
authentication evidence even when provider target, principal, and scopes are
unchanged.

The credential resolver used for an invocation MUST conditionally resolve the
exact expected credential-binding key. If the binding changes between the
readiness check and resolution, the invocation MUST stop and reauthenticate;
it MUST NOT fall back to the latest credential under the same binding ID.

Provider-target fields MAY be `not-applicable` only when the verified outcome
makes no provider call. Other fields that do not apply MUST also be represented
explicitly as `not-applicable`; they MUST NOT be omitted and then treated as
wildcards. Evidence from a sandbox, tenant, account, project, region, endpoint,
principal, or scope set MUST NOT verify another, even when all package and
runtime fields match.

An entry MAY therefore be instruction-verified while its executable scope is
`setup-required`, or be verified on one runtime and not verified on another. It
cannot hold both readiness states for the same scope and target key. A consumer
MUST match the complete target key before using readiness evidence. A catalog
view without a selected target MUST list the verified target coverage or show a
conservative aggregate; it MUST NOT promote untested targets from evidence for
another target.

### 2.1 Delivery states

#### `documentation-only`

The entry's intended product is guidance, a playbook, or other human- or
agent-readable instruction. It has no claimed runnable entrypoint. Examples and
incidental helper snippets do not make the entry executable. Attempting to
install or launch it as executable software MUST be rejected.

#### `template-only`

The entry supplies a scaffold, composition, connector definition, sample
configuration, or partial implementation that requires a user or developer to
add implementation, resolve dependencies, or replace placeholders before its
claimed executable outcome can run. Copying its files MAY be supported, but
that copy MUST NOT be described as an operational installation.

#### `implemented`

The released artifact contains every claimed entrypoint and its complete,
pinned or integrity-bound dependency closure; runtime registration can be
generated and validated; no implementation placeholders block the claimed
operation; and the package defines a first meaningful operation. This state
asserts implementation completeness only. It does not assert installation,
configuration, authentication, verification, or live readiness in any target
environment.

### 2.2 Operational readiness states

#### `setup-required`

The entry is implemented, but its executable outcome is not currently usable
in the evaluated target because at least one declared, user-actionable
configuration or authentication precondition is unsatisfied. The package MUST
identify each missing precondition and provide a guided setup path. Missing
code, unresolved dependencies, absent entrypoints, unsupported runtimes, or
expired verification are defects or `not-verified`; they MUST NOT be laundered
as `setup-required`.

#### `verified-usable`

Current, scoped evidence proves the entry's declared instruction or executable
outcome on the identified artifact and target. The evidence MUST satisfy
section 7 and be unexpired under section 9. This is an evidence claim, not an
author's assessment.

For instruction outcomes, verification proves that a clean client can obtain
the complete instructions and use them to produce the declared result against
documented acceptance criteria. For executable outcomes, verification proves
clean-client installation and the first meaningful operation. The evidence
scope MUST be recorded as `instruction` or `executable`.

### 2.3 Assurance and lifecycle states

#### `security-reviewed`

A named reviewer or approved automated review process has evaluated the exact
artifact digest and its dependency closure against the applicable security
checklist, recorded findings, and produced an accepted disposition that has not
exceeded the freshness rules in section 9. An accepted disposition is exactly
one of:

- `approved`: no open finding exceeds the checklist's acceptance threshold; or
- `approved-with-accepted-risk`: every otherwise-blocking finding has a scoped,
  unexpired acceptance from the policy-authorized risk owner, including the
  rationale and compensating controls.

`rejected`, `remediation-required`, `incomplete`, an absent disposition, or an
acceptance by the artifact author without the required policy authority MUST
NOT produce `security-reviewed`. Security review does not prove that an entry
is implemented, configured, authenticated, verified usable, authorized for a
particular user, or safe for every deployment.

#### `deprecated`

The owning project has withdrawn an entry version or declared version range
from recommended use. The state requires an effective time, reason, and
replacement identifier as required by the current manifest contract. Installers
and user interfaces MUST warn or block according to policy and MUST NOT show
`usable-now`, even if retained evidence remains unexpired.

`deprecated` is terminal for the affected version or version range. No actor,
including the entry owner, may transition that same version or range back to an
active lifecycle state. Clearing the flag, superseding the decision record, or
adding fresh verification MUST be rejected. Reintroduction requires a distinct
artifact version outside an explicitly bounded deprecated range, or a distinct
replacement entry when the deprecation applies to the entry as a whole. The new
artifact requires its own delivery classification and current readiness
evidence; it does not restore or rewrite the deprecated lifecycle history.
Because the current manifest flag is entry-level, `deprecated: true` applies to
the entry as a whole unless an authoritative version range is explicitly
recorded; an omitted range MUST NOT be inferred as only the latest version.

## 3. Instruction usability and executable usability

Instruction and executable usability are separate claims.

| Claim | What it permits | Minimum evidence |
| --- | --- | --- |
| Instruction usability | Read or apply guidance; generate the declared informational artifact | Complete clean-client retrieval; all referenced local resources present; representative instruction run evaluated against declared acceptance criteria |
| Executable usability | Invoke a packaged entrypoint to perform the declared operation | Published-artifact clean-client install; dependency and entrypoint checks; runtime registration; successful first meaningful operation; protocol and auth checks when applicable |

An instruction-usability result MUST NOT authorize a tool call, process launch,
network access, or external effect. An executable package MAY also carry
instruction evidence, but the two evidence records remain independently scoped
and independently expiring.

When an instruction-primary skill packages executable helpers, the instruction
classification MUST remain separate from an `executable_helpers` disclosure for
each runnable entrypoint. Each helper carries its own execution shape,
availability, limitations, and executable evidence scope; the instruction
classification MUST NOT deny the helper's existence or promote it implicitly.

Prompt tests assess instruction behavior. They MAY contribute to instruction
verification when their environment, artifact identity, expected outcomes, and
results are recorded. Prompt tests alone MUST NOT prove executable usability or
operational availability.

Schema validity proves only that serialized metadata has the expected shape.
Manifest or registry schema validity MUST NOT prove installation, dependency
closure, configuration, authentication, executable usability, live readiness,
or operational availability.

## 4. Environment terms

These terms describe a package in a specific target environment. They are not
global properties of a catalog entry.

### `installed`

The exact released artifact identified by version and digest has been obtained
from its declared public distribution source without a repository checkout;
its integrity has been checked; its complete dependency closure is present;
all declared entrypoints exist; and runtime-specific registration is generated
and accepted by the target runtime. Merely copying files is not sufficient.

### `configured`

All non-credential settings and credential references required for the claimed
operation are supplied, parse successfully, resolve to the intended target,
and pass a non-destructive configuration check. Configuration does not imply
credential validity or authorization.

### `authenticated`

For every required integration, the target environment holds or can securely
obtain a non-expired credential using the declared method, and the authoritative
provider has verified that credential's identity and required scopes using a
non-destructive check. The current non-secret credential-binding key MUST match
the key recorded by that check. The evidence MUST NOT expose the secret.
Presence of an environment variable, secret name, token-shaped string,
unchanged principal/scope metadata, or completed setup form does not prove
authentication.

An entry whose declared auth method is `none` satisfies the authentication
predicate without credential evidence, provided verification confirms that the
first meaningful operation does not encounter an undeclared auth requirement.

### `verified`

An evidence record satisfying section 7 exists and matches the complete target
key defined in section 2, including the exact contract version and artifact
digest. The observation time is before `expires_at`, and no invalidation event
in section 9 has occurred.

### `live-ready`

The evaluated environment is `installed`, `configured`, `authenticated` when
required, and `verified` for executable use. Applicable security review,
policy, approval, and authorization gates are current, and the intended live
capabilities are within the verified scope. `live-ready` is computed at use
time for a specific environment and operation; it MUST NOT be stored or shown
as a timeless catalog property. Live credentials or authority MUST NOT be
bundled in a released artifact.

## 5. Module contract

`Instruction` and `Executable` below are usability scopes. A module may expose
both only when it contains both a complete instruction outcome and a runnable
outcome with separate evidence.

| Module type | Allowed delivery states | Allowed usability scope | Execution rule |
| --- | --- | --- | --- |
| Documentation pack | `documentation-only` | Instruction | Never executable or installable |
| Skill | `documentation-only`, `template-only`, `implemented` | Instruction; executable only for packaged scripts/entrypoints | Instructions may guide reasoning; execution requires an implemented entrypoint and executable evidence |
| Agent | `template-only`, `implemented` | Instruction for template instantiation; executable for a packaged orchestrator | A declarative agent definition alone is not an executable agent |
| Plugin | `template-only`, `implemented` | Instruction for composition/setup; executable for an installable bundle | Referenced dependencies MUST be contained or securely resolved; advisory Markdown hooks are not enforced controls |
| Tool or MCP package | `documentation-only`, `template-only`, `implemented` | Instruction for documentation/template use; executable for a packaged connector/server | A tool definition, sample payload, or `TOOL.md` alone is not a connector; an MCP server claim requires a launchable transport and protocol evidence |

All module types MAY be `security-reviewed` or `deprecated`. Only an
`implemented` entry may be `setup-required` for executable use. Any
non-deprecated delivery state may be `verified-usable` for instruction use if
instruction evidence exists. Only `implemented` may be `verified-usable` for
executable use.

### 5.1 Registry compatibility projection

The existing `usability.availability` field is a user-facing compatibility
projection, not a fifth source-of-truth axis. Conforming producers map it as
follows for a stated scope and target key:

`not-verified` is introduced by manifest schema 1.1/2.1 and registry format 1.2.
Schema 2.0 and registry 1.1 consumers do not understand that value and MUST use
a compatible prior snapshot or upgrade; producers MUST NOT down-convert it to a
positive readiness claim.

| `usability.availability` | Contract source |
| --- | --- |
| `documentation-only` | Delivery is `documentation-only` |
| `template-only` | Delivery is `template-only` |
| `setup-required` | Derived `setup-required` predicate in section 6 |
| `not-verified` | An implementation is present, but no current evidence establishes a target-scoped readiness predicate |
| `usable-now` | Derived `usable-now:instruction` or `usable-now:executable` predicate in section 6, with the scope exposed separately |

`usability.execution` describes technical shape and does not prove delivery or
readiness. `usability.source` describes whether metadata was declared or
inferred and does not prove the claim. Until follow-on schemas carry evidence
references and target keys, a declared or inferred `usable-now` value is an
unverified classification: consumers MUST NOT treat that field alone as
operational availability.

## 6. Derived user-facing claims

User interfaces MUST qualify positive usability by scope, even if the visual
label is shortened.

```text
usable-now:instruction(target_key) =
  lifecycle != deprecated
  AND instruction verification exactly matches target_key and is current

usable-now:executable(target_key) =
  lifecycle != deprecated
  AND delivery == implemented
  AND executable verification exactly matches target_key and is current
  AND target_key declares no unsatisfied setup precondition

setup-required(target_key) =
  lifecycle != deprecated
  AND delivery == implemented
  AND executable setup requirements are declared
  AND target_key has at least one user-actionable configuration or auth requirement unsatisfied

live-ready(environment, operation) =
  usable-now:executable for the matching evidence scope
  AND installed(environment)
  AND configured(environment)
  AND authenticated(environment) when required
  AND current security, policy, approval, and authorization gates for operation
```

An interface MAY render `usable-now` as a shared badge only when it also exposes
`instruction` or `executable` in accessible text and machine-readable output.
It MUST NOT derive the badge from prompt-test presence, schema success,
`security_reviewed: true`, or a successful installation alone.

## 7. Evidence and decision-record requirements

Every time-bound operational or security evidence record used for promotion
MUST contain:

- entry ID, module type, version, and immutable artifact digest;
- contract ID and version;
- evidence scope: `instruction`, `executable`, `setup`, `authentication`, or
  `security`;
- test identity/version and the exact declared outcome evaluated;
- runtime and platform, plus relevant dependency-lock digest;
- configuration class with secrets redacted;
- authoritative provider-target fingerprint and its attestation reference when
  applicable;
- non-secret credential-binding ID and generation/revision, or an equivalent
  safe credential-generation fingerprint, for authentication evidence;
- start time, observation time, result, and `expires_at`;
- logs or result references sufficient to reproduce or audit the conclusion;
- producer identity and, where required, reviewer/approver identity;
- explicit coverage, skipped checks, partial results, and failure reason.

Evidence with a missing required field, a result that does not satisfy the
claim-specific predicate, a skipped REQUIRED check, unknown artifact identity,
or a broader claim than observed scope is ineligible for promotion.

A delivery-classification record MUST instead contain the entry ID, module
type, version, immutable artifact or content digest, contract version,
classifier identity/version, classification time, result, and reproducible
inspection reference. It has no time-based expiry. It remains current while the
classified artifact or content identity and applicable classification rules are
unchanged.

A deprecation decision record MUST contain the entry ID and version range,
contract version, owner identity, effective time, reason, and replacement
identifier required by the current manifest contract. It MUST NOT carry an
automatic `expires_at`. Deprecation remains permanently effective for the
recorded version or version range. A later release or replacement is a new
lifecycle subject, not a restoration decision. For an entry-level manifest flag,
the record MUST use an explicit all-versions range.

### 7.1 Evidence by claim

| Claim | Additional required observation |
| --- | --- |
| `documentation-only` | Inventory confirms no claimed runnable entrypoint |
| `template-only` | Inventory identifies placeholders, missing implementation, or user-supplied dependency that prevents the claimed executable outcome |
| `implemented` | Artifact inspection proves entrypoints, dependency closure, runtime registration generation, and absence of blocking placeholders |
| `verified-usable` / instruction | Clean client obtains all instruction resources and completes representative use against declared acceptance criteria |
| `verified-usable` / executable | Clean client installs the published artifact and its documented first meaningful operation succeeds; protocol smoke test for MCP; non-destructive provider sandbox test when integrated |
| `setup-required` | Implemented evidence plus a deterministic setup check identifying only declared, user-actionable missing configuration/auth requirements |
| `security-reviewed` | Exact artifact/dependency scope, checklist/profile version, findings, accepted disposition, reviewer, review time, and risk-owner acceptance details when disposition is `approved-with-accepted-risk` |
| `deprecated` | Owner decision, effective time, reason, and replacement identifier |

Simulation, mocks, and fixtures MUST be labelled. They MAY prove deterministic
behavior but MUST NOT stand in for required clean-client, runtime, protocol,
credential, provider, or live-sandbox observations.

## 8. Deterministic promotion and demotion

States are derived from current facts; they are not rewards accumulated forever.
Given identical entry metadata, artifact identity, target scope, current time,
and evidence records, every conforming consumer MUST compute identical results.

### 8.1 Promotion

A producer MAY propose a state. A validator promotes it only when:

1. the state is allowed for the module type under section 5;
2. every required evidence field and observation under section 7 is present;
3. evidence identifiers and scope exactly match the entry, artifact, runtime,
   complete target key, and claim;
4. every REQUIRED verification step completed with no hidden skip or partial
   result, and observed findings satisfy the claim-specific predicate;
5. for time-bound operational or security evidence,
   `observed_at <= now < expires_at`;
6. no invalidation event from section 9 occurred after observation; and
7. any required reviewer or owner decision is recorded, including an accepted
   security disposition and authorized risk acceptance when applicable.

Delivery promotion uses the matching artifact-bound classification record and
does not depend on wall-clock age. Deprecation takes effect at its recorded
effective time and does not expire. Passage of time alone MUST NOT change either
classification.

Promotion MUST be atomic for one claim, evidence scope, and complete target key.
Evidence for one runtime, platform, provider target, auth method, principal,
scope set, configuration class, capability, operation set, environment, or
artifact digest MUST NOT promote another. A broader UI claim is the intersection
of its displayed scope, never the union of unrelated passing records.

### 8.2 Demotion

A validator MUST remove a derived readiness or assurance state, narrow its
scope, or recompute a delivery classification immediately when its applicable
promotion predicate becomes false. Demotion does not require an owner vote.
It MUST record the prior state, reason, invalidating event or time, and affected
scope. The underlying delivery state is then recomputed:

- missing or incomplete executable implementation demotes `implemented` to
  `template-only` when a usable scaffold remains, otherwise to
  `documentation-only` when only guidance remains;
- an actionable configuration or authentication failure on an otherwise
  implemented artifact yields `setup-required`;
- a test, runtime, provider, dependency, or first-operation failure removes
  `verified-usable` and yields `not-verified`, not `setup-required`, unless the
  failure is exactly a declared user-actionable setup precondition;
- an expired or invalidated security review removes `security-reviewed` without
  changing delivery or usability evidence;
- `deprecated` suppresses positive user-facing usability regardless of other
  retained states.

Re-promotion always requires current qualifying evidence. Editing a state field,
rerunning schema validation, or clearing a failure flag cannot re-promote it.
`deprecated` MUST NOT be demoted or transitioned back to active for its affected
version or range. A later version outside an explicitly bounded range or a
replacement entry is classified and verified as a distinct lifecycle subject.

## 9. Freshness and invalidation

Every time-bound operational verification, setup evaluation, authentication
verification, and security-review record MUST carry an explicit `expires_at`.
Its validity is the half-open interval `observed_at <= now < expires_at`; at
`expires_at` it is stale and the corresponding readiness or assurance state
MUST be removed or recomputed.

Delivery classifications and deprecation decisions MUST NOT expire merely
because time passes. Delivery is recomputed when the artifact/content identity
or classification rules change. Deprecation never changes for its affected
version or range; later owner decisions may deprecate additional versions but
MUST NOT reactivate an already deprecated lifecycle subject. Retained historical
verification never reverses it.

Unless a stricter provider or policy limit applies, `expires_at` MUST NOT exceed:

| Evidence scope | Maximum lifetime from `observed_at` |
| --- | --- |
| Instruction usability | 180 days |
| Executable clean-client/runtime verification | 90 days |
| Setup evaluation | 30 days |
| Authentication or provider sandbox verification | 30 days |
| Security review | 365 days |

The effective expiry for a composite time-bound claim is the earliest expiry of
all required time-bound evidence. Non-expiring delivery and deprecation records
do not extend or shorten that interval. Clock comparison MUST use UTC instants.
An accepted-risk disposition is time-bound by its policy-authorized acceptance
and MUST expire no later than the security review it supports.
Authentication evidence MUST expire at the earliest of its contract maximum,
the verified credential's expiry, and the provider identity attestation's
expiry. Credential refresh or replacement invalidates the evidence rather than
extending it in place.

Time-bound evidence becomes invalid before expiry when any scoped input changes,
including:

- artifact or dependency-lock digest;
- entrypoint, runtime registration, setup flow, or first meaningful operation;
- instruction content or referenced resource used by instruction evidence;
- supported runtime/platform or its incompatible major version;
- any provider-target component or attestation, including native scope identity,
  endpoint, region, sandbox/live tier, principal, API/protocol version, auth
  method, or granted scope set;
- any credential replacement, refresh, renewal, revocation, or source rebinding,
  including a change or loss of its non-secret binding ID, generation, revision,
  or safe credential-generation fingerprint;
- security checklist/profile or a security-relevant dependency;
- revocation, credential failure, policy withdrawal, or a newly observed test
  failure within the claimed scope.

Metadata-only changes that do not affect the claim MAY retain evidence only when
an automated comparison proves none of the scoped inputs changed and records
that decision. Unavailable verification infrastructure does not extend expiry.

## 10. Authority and audit boundary

| Claim | May propose | Must verify or attest | May authorize use/effect |
| --- | --- | --- | --- |
| Delivery classification | Maintainer or automation | Deterministic artifact inspection | Nobody; classification grants no execution authority |
| Operational verification | Maintainer or CI | Approved verifier observing the target | Runtime policy for the matching scope only |
| Authentication | Setup flow or user | Authoritative provider/non-destructive identity check | Provider plus local policy; identity alone is insufficient |
| Security review | Reviewer or approved scanner | Named review process plus disposition authority; policy-authorized risk owner for accepted risk | Security/policy authority for its recorded scope only |
| Deprecation | Entry owner | Repository governance | Installer/UI policy |
| Live operation | Agent or user | Current environment, provider, verification, security, and policy checks | User/organization policy and any operation-specific approval |

Model output, confidence, readable names, prompt quality, cached state, and
previously valid evidence are proposals or observations, never independent
authority to promote a claim or execute an effect.

Consumers MUST preserve evidence provenance, scope, version, and freshness.
Unknown, missing, stale, contradictory, partial, and unavailable evidence MUST
remain explicit and MUST fail closed for positive claims.

## 11. Compatibility and conformance

This is major contract version 1. A consumer conforms when it:

- implements the axes, definitions, module constraints, derived predicates,
  transition rules, and expiry semantics in this document;
- rejects unknown v1 states rather than guessing their meaning;
- does not promote from prompt tests, schema validity, or self-asserted metadata;
- exposes the evidence scope behind every positive usability claim; and
- retains enough reason and provenance data to explain promotion, demotion, and
  expiry decisions.

Compatible clarifications increment the minor or patch contract version.
Changing a state's meaning, weakening evidence, changing a derived predicate,
or permitting a new authority path requires a new major-version contract and
an explicit migration. Historical records MUST retain the contract version
under which they were evaluated.

## 12. Follow-on integration requirements

Follow-on work MUST cite `operational-readiness` version `1.x` and preserve this
contract at these boundaries:

- schemas represent the axes, evidence scope, timestamps, and provenance without
  collapsing them into one maturity enum;
- validators implement sections 5 through 9 and emit deterministic reason codes;
- installers report installed/configured/authenticated predicates separately
  and never bundle credentials;
- registry builders derive catalog claims from qualifying evidence rather than
  folder or module-name inference;
- user interfaces distinguish instruction from executable usability, show
  target coverage, setup and expiry reasons, and suppress positive claims for
  deprecated entries;
- release and clean-client tests record artifact-bound evidence rather than
  treating prompt or schema checks as operational proof.

## 13. Non-goals

This contract does not choose a manifest layout, evidence storage service,
installer implementation, provider-specific authorization flow, badge design,
or policy for approving live mutations. It does not certify current repository
entries; until follow-on audit and evidence work is complete, absent evidence
means `not-verified`.

## 14. Minimum conformance scenarios

Follow-on validators MUST exercise at least these independent-oracle cases:

1. evidence for runtime A does not verify runtime B with the same entry and
   scope;
2. instruction verification can coexist with executable `setup-required` for
   the same artifact because their scope/target keys differ;
3. operational and security states disappear at their exact UTC expiry while
   unchanged delivery classifications remain stable;
4. deprecation remains effective after all historical evidence expires and
   rejects flag clearing, owner restoration, and fresh-evidence resurrection for
   the same affected version or range;
5. `approved` and authorized `approved-with-accepted-risk` can produce
   `security-reviewed`, while rejected, incomplete, remediation-required, and
   unauthorized risk acceptance cannot;
6. prompt-test success, schema validity, installation success, and
   `usability.availability: usable-now` each fail independently to prove
   executable operational availability; and
7. substituting a provider tenant, account, project, endpoint, region,
   sandbox/live tier, principal, or scope set fails target matching even when the
   package, runtime, provider type, API version, and auth method are unchanged;
8. replacing credential A with expired, malformed, revoked, or otherwise
   unusable credential B invalidates A's authentication evidence immediately,
   even when provider target, principal, scopes, auth method, and credential
   binding ID are unchanged, because the generation/revision must change; and
9. changing any member of a target key invalidates or narrows the matching
   readiness claim without affecting evidence for a different target key.

These are contract test vectors, not evidence that the current repository has a
conforming validator. Until the follow-on validator work implements them, CI
success proves only the checks it actually executes.
