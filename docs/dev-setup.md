# Development Setup

## Prerequisites

- Go `>= 1.22`
- Python `>= 3.10`
- Node `>= 20`
- pnpm `>= 10`
- `check-jsonschema` (for manifest/schema checks)

## Install Options for `check-jsonschema`

### Option A: pipx (recommended for global CLI tools)

```bash
pipx install check-jsonschema
```

### Option B: pip in a virtual environment

```bash
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install --upgrade pip check-jsonschema
```

### Option C: uv tool install (if you use uv)

```bash
uv tool install check-jsonschema
```

## Verify Local Environment

```bash
make doctor
```

## Common Local Commands

```bash
make registry
make manifests
make test
make cli-test
make cli-build
```

## Web App + E2E QA

```bash
cd apps/web
pnpm install
pnpm test:e2e:setup
pnpm test:e2e
```
