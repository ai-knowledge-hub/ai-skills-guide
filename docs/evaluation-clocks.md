# Isolated package evaluation clocks

Implemented for #86 in PR #94 on 31 October 2026. The process-global package clock
and all ten test overrides were replaced by operation-local values. The later
[#83 policy migration](runtime-evidence-policy.md) removes implicit package-age
expiry; the isolation and live production defaults remain.

## Time ownership

| Domain | Clock and behavior |
| --- | --- |
| Package admission | `registry.ValidatePackageManifestAt(path, instant)` checks observation sanity; zero defaults to real time. Historical age is not expiry. |
| Installer operation | Zero `installer.Evaluation` and `EvaluationAt(time.Time{})` use live time per boundary. Explicit nonzero instants fix historical test time. Root and child checks share one sampled instant per check. |
| Structural repository/index and retained releases | No age gate. Schema, layout, identity, graph and immutable history checks remain. |
| Composite readiness | `CompositeStatusOptions.Now` reaches nested package and provider checks; it is independent of other operations. |
| Authentication and execution | Existing readiness/agent clocks retain actual credential, attestation, smoke and execution deadlines, scope/target and generation checks. An installer clock does not override them. |
| Receipts, caches, network timers | Actual wall or elapsed time; a historical package context does not fabricate an installation date or freeze a timeout. |
| Editorial compatibility dates | Historical context, with no elapsed-age expiry or live authority. |

## Operation composition

Existing installer function signatures bind a live clock. Trusted internal tests
can compose admission, preflight, installation and compilation with
`installer.EvaluationAt(instant)`. There is no CLI flag, environment override,
serialized bypass or process-global clock.

The CLI carries its isolated live clock through local/remote admission,
dependency preflight and compilation. Downloads finish before admission samples
time. Staged root/child checks run before activation and closure finalization;
failures use existing rollback paths. These now enforce observation sanity,
including backward clock shifts, rather than an implicit age deadline.

Commit guards check already admitted metadata, since compilation may legitimately
rewrite artifacts. Original archive admission still checks integrity. Explicit
fixed historical evaluations remain fixed when transport time changes.

## Current bounded evidence

- Three concurrent instants around the five-minute future-observation boundary,
  ten admissions each, against identical retained GA4 bytes.
- Three concurrent composite instants: ready, actual credential expiry, and a
  future package observation; both providers use the supplied instant.
- Default zero-value, zero-instant and registry rejection of future observations;
  an actual built CLI rejects before creating its installation target.
- Three future-skew boundaries and six actual credential/attestation deadline
  boundaries, all deterministic.
- Six registry/artifact download-skew combinations, one forced replacement and
  two closure rollback cases with an operation-local changing clock. A separate
  live-clock forward-time control installs after 100 years of elapsed age.
- Fixed historical transport-clock control and unchanged immutable archive tests.

Independent oracles are the retained observation timestamp, v3's five-minute
sanity rule, and explicit provider deadlines. Metadata dates cannot renew live
authority. See [runtime policy tests](runtime-evidence-policy.md) for the expiry
migration and assurance limits.

## Historical counter-review

At merge, #86 detected 5/5 selected isolation/default mutations and 3/3 remote
freshness regressions after passing controls and restored original sources.
Those results describe v1's then-active expiry policy. Age-expiry cases have
since been replaced with v3 observation-sanity and actual-provider-deadline
oracles; they do not claim the old package expiry guard remains active.

This removes the package clock reason for serial tests. Environment mutation,
transaction fault hooks and runtime trust fixtures retain separate isolation
requirements. It does not prove real vendor compatibility or exhaustive
concurrency coverage.
