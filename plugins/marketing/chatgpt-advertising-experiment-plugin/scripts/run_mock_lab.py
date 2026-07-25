#!/usr/bin/env python3
"""Build a deterministic ChatGPT Ads mock evidence bundle without network access."""

import json
import subprocess
import sys
from pathlib import Path

PLUGIN_DIR = Path(__file__).resolve().parents[1]
RUNTIME_ROOT = PLUGIN_DIR.parents[2]
TOOLS_ROOT = RUNTIME_ROOT / "tools-mcp" / "adtech"
CLIENT = TOOLS_ROOT / "openai-ads-api-client" / "scripts" / "openai_ads_client.py"
VALIDATOR = TOOLS_ROOT / "openai-ads-api-client" / "scripts" / "validate_conversion_events.py"
CONVERSION_FIXTURE = TOOLS_ROOT / "openai-ads-api-client" / "examples" / "conversion-events.json"
RECONCILER = TOOLS_ROOT / "conversion-event-reconciler" / "scripts" / "reconcile_events.py"
RECONCILIATION_INPUT = PLUGIN_DIR / "examples" / "reconciliation-input.json"
EXPERIMENT = PLUGIN_DIR / "examples" / "experiment.json"
OUTPUT = PLUGIN_DIR / "output" / "mock-evidence-bundle.json"
REFERENCE_NOW_MS = "1784937600000"


def run_json(*args):
    process = subprocess.run(args, check=True, capture_output=True, text=True)
    return json.loads(process.stdout)


def require(path):
    if not path.exists():
        raise SystemExit(f"missing installed dependency: {path}")


def main():
    for path in (CLIENT, VALIDATOR, CONVERSION_FIXTURE, RECONCILER, RECONCILIATION_INPUT, EXPERIMENT):
        require(path)

    bundle = {
        "experiment": json.loads(EXPERIMENT.read_text()),
        "platform": {
            "account": run_json(sys.executable, str(CLIENT), "--mode", "mock", "account"),
            "campaigns": run_json(sys.executable, str(CLIENT), "--mode", "mock", "campaigns"),
            "insights": run_json(sys.executable, str(CLIENT), "--mode", "mock", "insights", "--scope", "campaign", "--entity-id", "cmpn_101"),
        },
        "conversion_validation": run_json(sys.executable, str(VALIDATOR), str(CONVERSION_FIXTURE), "--now-ms", REFERENCE_NOW_MS),
        "reconciliation": run_json(sys.executable, str(RECONCILER), str(RECONCILIATION_INPUT)),
        "network_calls": 0,
        "next_step": "Review this bundle with adtech/chatgpt-ads-experiment-supervisor in mock-run mode."
    }
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(json.dumps(bundle, indent=2, sort_keys=True) + "\n")
    print(OUTPUT)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
