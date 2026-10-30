# Test Prompts

1. Check policy for an exact Google Ads budget change, request the required approval, and stop before authorization while approval is pending.
2. Attempt to substitute another account, action input hash, actor, and tenant into an existing decision or approval.
3. Retry the same policy check, approval request, and action lifecycle event, then try conflicting duplicates and reordered terminal events.
4. Interrupt persistence after transaction intent, domain record, and audit append; restart and verify deterministic recovery.
5. Delete, substitute, and reorder audit events—including the tail—and verify that the chain and signed head detect each mutation.
