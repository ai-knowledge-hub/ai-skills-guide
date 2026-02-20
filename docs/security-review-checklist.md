# Security Review Checklist

- Confirm external tool/API calls are necessary and scoped.
- Identify file write, shell execution, and publish actions.
- Ensure secrets are not embedded in `SKILL.md` or scripts.
- Require explicit user confirmation for destructive operations.
- Add fallback behavior for tool/API failure.
- Verify logs or output preserve traceability.
