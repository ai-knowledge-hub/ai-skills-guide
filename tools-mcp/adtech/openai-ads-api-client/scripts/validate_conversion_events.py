#!/usr/bin/env python3
"""Validate OpenAI Ads conversion events locally or via remote validate-only mode."""

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

CAPI_URL = "https://bzr.openai.com/v1/events"
ALLOWED_ACTION_SOURCES = {"web", "mobile_app", "offline", "physical_store", "phone_call", "email", "other"}


def validate_payload(payload, now_ms=None):
    now_ms = int(now_ms if now_ms is not None else time.time() * 1000)
    errors = []
    events = payload.get("events")
    if not isinstance(events, list) or not events:
        return ["events must be a non-empty array"]
    if len(events) > 1000:
        errors.append("events cannot contain more than 1000 items")

    for index, event in enumerate(events):
        prefix = f"events[{index}]"
        if not isinstance(event, dict):
            errors.append(f"{prefix} must be an object")
            continue
        for field in ("id", "type", "timestamp_ms"):
            if event.get(field) in (None, ""):
                errors.append(f"{prefix}.{field} is required")
        timestamp = event.get("timestamp_ms")
        if isinstance(timestamp, int):
            if timestamp < now_ms - 7 * 24 * 60 * 60 * 1000:
                errors.append(f"{prefix}.timestamp_ms is older than 7 days")
            if timestamp > now_ms + 10 * 60 * 1000:
                errors.append(f"{prefix}.timestamp_ms is more than 10 minutes in the future")
        elif timestamp is not None:
            errors.append(f"{prefix}.timestamp_ms must be an integer")
        action_source = event.get("action_source")
        if action_source and action_source not in ALLOWED_ACTION_SOURCES:
            errors.append(f"{prefix}.action_source is unsupported")
        if action_source == "web" and not event.get("source_url"):
            errors.append(f"{prefix}.source_url is required for web events")
        if event.get("type") == "custom" and not event.get("custom_event_name"):
            errors.append(f"{prefix}.custom_event_name is required for custom events")
    return errors


def remote_validate(payload, pixel_id, api_key, endpoint=CAPI_URL):
    if not pixel_id or not api_key:
        raise ValueError("pixel ID and Conversions API key are required for remote validation")
    body = dict(payload)
    body["validate_only"] = True
    url = f"{endpoint}?{urllib.parse.urlencode({'pid': pixel_id})}"
    request = urllib.request.Request(
        url,
        method="POST",
        data=json.dumps(body).encode("utf-8"),
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"Conversions API returned HTTP {exc.code}: {detail}") from exc


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("--now-ms", type=int)
    parser.add_argument("--remote-validate", action="store_true")
    parser.add_argument("--pixel-id-env", default="OPENAI_ADS_PIXEL_ID")
    parser.add_argument("--api-key-env", default="OPENAI_ADS_CONVERSIONS_API_KEY")
    args = parser.parse_args(argv)

    payload = json.loads(args.input.read_text())
    errors = validate_payload(payload, args.now_ms)
    result = {"valid": not errors, "errors": errors, "event_count": len(payload.get("events", [])), "validate_only": True}
    if errors:
        print(json.dumps(result, indent=2, sort_keys=True))
        return 1
    if args.remote_validate:
        result["provider_response"] = remote_validate(
            payload,
            os.environ.get(args.pixel_id_env),
            os.environ.get(args.api_key_env),
        )
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
