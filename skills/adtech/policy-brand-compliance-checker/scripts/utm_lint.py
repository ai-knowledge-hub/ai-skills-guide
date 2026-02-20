#!/usr/bin/env python3
import json, sys
from urllib.parse import urlparse, parse_qs

REQUIRED = ["utm_source", "utm_medium", "utm_campaign"]

if __name__ == "__main__":
    payload = json.load(sys.stdin)
    out = []
    for url in payload.get("urls", []):
        qs = parse_qs(urlparse(url).query)
        missing = [k for k in REQUIRED if k not in qs]
        out.append({"url": url, "valid": len(missing) == 0, "missing": missing})
    print(json.dumps({"results": out}, indent=2))
