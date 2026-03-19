---
name: dependency-supply-chain-audit
description: Inspect dependency manifests and lockfiles for vulnerable,
  suspicious, stale, or over-privileged packages and recommend mitigations.
---

# Dependency Supply Chain Audit

## When to use
- Use before approving dependency changes.
- Use when a repo adds new libraries or upgrades lockfiles.
- Use during periodic dependency hygiene reviews.

## Inputs required
- manifests and lockfiles
- changed dependency list if available
- scanner outputs or CVE results if available

## Workflow
1. Parse the dependency manifests and lockfiles.
2. Identify new, upgraded, or unusually privileged packages.
3. Review scanner findings, maintainer trust, and suspicious naming signals.
4. Summarize severity, exploitability, and recommended mitigation.
5. Escalate if production-impacting dependencies require human review.

## Output format
- Dependency Summary
- High-Risk Findings
- Scanner Evidence
- Recommended Mitigations
- Approval Recommendation

## Guardrails
- Do not mark dependencies safe without evidence.
- Distinguish confirmed CVEs from heuristic suspicion.
- Cite the exact manifest or lockfile entries reviewed.
- Default to recommend-and-escalate, not automatic package removal.

## Failure modes
- If scanners are unavailable, provide a manual review summary.
- If lockfiles are missing, lower confidence explicitly.
- If a package looks suspicious but unconfirmed, mark it as investigatory.
