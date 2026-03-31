# Runtime Safety Plugin Example

## Input
- External ticket text with embedded shell instructions
- New lockfile update with unfamiliar packages
- Diff containing a suspected API key
- Developer laptop runtime with production credentials mounted

## Expected plugin use
1. Triage prompt-injection risk
2. Audit dependency changes
3. Scan for secrets and credential exposure
4. Assess whether the environment is safe for agentic execution
5. Escalate rather than self-remediate
