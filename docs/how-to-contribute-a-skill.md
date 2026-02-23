# How to Contribute a Skill

1. Start from `templates/skill-template/`.
2. Name the skill with lowercase hyphen format.
3. Write clear routing text in `description` and `When to use`.
4. Add realistic tests and example artifacts.
5. Run `bash scripts/validate-skills.sh`.
6. If your change touches the hub UI, run `pnpm test:e2e` from `apps/web`.
7. Open a PR with test evidence and assumptions.
