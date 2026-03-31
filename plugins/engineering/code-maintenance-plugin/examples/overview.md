# Code Maintenance Plugin Example

## Input
- Changed backend and frontend files
- Failing lint in one package
- No recent coverage report for changed modules

## Expected plugin use
1. Plan repo impact with `engineering/implementation-strategy`
2. Run stack-aware verification with `engineering/code-change-verification`
3. Identify missing scenarios with `engineering/test-gap-analyzer`
4. Review coverage blind spots with `engineering/coverage-gap-reporter`
5. Draft findings and PR summary with `engineering/pr-review-and-draft`
