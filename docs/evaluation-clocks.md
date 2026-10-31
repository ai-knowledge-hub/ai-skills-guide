# Isolated package evaluation clocks

Implemented for #86 on 31 October 2026 using the private
failure-space-test-design skill. This replaces the process-wide package clock
introduced by #82. It changes time ownership, not the expiry policy in #83.

## Time ownership and authority

| Domain | Evaluation source | Purpose / evidence |
| --- | --- | --- |
| Operational package admission | `registry.ValidatePackageManifestAt(path, instant)` | Existing v1 manifest-age and future-clock checks; zero uses real time. The default validator captures real time. |
| Install/preflight/compile operation | Immutable `installer.Evaluation` value | `EvaluationAt(instant)` retains one UTC instant through root admission, dependencies, remote closure plans and runtime compilation. Zero captures real time at operation entry. |
| Structural repository/index and retained-release validation | No evaluation clock | Schema, layout, graph, digest and immutable-history checks remain independent of elapsed age. |
| Composite provider readiness | Existing `CompositeStatusOptions.Now` | One captured instant reaches nested package admission and each provider's credential/attestation evaluation. It is independent of other simultaneous operations. |
| Live authentication/configuration and runtime execution | Existing readiness/agent clocks | Those defaults, credential expiry, revocation, scope/target and generation checks remain active. An installer evaluation cannot override them. |
| Receipt timestamps, cache names, network timeouts and timers | Actual wall/elapsed time | An historical package evaluation does not fabricate an installation time, freeze a request timeout or reuse a cache filename. |
| Editorial API/SDK version notes | Informational review date | [Catalogue compatibility v1](../shared/contracts/catalogue-compatibility-v1.md) has no age-based content expiry. These dates are not operational clocks. |

**Confirmed:** all ten test call sites that assigned the shared package clock
now use explicit parameters or operation values. `internal/evidenceclock` and
its mutable `Now` function, together with the global test helper, are removed.
The discovery inventory scanned 1,091 files; the clock map above was confirmed
by tracing executable call paths, not by interpreting those counts as coverage.

## Operation composition

Existing installer function signatures remain available. Each is a wrapper
that starts a real-time evaluation. Internal callers that compose several steps
can reuse one operation value:

```go
evaluation := installer.EvaluationAt(instant)
manifest, err := evaluation.ValidatePackageManifest(manifestPath)
// The same value can preflight dependencies, install or compile this operation.
```

The CLI install path creates one real-time value and carries it through local
or remote admission, preflight, dependency installation and compilation. Nested
methods reuse that value instead of invoking a default-clock wrapper. Values
are copied; there is no mutable shared clock or clock cleanup to coordinate.

Historical installed-contract command tests use private admission-time
parameters. Authentication configuration and provider commands keep their own
time behavior. The explicit time is an internal Go testing/composition seam,
not a new CLI flag, environment setting, serialized contract or user bypass.

The zero `Evaluation` value also works: an entry-point method normalizes it
once before forwarding it. To share a snapshot across separate entry-point
calls, construct and reuse `EvaluationAt(instant)` rather than reusing an
uninitialized zero value.

## Invariants and independent oracles

| Protected invariant | Observable test evidence |
| --- | --- |
| Independent evaluations cannot leak time into one another | `TestEvaluationsAdmitSameRetainedPackageConcurrentlyAtDifferentInstants` runs three parallel evaluations against one extracted GA4 0.2.0 release, ten admissions each. Before the v1 deadline succeeds; at/after it fail for expiry. Manifest bytes remain equal. |
| Nested package and credential checks use the composite instant | `TestCompositeStatusUsesSuppliedEvaluationTime` reads one installed closure and configured profiles concurrently at ready, credential-deadline and package-deadline instants. Both providers reflect the selected state; actual credential observations retain the supplied instant. |
| Production defaults still enforce current admission | `TestEvaluationDefaultsUseRealTimeWithoutChangingExplicitContexts` compares explicit historical, zero-value, zero-instant and default registry calls. `TestBuiltBinaryRejectsExpiredPackageUsingRealTime` builds and invokes the actual CLI, rejects an otherwise admissible expired scratch package and verifies no installed target was created. |
| Expiry boundaries retain their approved semantics | Four existing evidence scopes retain before/at/after cases under operational v1. The future-clock tolerance has explicit boundaries. `TestAuthenticationDeadlineBoundariesUseSuppliedTime` separately tests actual credential and provider-attestation deadlines. |
| Structural integrity and retained history do not depend on today's date | The structural test uses explicit instants only for its operational comparison. Historical archive/installer/agent tests pass their own admissible instant without changing manifests, archives, hashes or verification dates. |

The independent time oracles are the retained manifest's observation date,
[operational v1](../shared/contracts/operational-readiness-v1.md)'s existing
deadlines and the provider fixture's explicit credential/attestation expiry.
The tests deliberately do not derive the expected 30/90/180-day limits from
production constants. This preserves current policy while leaving its rationale
and any future migration to #83. A review date or API version does not renew
credentials or any operational evidence.

## Bounded failure space and counter-review

The selected interaction clusters are evaluation identity × deadline boundary,
root/child admission × clock forwarding, and production entry point × expired
evidence × filesystem effect. Tests use explicit instants rather than sleeps or
changing fixture dates as the calendar advances.

- Retained-package isolation: 3/3 boundary instants, 30 admissions in parallel.
- Composite isolation: 3/3 instants across two providers, six provider outcomes.
- Legacy evidence deadlines: 12/12 existing scope/boundary cases retained.
- Future-clock tolerance: 3/3 boundary cases.
- Credential and provider-attestation expiry: 6/6 boundary cases.
- Production defaults: zero-value, zero-instant and default registry rejection,
  plus an actual CLI rejection before installation; historical positive controls
  ensure those failures are not malformed fixtures.

All 5/5 selected harmful mutations were detected after passing controls, one
source change at a time, with original bytes restored after each case:

| Deliberate mutation | Falsifying test |
| --- | --- |
| Discard `Evaluation.at` and call default admission | Parallel retained-package evaluation |
| Drop the composite's instant from nested package admission | Composite supplied-time test |
| Drop the composite's instant from credential status | Composite supplied-time test |
| Reset agent dependency preflight to the default wrapper | Historical pinned-agent install/compile test |
| Freeze the production CLI evaluation at a historical instant | Built-binary current-admission test |

The focused tests are reproducible with `go test -count=1` and the named tests
above. Mutation results and the broader Go/race and registry/schema gates are
recorded in the PR. This is a bounded probe of time-routing and authority
defaults, not exhaustive interleaving or fault coverage.

## Limits

This does not make every repository test parallel: environment changes,
transaction fault hooks and runtime model trust fixtures still have separate
isolation requirements. It removes the package-clock reason for serialization.
Provider availability, actual vendor compatibility and the justification for
the legacy time ceilings are not established by these tests. No runtime age
cap, credential rule, release identity or static-catalogue policy is changed.
