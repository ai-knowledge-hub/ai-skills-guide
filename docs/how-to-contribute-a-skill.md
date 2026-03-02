# How to Contribute a Module Entry

1. Choose a module:
   - skills (`skills/<domain>/<slug>`)
   - agents (`agents/<domain>/<slug>`)
   - tools-mcp (`tools-mcp/<domain>/<slug>`)
2. Name the folder with lowercase hyphen format.
3. Add a clear manifest:
   - skills: `skill.yaml`
   - agents: `agent.yaml`
   - tools-mcp: `tool.yaml`
4. Add required files:
   - spec file (`SKILL.md` or `AGENT.md` or `TOOL.md`)
   - `README.md`
   - `tests/test-prompts.md` (>= 5 prompts)
   - `examples/` sample artifacts
5. Run `bash scripts/validate-skills.sh`.
6. Run `go run ./cmd/registry-builder` to refresh indexes.
7. Run `bash scripts/validate-manifests.sh` (requires `check-jsonschema`).
8. If your change touches the hub UI, run `pnpm test:e2e` from `apps/web`.
9. Open a PR with test evidence and assumptions.
