---
name: secrets-and-credential-hygiene
description: Review code, config, and environment conventions for secret
  leakage, risky storage, and over-broad credential scope.
---

# Secrets and Credential Hygiene

## When to use
- Use on diffs that touch config, CI, auth, or environment files.
- Use during repo security review or before sharing code externally.
- Use when a token, key, or secret may have leaked into logs or source.

## Inputs required
- diff or file set
- environment and config file paths
- scanner output if available

## Workflow
1. Inspect changed files for hard-coded credentials or risky placeholders.
2. Review environment and config patterns for scope, storage, and exposure.
3. Classify findings by severity and remediation urgency.
4. Recommend rotation, redaction, or secret-manager migration as needed.
5. Escalate immediately on confirmed secret leakage.

## Output format
- Findings Summary
- Exposed Secret Candidates
- Risky Storage Patterns
- Remediation Steps
- Escalation Status

## Guardrails
- Do not print or repeat full secrets in output.
- Redact suspected credential values.
- Treat uncertain findings as sensitive until reviewed.
- Require human review for any credential rotation or revocation action.

## Failure modes
- If scanner tools are missing, do a manual pattern review and note it.
- If a value is truncated or partial, mark confidence accordingly.
- If logs contain secrets, recommend cleanup and retention review.
