#!/usr/bin/env python3
import hashlib
import json
import sys
from pathlib import Path


def event_key(event):
    event_id = event.get("event_id")
    if event_id:
        return f'{event.get("event_name", "unknown")}:{event_id}'
    raw = json.dumps(event, sort_keys=True, separators=(",", ":"))
    return "unmatched:" + hashlib.sha256(raw.encode()).hexdigest()[:16]


def reconcile(payload):
    canonical = {}
    warnings = []
    raw_events = []
    for source_name in ("browser_events", "server_events"):
        for event in payload.get(source_name, []):
            item = dict(event)
            item["_source"] = source_name
            raw_events.append(item)
            key = event_key(item)
            if not item.get("event_id"):
                warnings.append({"code": "missing_event_id", "source": source_name, "key": key})
            if key not in canonical:
                canonical[key] = {
                    "key": key,
                    "event_name": item.get("event_name"),
                    "event_id": item.get("event_id"),
                    "order_id": item.get("order_id"),
                    "value": item.get("value"),
                    "currency": item.get("currency"),
                    "sources": [source_name],
                    "conflicts": [],
                }
                continue
            current = canonical[key]
            current["sources"].append(source_name)
            for field in ("order_id", "value", "currency"):
                if item.get(field) is not None and current.get(field) != item.get(field):
                    current["conflicts"].append({
                        "field": field,
                        "kept": current.get(field),
                        "received": item.get(field),
                        "source": source_name,
                    })

    outcomes = {item.get("order_id"): item for item in payload.get("outcomes", []) if item.get("order_id")}
    verified = 0
    reversed_count = 0
    for event in canonical.values():
        outcome = outcomes.get(event.get("order_id"))
        event["outcome"] = outcome
        if outcome and outcome.get("status") in {"settled", "qualified", "active"}:
            verified += 1
        if outcome and outcome.get("status") in {"refunded", "cancelled", "disputed"}:
            reversed_count += 1
        if event["conflicts"]:
            warnings.append({"code": "event_conflict", "key": event["key"], "fields": [c["field"] for c in event["conflicts"]]})

    duplicate_count = sum(max(0, len(item["sources"]) - 1) for item in canonical.values())
    return {
        "summary": {
            "raw_event_count": len(raw_events),
            "canonical_event_count": len(canonical),
            "duplicate_event_count": duplicate_count,
            "verified_outcome_count": verified,
            "reversed_outcome_count": reversed_count,
        },
        "canonical_events": list(canonical.values()),
        "warnings": warnings,
    }


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: reconcile_events.py INPUT.json")
    payload = json.loads(Path(sys.argv[1]).read_text())
    print(json.dumps(reconcile(payload), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
