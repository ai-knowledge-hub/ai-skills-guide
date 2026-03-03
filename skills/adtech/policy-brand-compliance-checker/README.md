# Policy + Brand Compliance Checker

Use this skill to validate ad copy, URLs, and tracking conventions
before launch.

## What this skill does

- Checks policy and brand rule compliance.
- Validates URLs and UTMs.
- Produces severity-ranked findings with fixes.

## Before you start

1. Gather ad copy and destination URLs.
2. Specify target platform.
3. Confirm policy rules and brand constraints.

## Install

```bash
./bin/skills-hub install \
  adtech/policy-brand-compliance-checker@latest \
  --runtime codex
```

## First run prompt

```text
Use Policy + Brand Compliance Checker.
Platform: Meta Ads
Validate ad copy and landing URL set.
Return PASS/FAIL, findings table, and launch recommendation.
```

## What good output looks like

- Every issue includes severity and fix.
- Ambiguous cases are marked for review.
- Recommendation is explicit.

## Beginner safety checklist

- Do not assume guaranteed ad approval.
- Escalate unclear policy cases.
- Do not modify live assets automatically.
