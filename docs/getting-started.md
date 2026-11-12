# Getting Started

## Track A: Marketing Practitioner (no coding required)

1. Choose one entry matching your workflow:
   - `skills/` for task-level expertise
   - `agents/` for orchestrated templates
   - `plugins/` for reference compositions and implemented bundles
   - `tools-mcp/` for integration connectors
2. Check the entry's usability label and limitations. Treat `not-verified` as implemented but unproven for your target; use `setup-required` only after completing every listed setup item.
3. Install instruction or implemented entries only when their availability permits it. For `template-only`, inspect or copy repository source as a scaffold without runtime registration.
4. Run the prompts in `tests/test-prompts.md`.
5. Evaluate output consistency against expected format.
6. Tune wording and constraints, then re-test.

## Install published files

Reading or copying instructions from the site needs no installer. For file
installation, open the resource's installation section, prepare the installer,
then choose its Codex, Claude or generic adapter. Commands remain readable with
JavaScript disabled; expand the relevant adapter and select text to copy.

Download the installation-only [v0.4.0-alpha.1 release](https://github.com/ai-knowledge-hub/ai-skills-guide/releases/tag/v0.4.0-alpha.1)
for your OS and architecture and compare its SHA-256 with the matching entry in
`SHA256SUMS`. Rename it to `skills-hub` (`skills-hub.exe` on Windows) and add its
directory to PATH. On macOS/Linux, make it executable with `chmod +x ./skills-hub`.
Check `skills-hub help`. If it is outside PATH, use its absolute path instead of
`skills-hub` in the examples. Published assets do not establish every native
application/version combination; [#52 tracks that evidence](https://github.com/ai-knowledge-hub/ai-skills-guide/issues/52).

With that installer on PATH, this command can run outside a repository checkout:

```sh
skills-hub install --source remote --module skills --registry-url https://skills.ai-knowledge-hub.org/registry/skills-index.json --entry marketing/meta-google-weekly-performance-review@0.1.0 --runtime codex
```

The adapter's default skills root is `~/.codex/skills`, unless `CODEX_HOME`
changes it. The installer appends `marketing/meta-google-weekly-performance-review`.
Confirm your app discovers that location; copying files is not native adoption.
Claude uses `~/.claude/<module>` with `CLAUDE_HOME`/`CLAUDE_CODE_HOME` overrides.
Generic requires an explicit `--target` module directory relative to your chosen
working directory, or an absolute target. Bundle dependencies can also write
sibling module directories. Agent dependency closures can also write sibling
skills, agents and tools. Review their generated adapter configuration before
connecting them to your app. The installer reports the actual destinations.

The site derives execution environments and common supported hosts from the
selected release and the manifests in its retained dependency closure. For
example, the weekly reporting agent needs Node.js 22 for its bundled GA4 and
BigQuery servers even though the agent has no executable root entrypoint. These
dependencies restrict its installer hosts to Linux/macOS; a Windows CLI binary
does not make that package installable on Windows. Mixed closures may require
both Node.js and Python. If the complete requirements cannot be read, the site
withholds a generated command and directs you to inspect the release.

Run `skills-hub help setup` after installation. Follow the resource's documented
runtime, working directory, inputs and expected output for its first operation.
Mock results, native discovery and connected provider operation are separate
checks. The public installer does not run workflows or acquire credentials.

### Developer checkout alternative

With Git, Go 1.22+ and Make installed:

```sh
git clone https://github.com/ai-knowledge-hub/ai-skills-guide.git
cd ai-skills-guide
make cli-build
./bin/skills-hub help
```

This is an unreleased source build. From the checkout root, substitute
`./bin/skills-hub` for `skills-hub` in remote commands. Keep `--source remote`
and the HTTPS index for published files. The default `--source local` instead
reads mutable checkout content for development.

## Track B: Ad-Tech Engineer

1. Start with the same steps as Track A.
2. Move deterministic logic into `scripts/`.
3. Add strict input validation and failure handling.
4. Add sample data files under `examples/`.
5. Open a PR with test evidence.

## Runtime notes

These skills are authored for Agent Skills-style runtimes and can be adapted to Codex, Claude-style, and similar ecosystems.

## Hub UI quick check

If you are working on the website catalog:

1. `cd apps/web`
2. `pnpm install`
3. `pnpm dev`
4. `pnpm test:e2e`

Primary routes:
- `/skills`
- `/agents`
- `/tools-mcp`
- `/plugins`

Read [using-the-catalog.md](using-the-catalog.md) before assuming installation means an external integration is connected.
