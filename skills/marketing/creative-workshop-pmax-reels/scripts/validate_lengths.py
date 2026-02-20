#!/usr/bin/env python3
import json, sys

LIMITS = {"pmax_headline": 30, "pmax_description": 90, "reels_primary": 125}

if __name__ == "__main__":
    data = json.load(sys.stdin)
    errors = []
    for key, limit in LIMITS.items():
      text = data.get(key, "")
      if len(text) > limit:
          errors.append({"field": key, "max": limit, "actual": len(text)})
    print(json.dumps({"valid": len(errors) == 0, "errors": errors}, indent=2))
