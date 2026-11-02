#!/usr/bin/env python3
"""Bounded, serial mutation check for #85. Restores each source in finally."""
from pathlib import Path
import os
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
ENV = os.environ.copy()
GO_COMMAND = ["go", "test", "./internal/registry", "-run",
              "^TestCatalogClassificationsAreDeclaredAndTruthful$", "-count=1"]
WEB_COMMAND = ["pnpm", "exec", "playwright", "test", "catalogue-contract.spec.ts",
               "-g", "annotation selection"]
BUILDER = ROOT / "internal/registry/builder.go"
LOOKUP = ROOT / "apps/web/lib/catalogueCompatibility.ts"
NEEDLE = '\tif strings.HasPrefix(m.SchemaVersion, "2.") {'
CASES = [
    ("substitute BigQuery identity", BUILDER, NEEDLE,
     '\tif m.ID == "warehouse/bigquery-mcp-query-runner" { entry.ID = "warehouse/other-query-runner" }\n' + NEEDLE,
     GO_COMMAND, ROOT, "missing required catalogue identity"),
    ("hide weekly reporting helper", BUILDER, NEEDLE,
     '\tif m.ID == "marketing/meta-google-weekly-performance-review" { entry.Usability.ExecutableHelpers = nil }\n' + NEEDLE,
     GO_COMMAND, ROOT, "missing required helper"),
    ("promote unverified reconciler", BUILDER, NEEDLE,
     '\tif m.ID == "adtech/conversion-event-reconciler" { entry.Usability.Availability = "usable-now" }\n' + NEEDLE,
     GO_COMMAND, ROOT, "want declared not-verified/local-tool"),
    ("ignore compatibility module", LOOKUP, "item.module === module && ", "",
     WEB_COMMAND, ROOT / "apps/web", "Received:"),
    ("ignore compatibility identity", LOOKUP, "item.id === id && ", "",
     WEB_COMMAND, ROOT / "apps/web", "Received:"),
    ("ignore compatibility version", LOOKUP, " && item.version === version", "",
     WEB_COMMAND, ROOT / "apps/web", "Received:"),
]


def run(command, cwd):
    return subprocess.run(command, cwd=cwd, env=ENV, stdout=subprocess.PIPE,
                          stderr=subprocess.STDOUT, text=True, timeout=90)


def main():
    # A failing control is an environment/test failure, not a killed mutation.
    for command, cwd in [(GO_COMMAND, ROOT), (WEB_COMMAND, ROOT / "apps/web")]:
        result = run(command, cwd)
        if result.returncode:
            raise SystemExit("Control failed; no mutations applied:\n" + result.stdout)
    for name, source, before, after, command, cwd, signal in CASES:
        original = source.read_bytes()
        before, after = before.encode(), after.encode()
        if original.count(before) != 1:
            raise SystemExit(f"Mutation site is not unique: {name}")
        started = time.monotonic()
        try:
            source.write_bytes(original.replace(before, after, 1))
            result = run(command, cwd)
        finally:
            source.write_bytes(original)
        if result.returncode != 1 or signal not in result.stdout:
            raise SystemExit(f"Surviving mutation or harness failure: {name}\n{result.stdout}")
        print(f"DETECTED: {name} ({time.monotonic() - started:.1f}s)", flush=True)
    print(f"Detected {len(CASES)}/{len(CASES)} selected harmful producer mutations; sources restored.")


if __name__ == "__main__":
    main()
