---
name: adtech-compliance-checker
description: Review ad copy, landing links, and tracking conventions for policy and implementation compliance. Use before campaign launch or when diagnosing disapprovals.
---

# Ad-Tech Compliance Checker

## When to use
Use for pre-launch QA, disapproval triage, UTM linting, and policy-sensitive copy review.

## Inputs required
- ad copy (headlines, descriptions, primary text)
- destination URLs
- target platform(s)

## Workflow
1. Parse creative text and destination URLs.
2. Validate URL and tracking conventions.
3. Check policy-sensitive patterns for selected platforms.
4. Label findings by severity and provide suggested fixes.

## Output format
- Overall status: PASS/FAIL
- Findings table: severity, category, issue, fix
- Launch recommendation

## Guardrails
- Do not claim platform approval certainty.
- Mark ambiguous issues as "needs human review".
- Require confirmation before any automated publishing action.
