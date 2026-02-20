---
name: policy-brand-compliance-checker
description: Validate ad copy, landing pages, and UTM tagging for platform policy and brand compliance before campaign launch.
---

# Policy + Brand Compliance Checker

## When to use
Use before launch, after ad disapprovals, or when auditing tracking and brand alignment.

## Inputs required
- ad_copy
- destination_urls
- target_platform

## Workflow
1. Parse text and URL components.
2. Run UTM and URL validation.
3. Check policy/brand rule sets.
4. Label issues by severity and remediation.

## Output format
- PASS/FAIL
- Findings table (severity, issue, fix)
- Launch recommendation

## Guardrails
- Do not claim guaranteed approval.
- Escalate ambiguous policy cases.
- Never modify live assets without confirmation.
