# Catalogue compatibility implementation

The website uses maintainer-updated content under
[catalogue compatibility v1](../shared/contracts/catalogue-compatibility-v1.md).
See [the maintainer guide](maintaining-compatibility.md) for the update process.

## Current implementation

- Editorial notes live in `apps/web/lib/catalogueCompatibility.ts` and match the
  exact module, package ID and release version. They include a source, review
  date, relevant versions and limitations.
- All four detail families display a matching note. Where no note exists, they
  retain the package's own manifest declarations or report unknown information.
- Schema 2.2's optional manifest declarations remain supported. They describe
  maintainer requirements and do not prove independent compatibility testing.
- Static builds publish these notes without provider credentials, live calls,
  signing keys, a verification service or an evaluator dependency.
- Notes cover all seven implemented tools, including GA4, BigQuery and Meta
  Ads. The [release review inventory](tool-version-review.md) records their
  sources and the template-only exclusion.

## Removed prototype and historical work

PR #88's standalone evaluator, its internal package, synthetic example and
observation schema were removed after a repository consumer audit. Only that
command and its tests depended on the package; no site, registry or installer
code called it. The former design is recoverable in Git history. Its historical
[guide](compatibility-evaluation.md) records the retirement.

PR #89's signed GA4 fixture producer was closed without merging. The staged
operational v2 design was withdrawn in favour of the separate catalogue policy.
The useful schema, typed declaration support, CLI declaration display and static
website presentation from #87 remain.

## Separate runtime work

Operational v1 admission, artifact checks, credential expiry, target identity,
revocation and existing age gates remain unchanged. The 30-day runtime gate is
still tracked under #83. Removing an unused diagnostic must not be presented as
having changed that behavior. Further runtime changes require a focused consumer
and authority review independent of catalogue editorial content.
