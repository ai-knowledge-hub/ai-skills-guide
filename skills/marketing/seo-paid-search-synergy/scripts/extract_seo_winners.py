#!/usr/bin/env python3
import json, sys

if __name__ == "__main__":
    data = json.load(sys.stdin)
    rows = data.get("queries", [])
    winners = [r for r in rows if float(r.get("ctr", 0)) >= 0.08 and int(r.get("clicks", 0)) >= 100]
    print(json.dumps({"winners": winners}, indent=2))
