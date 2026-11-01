# Runtime evidence policy migration (#83)

<!-- markdownlint-disable MD013 -->

Decision date: 1 November 2026. Normative authority:
[operational readiness v3](../shared/contracts/operational-readiness-v3.md).
The withdrawn operational v2 design stays withdrawn.

## Findings and decision

**Confirmed:** v1's 30-day package gate came from repository policy, not a vendor
API or SDK requirement. It derived a deadline from `last_verified_at` and whether
authentication was declared, even before a user could configure credentials.
The related 90/180-day package gates had the same problem. Actual authentication
also imposed a separate 30-day maximum even when the provider supplied a longer
valid credential and attestation interval.

Remove these implicit package-age gates and the authentication maximum. Keep
actual provider deadlines and current checks. Static API/SDK notes remain
maintainer-updated content keyed to exact releases. No evaluator, signing
service, background process or API probe is added to the website.

**Confirmed consumer path:** CLI local/remote install -> installer evaluation ->
registry schema/layout/closure admission -> staged commit guards. These paths
still enforce evidence references, timestamp parsing and future-observation
sanity, plus integrity, platform, runtime and lifecycle constraints. They no
longer reject a package because its observation is old. Admission grants no
credential or effect authority.

**Confirmed live path:** runtime status/doctor/smoke/composite -> current
installation/target checks -> configured provider validator -> credential
identity, scope, account, generation, revocation and explicit deadlines. A
provider attestation without an expiry is rejected. If the credential has no
expiry, the mandatory attestation still bounds authentication. Each status call
rechecks the provider; a stored successful report is not reused as authority.

**Confirmed static paths:** the website loader applies
`apps/web/lib/currentCatalogueProjection.mjs` to canonical declarations before
adding compatibility notes. The no-Go Vercel publisher uses the same projection;
the normal publisher uses the Go projection. Canonical indexes and immutable
release sidecars remain declarations, not current-readiness output. Opaque
evidence references and dates cannot prove current target readiness; unresolved positive claims are demoted irrespective of age.
Exact-release compatibility annotations remain a separate editorial display.

**Unknown:** this work does not establish real vendor compatibility, guarantee
that maintainers detect every vendor change, or prove that a configured provider
validator tells the truth. Validator trust and provider provenance remain the
operator's responsibility. Missing version information stays unknown.

## Evidence scopes retained

- Five-minute future-observation tolerance: timestamp sanity only. It never
  extends a real expiry. Package clocks stay operation-local and live by default.
- Doctor: recomputed inspection; its persisted snapshot has equal observation
  and expiry times. Current checks may succeed without implying a reusable lease.
- Smoke: explicit 90-day maximum for reusing a local execution result, shortened
  by authentication deadlines. This separate execution recheck window is retained
  because not all environment changes are fingerprinted; it does not invalidate
  package or API/SDK support. Maintainers own this policy and may revisit it.
- Security: existing explicit review/risk deadlines and 365-day maximum stay
  separate; no security-review enforcement or evidence is invented here.

## Invariants and bounded tests

| Invariant | Independent oracle and exercise |
| --- | --- |
| Age does not reject otherwise valid package metadata | Four scopes x ten literal ages around former 30/90/180-day boundaries and 100 years: 40 cases; actual package admission and an unchanged retained GA4 release are also exercised |
| Missing/malformed/future evidence is not repaired by removing expiry | Reference and timestamp rejection tests; 3 future-skew boundaries; parallel contexts use the same immutable release |
| Provider deadlines replace a fixed authentication lifetime | Four explicit credential/attestation combinations, including 45/60-day limits and no credential expiry; each includes before/at/after actual deadline: 12 cases |
| Long-lived credentials do not grant missing authority | Four distinct missing-attestation/revoked/wrong-scope/wrong-account cases, plus retained generation/target mutation and credential boundary suites |
| Old package observations do not prove readiness | Composite starts unauthenticated despite 2020 metadata, then becomes ready only after current provider checks; static projections remain conservative for recent and ten-year-old observations |
| Elapsed install time is not a package expiry | HTTPS artifact download advances the operation clock 100 years and installs without changing metadata |
| Admission and commit still use the current clock | Six download/skew cases; forced replacement and two closure activation/rollback cases model a backward clock shift making a child observation future-dated |
| Local execution receipts remain bounded | Smoke with 120-day credentials gets a 90-day deadline; tampered lifetime fails; doctor has no reusable lease |
| Historical receipts do not gain authority on upgrade | Legacy/current composite contract acceptance with exact child deadlines, and smoke readers tested against both policy versions with an unchanged explicit expiry |

All time values are supplied explicitly; no calendar updates or sleeps are
introduced. Cases model named finite interactions, not every possible schedule.
The discovery scanner inventoried 1,093 files; that count is not test coverage.
The private failure-space-test-design skill guided the authority map and probes.

## Counter-review

Six of six selected harmful mutations were detected by the intended assertion
after passing controls: restoring package age rejection, restoring the 30-day
authentication cap, ignoring the earlier credential deadline, ignoring
revocation, extending a stored receipt beyond its explicit expiry, and promoting
an unresolved static claim. Every source file was restored after each probe.
These are bounded semantic probes, not proof of all possible authority failures.

## Migration and verification

New doctor/smoke records carry contract `3.0.0`. New readers also accept original
`1.0.3` records and preserve their explicit deadlines, target matching and failed
observations. Old readers may reject the newer receipts or keep the older
package-age policy. Roll back the consumer conservatively and rerun checks;
never rewrite a retained release or relabel historical evidence to satisfy it.

Release archives, manifest observations, canonical declarations and their hashes
are unchanged. The website stays content-only. See the normative contract for
worked examples and the complete policy decision table.

Local verification passed: full Go suite; race checks across installer, registry,
readiness, agents and CLI; final historical-clock controls; CLI build; canonical
registry and schema gates; and a Node 20 production site build (76 pages, lint
and types). Registry indexes and immutable source/release bytes remain unchanged.
No provider calls or credentials were used. The consumer follow-up also passed
all 25 Node 20 browser tests, including real catalogue and capability displays.

## Static consumer review follow-up

Two regressions now cover the consumers as well as the Go producer: the loader
cannot return raw positive declarations, and the Vercel fallback cannot promote
recent opaque evidence. Forty fixtures compare the fallback export with the Go
projection across five aggregate states, four observation forms (recent, old,
future and missing), and active/deprecated lifecycle. Literal v3 checks anchor
the Go aggregate/capability oracle. Each JavaScript result also preserves its
input and is idempotent.

The real no-Go publishing path rejects unresolved positive aggregate/capability
claims. Browser checks exercise plugin list/detail badges and each advertising
experiment capability; loader tests also preserve release hashes, versions,
verification history and compatibility declarations. Pure annotation decoration
is tested separately from readiness projection.

Both selected consumer regressions were reproduced by bounded mutations after
passing controls: bypassing the loader projection and restoring the old
age-based fallback. Each produced its intended assertion failure (2/2), and
sources were restored before the final browser/build checks.
