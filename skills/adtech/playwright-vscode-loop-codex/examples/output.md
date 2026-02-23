# Example Output

## Test Plan
- Routes: `/`, `/skills`, `/skills/marketing/meta-google-weekly-performance-review`
- Interactions: filter apply, copy command button click

## Generated/Updated Files
- `apps/web/playwright.config.ts`
- `apps/web/e2e/smoke.spec.ts`

## Run Summary
- Passed: 3
- Failed: 0

## Healing Notes
- Updated one selector from text match to role+name for stability.

## Residual Risk
- Dynamic content cards may need test-id anchors if UI copy changes frequently.
