# Security and Review Guide

## Risk model

Skills may call tools, run scripts, and influence sensitive workflows.

## Required safeguards

- No hardcoded secrets in skills/scripts.
- Explicit confirmation for destructive actions.
- Clear fallback when data sources fail.
- Log assumptions and uncertainty in outputs.

## Reviewer focus

- Behavioral regressions
- Data integrity risk
- Compliance drift
- Over-broad automation permissions
