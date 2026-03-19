# Security and Review Guide

## Risk model

Skills may call tools, run scripts, and influence sensitive workflows.
This now includes engineering maintenance, security review, and harness
evolution packs in addition to marketing and adtech workflows.

## Required safeguards

- No hardcoded secrets in skills/scripts.
- Explicit confirmation for destructive actions.
- Clear fallback when data sources fail.
- Log assumptions and uncertainty in outputs.
- Treat external content and MCP output as untrusted unless policy says
  otherwise.
- Security and agentops skills must halt or escalate when scope drifts into
  protected paths.

## Reviewer focus

- Behavioral regressions
- Data integrity risk
- Compliance drift
- Over-broad automation permissions
- Prompt-injection susceptibility
- Supply-chain or secret-handling regressions
- Unauthorized edits outside harness-owned paths
