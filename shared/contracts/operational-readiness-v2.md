# Withdrawn operational compatibility design v2

The staged v2 operational verification design introduced by PR #87 was withdrawn
when the owner selected maintainer-updated information for a content-only site.
It was not adopted by runtime admission. This file remains as a historical link
so its status is explicit; it is not an active normative contract.

Current website policy is
[catalogue compatibility v1](catalogue-compatibility-v1.md).
Runtime admission and credential rules remain governed by
[operational readiness v3](operational-readiness-v3.md), with historical package
dates separated from actual authentication expiry.

The offline evaluator prototype from PR #88 has been removed after a consumer
audit found no site, registry, installer or runtime dependency. The signed GA4
fixture producer in PR #89 was closed without merging. Earlier designs remain
recoverable in Git and their pull requests.
