# Test Prompts

1. Preview a closed Google Ads campaign-budget plan and stop without claiming an execution grant.
2. Attempt to replace the approved account and resource ID while preserving the rest of the grant; reject before any provider call.
3. Execute the same approved DV360 line-item status change concurrently and demonstrate one provider mutation and one durable receipt.
4. Reconcile a timeout after a provider mutation when current state matches the proposed post-image, without replaying the write.
5. Reject rollback when another authorized writer changed provider state after the original verified execution.
6. Prove that an unexpected post-write third state is not overwritten and requires reconciliation or separately authorized rollback.
