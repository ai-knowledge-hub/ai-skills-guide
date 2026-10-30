#!/usr/bin/env python3
"""Validate bounded OpenAI Ads conversion events locally or in validate-only mode."""

import argparse
import ipaddress
import json
import math
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

CAPI_URL = "https://bzr.openai.com/v1/events"
MAX_INPUT_BYTES = 1024 * 1024
MAX_REQUEST_BYTES = 1024 * 1024
MAX_RESPONSE_BYTES = 2 * 1024 * 1024
MAX_ERRORS = 100
MAX_CUSTOM_DEPTH = 4
MAX_CUSTOM_NODES = 1000
ALLOWED_ACTION_SOURCES = {"web", "mobile_app", "offline", "physical_store", "phone_call", "email", "other"}
ALLOWED_EVENT_TYPES = {
    "appointment_scheduled", "checkout_started", "contents_viewed", "custom", "items_added",
    "lead_created", "order_created", "page_viewed", "registration_completed",
    "subscription_created", "trial_started", "app_installed", "app_opened",
}
EVENT_DATA_TYPES = {
    "app_installed": "customer_action",
    "app_opened": "customer_action",
    "appointment_scheduled": "customer_action",
    "checkout_started": "contents",
    "contents_viewed": "contents",
    "custom": "custom",
    "items_added": "contents",
    "lead_created": "customer_action",
    "order_created": "contents",
    "page_viewed": "contents",
    "registration_completed": "customer_action",
    "subscription_created": "plan_enrollment",
    "trial_started": "plan_enrollment",
}
TOP_LEVEL_FIELDS = {"validate_only", "integration_source", "events"}
EVENT_FIELDS = {"id", "type", "timestamp_ms", "custom_event_name", "oppref", "source_url",
                "action_source", "user", "opt_out", "data"}
USER_FIELDS = {"phone_numbers_sha256", "emails_sha256", "external_ids_sha256", "first_names_sha256",
               "last_names_sha256", "regions", "postal_codes", "cities", "countries",
               "android_advertising_id", "obref", "ip_address", "user_agent"}
DATA_FIELDS = {"type", "amount", "currency", "contents", "plan_id"}
DATA_FIELDS_BY_TYPE = {
    "contents": {"type", "amount", "currency", "contents"},
    "customer_action": {"type", "amount", "currency"},
    "plan_enrollment": {"type", "plan_id", "amount", "currency", "contents"},
}
CONTENT_FIELDS = {"id", "group_id", "name", "content_type", "quantity", "amount", "currency", "variant_dict"}
DATA_TYPES = {"contents", "customer_action", "plan_enrollment", "custom"}
HASH_FIELDS = {"phone_numbers_sha256", "emails_sha256", "external_ids_sha256",
               "first_names_sha256", "last_names_sha256"}
HASH = re.compile(r"^[0-9a-f]{64}$")
COUNTRY_CODE = re.compile(r"^[A-Za-z]{2}$")
POSTAL_CODE = re.compile(r"^[A-Za-z0-9 -]{1,32}$")
GAID = re.compile(
    r"^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$"
)
INTEGRATION_SOURCE = re.compile(r"^[A-Za-z0-9](?:[A-Za-z0-9._-]{0,62}[A-Za-z0-9])?$")
CUSTOM_EVENT_NAME = re.compile(r"^[A-Za-z0-9](?:[A-Za-z0-9_-]{0,62}[A-Za-z0-9])?$")
PIXEL_ID = re.compile(r"^[A-Za-z0-9_-]{1,256}$")
SAFE_HEADER_CREDENTIAL = re.compile(r"^[\x21-\x7e]{1,4096}$")


class _RejectRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise RuntimeError("Conversions API redirects are not allowed")


def _append(errors, message):
    if len(errors) < MAX_ERRORS:
        errors.append(message)


def _append_critical(errors, message):
    if message in errors:
        return
    if len(errors) < MAX_ERRORS:
        errors.append(message)
    elif errors:
        errors[-1] = message


def _bounded_string(value, limit, *, allow_empty=False):
    if not isinstance(value, str) or (not allow_empty and not value):
        return False
    if len(value) > limit:
        return False
    return len(value.encode("utf-8")) <= limit


def _validate_string_list(errors, value, prefix, limit, validator=None):
    if not isinstance(value, list) or len(value) > 3:
        _append(errors, f"{prefix} must be an array of at most 3 values")
        return
    for item in value:
        if not _bounded_string(item, limit) or (validator and not validator.fullmatch(item)):
            _append(errors, f"{prefix} contains an invalid value")
            return


def _validate_custom_value(value, errors, prefix, depth=0, nodes=None):
    nodes = nodes if nodes is not None else [0]
    nodes[0] += 1
    if nodes[0] > MAX_CUSTOM_NODES or depth > MAX_CUSTOM_DEPTH:
        _append(errors, f"{prefix} exceeds the custom-data structure limit")
        return
    if value is None or isinstance(value, (bool, int)):
        return
    if isinstance(value, float):
        if not math.isfinite(value):
            _append(errors, f"{prefix} contains a non-finite number")
        return
    if isinstance(value, str):
        if not _bounded_string(value, 4096, allow_empty=True):
            _append(errors, f"{prefix} exceeds the string limit")
        return
    if isinstance(value, list):
        if len(value) > 100:
            _append(errors, f"{prefix} exceeds the array limit")
            return
        for item in value:
            _validate_custom_value(item, errors, prefix, depth + 1, nodes)
        return
    if isinstance(value, dict):
        if len(value) > 100:
            _append(errors, f"{prefix} exceeds the object-field limit")
            return
        for key, item in value.items():
            if not _bounded_string(key, 128):
                _append(errors, f"{prefix} contains an invalid field name")
                return
            _validate_custom_value(item, errors, prefix, depth + 1, nodes)
        return
    _append(errors, f"{prefix} contains an unsupported value type")


def _validate_user(user, errors, prefix):
    if not isinstance(user, dict):
        _append(errors, f"{prefix} must be an object")
        return
    if len(user) > len(USER_FIELDS) or any(field not in USER_FIELDS for field in user):
        _append(errors, f"{prefix} contains unsupported fields")
    for field in HASH_FIELDS:
        if field in user:
            _validate_string_list(errors, user[field], f"{prefix}.{field}", 64, HASH)
    for field, limit in (("regions", 128), ("cities", 128)):
        if field in user:
            _validate_string_list(errors, user[field], f"{prefix}.{field}", limit)
    if "postal_codes" in user:
        _validate_string_list(errors, user["postal_codes"], f"{prefix}.postal_codes", 32, POSTAL_CODE)
    if "countries" in user:
        _validate_string_list(errors, user["countries"], f"{prefix}.countries", 2, COUNTRY_CODE)
    for field, limit in (("android_advertising_id", 36), ("obref", 1024), ("ip_address", 45), ("user_agent", 1024)):
        if field in user and not _bounded_string(user[field], limit):
            _append(errors, f"{prefix}.{field} is invalid or exceeds its limit")
    if "ip_address" in user and _bounded_string(user["ip_address"], 45):
        try:
            ipaddress.ip_address(user["ip_address"])
        except ValueError:
            _append(errors, f"{prefix}.ip_address is invalid")
    if "android_advertising_id" in user and (
        not isinstance(user["android_advertising_id"], str) or
        not GAID.fullmatch(user["android_advertising_id"])
    ):
        _append(errors, f"{prefix}.android_advertising_id must be a GAID UUID")


def _validate_contents(contents, event_currency, errors, prefix):
    if not isinstance(contents, list) or len(contents) > 100:
        _append(errors, f"{prefix} must be an array of at most 100 items")
        return
    for item in contents:
        if not isinstance(item, dict):
            _append(errors, f"{prefix} items must be objects")
            continue
        if len(item) > len(CONTENT_FIELDS) or any(field not in CONTENT_FIELDS for field in item):
            _append(errors, f"{prefix} items contain unsupported fields")
        for field in ("id", "group_id", "name", "content_type"):
            if field in item and not _bounded_string(item[field], 512):
                _append(errors, f"{prefix}.{field} is invalid or exceeds its limit")
        for field in ("quantity", "amount"):
            if field in item and (isinstance(item[field], bool) or not isinstance(item[field], int)):
                _append(errors, f"{prefix}.{field} must be an integer")
        if "amount" in item and "currency" not in item and event_currency is None:
            _append(errors, f"{prefix}.currency or event-level currency is required with amount")
        if "currency" in item and (not isinstance(item["currency"], str) or not re.fullmatch(r"[A-Za-z]{3}", item["currency"])):
            _append(errors, f"{prefix}.currency must be a three-letter code")
        if "variant_dict" in item:
            variants = item["variant_dict"]
            if not isinstance(variants, dict) or len(variants) > 32 or any(
                not _bounded_string(key, 128) or not _bounded_string(value, 128, allow_empty=True)
                for key, value in variants.items()
            ):
                _append(errors, f"{prefix}.variant_dict is invalid or exceeds its limits")


def _validate_data(data, event_type, errors, prefix):
    if not isinstance(data, dict):
        _append(errors, f"{prefix} must be an object")
        return
    if len(data) > 100:
        _append(errors, f"{prefix} exceeds the object-field limit")
        return
    data_type = data.get("type")
    if data_type not in DATA_TYPES:
        _append(errors, f"{prefix}.type is unsupported")
    expected_type = EVENT_DATA_TYPES.get(event_type)
    if expected_type is not None and data_type != expected_type:
        _append(errors, f"{prefix}.type must be {expected_type} for {event_type}")
    unknown = [field for field in data if field not in DATA_FIELDS]
    allowed_fields = DATA_FIELDS_BY_TYPE.get(data_type)
    if allowed_fields is not None and any(field not in allowed_fields for field in data):
        _append(errors, f"{prefix} contains unsupported fields")
    if "amount" in data and (isinstance(data["amount"], bool) or not isinstance(data["amount"], int)):
        _append(errors, f"{prefix}.amount must be an integer")
    if "currency" in data and (not isinstance(data["currency"], str) or not re.fullmatch(r"[A-Za-z]{3}", data["currency"])):
        _append(errors, f"{prefix}.currency must be a three-letter code")
    if "amount" in data and "currency" not in data:
        _append(errors, f"{prefix}.currency is required with amount")
    if "plan_id" in data and not _bounded_string(data["plan_id"], 512):
        _append(errors, f"{prefix}.plan_id is invalid or exceeds its limit")
    if "contents" in data:
        _validate_contents(data["contents"], data.get("currency"), errors, f"{prefix}.contents")
    if data_type == "custom":
        nodes = [0]
        for field in unknown:
            if not _bounded_string(field, 128):
                _append(errors, f"{prefix} contains an invalid custom field name")
                continue
            _validate_custom_value(data[field], errors, f"{prefix}.custom", nodes=nodes)


def _serialized_request(payload):
    body = dict(payload)
    body["validate_only"] = True
    encoded = json.dumps(body, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    if len(encoded) > MAX_REQUEST_BYTES:
        raise ValueError("conversion request exceeds the safe size limit")
    return encoded


def validate_payload(payload, now_ms=None):
    now_ms = int(now_ms if now_ms is not None else time.time() * 1000)
    errors = []
    if not isinstance(payload, dict):
        return ["payload must be an object"]
    if len(payload) > len(TOP_LEVEL_FIELDS) or any(field not in TOP_LEVEL_FIELDS for field in payload):
        _append(errors, "payload contains unsupported fields")
    if "validate_only" in payload and not isinstance(payload["validate_only"], bool):
        _append(errors, "validate_only must be a boolean")
    if "integration_source" in payload and (
        not isinstance(payload["integration_source"], str) or
        not INTEGRATION_SOURCE.fullmatch(payload["integration_source"])
    ):
        _append(errors, "integration_source is invalid")
    events = payload.get("events")
    if not isinstance(events, list) or not events:
        _append(errors, "events must be a non-empty array")
        return errors
    if len(events) > 1000:
        _append(errors, "events cannot contain more than 1000 items")

    for index, event in enumerate(events[:1000]):
        prefix = f"events[{index}]"
        if not isinstance(event, dict):
            _append(errors, f"{prefix} must be an object")
            continue
        if len(event) > len(EVENT_FIELDS) or any(field not in EVENT_FIELDS for field in event):
            _append(errors, f"{prefix} contains unsupported fields")
        if not _bounded_string(event.get("id"), 256):
            _append(errors, f"{prefix}.id is required and limited to 256 bytes")
        event_type = event.get("type")
        if event_type not in ALLOWED_EVENT_TYPES:
            _append(errors, f"{prefix}.type is unsupported")
        timestamp = event.get("timestamp_ms")
        if isinstance(timestamp, bool) or not isinstance(timestamp, int):
            _append(errors, f"{prefix}.timestamp_ms must be an integer")
        else:
            if timestamp < now_ms - 7 * 24 * 60 * 60 * 1000:
                _append(errors, f"{prefix}.timestamp_ms is older than 7 days")
            if timestamp > now_ms + 10 * 60 * 1000:
                _append(errors, f"{prefix}.timestamp_ms is more than 10 minutes in the future")
        action_source = event.get("action_source")
        if action_source is not None and action_source not in ALLOWED_ACTION_SOURCES:
            _append(errors, f"{prefix}.action_source is unsupported")
        if action_source == "web":
            source_url = event.get("source_url")
            if not _bounded_string(source_url, 2048):
                _append(errors, f"{prefix}.source_url is required and bounded for web events")
            else:
                parsed = urllib.parse.urlparse(source_url)
                if parsed.scheme not in {"http", "https"} or not parsed.hostname:
                    _append(errors, f"{prefix}.source_url must be an HTTP(S) URL with a host")
        for field, limit in (("oppref", 2048), ("source_url", 2048)):
            if field in event and not _bounded_string(event[field], limit):
                _append(errors, f"{prefix}.{field} is invalid or exceeds its limit")
        if "opt_out" in event and not isinstance(event["opt_out"], bool):
            _append(errors, f"{prefix}.opt_out must be a boolean")
        custom_name = event.get("custom_event_name")
        if event_type == "custom" and (not isinstance(custom_name, str) or not CUSTOM_EVENT_NAME.fullmatch(custom_name)):
            _append(errors, f"{prefix}.custom_event_name is required and invalid")
        elif event_type == "custom" and custom_name.lower() in ALLOWED_EVENT_TYPES:
            _append(errors, f"{prefix}.custom_event_name cannot match a standard event name")
        elif event_type != "custom" and custom_name is not None:
            _append(errors, f"{prefix}.custom_event_name is only allowed for custom events")
        if event_type in {"app_installed", "app_opened"} and action_source != "mobile_app":
            _append(errors, f"{prefix}.action_source must be mobile_app for {event_type}")
        if "user" in event:
            _validate_user(event["user"], errors, f"{prefix}.user")
        _validate_data(event.get("data"), event_type, errors, f"{prefix}.data")
    if not errors:
        try:
            _serialized_request(payload)
        except (TypeError, ValueError):
            _append_critical(errors, "conversion request exceeds the safe serialized boundary")
    return errors


def _validated_remote_credentials(pixel_id, api_key):
    if not isinstance(pixel_id, str) or not PIXEL_ID.fullmatch(pixel_id):
        raise ValueError("Conversions API credentials are missing or malformed")
    if not isinstance(api_key, str) or not SAFE_HEADER_CREDENTIAL.fullmatch(api_key):
        raise ValueError("Conversions API credentials are missing or malformed")
    return pixel_id, api_key


def remote_validate(payload, pixel_id, api_key, endpoint=CAPI_URL, *, allow_loopback=False, opener=None, timeout=30):
    pixel_id, api_key = _validated_remote_credentials(pixel_id, api_key)
    parsed = urllib.parse.urlparse(endpoint)
    loopback = parsed.scheme == "http" and parsed.hostname in {"127.0.0.1", "::1", "localhost"}
    if endpoint != CAPI_URL and not (allow_loopback and loopback):
        raise ValueError("Conversions API endpoint is not trusted")
    if not 1 <= timeout <= 120:
        raise ValueError("timeout must be between 1 and 120 seconds")
    if validate_payload(payload):
        raise ValueError("conversion payload is invalid")
    encoded = _serialized_request(payload)
    url = f"{endpoint}?{urllib.parse.urlencode({'pid': pixel_id})}"
    request = urllib.request.Request(
        url, method="POST", data=encoded,
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
    )
    try:
        transport = opener or urllib.request.build_opener(_RejectRedirects())
        with transport.open(request, timeout=timeout) as response:
            if response.headers.get_content_type() != "application/json":
                raise RuntimeError("Conversions API returned an unsupported content type")
            raw = response.read(MAX_RESPONSE_BYTES + 1)
            if len(raw) > MAX_RESPONSE_BYTES:
                raise RuntimeError("Conversions API response exceeds the safe size limit")
            try:
                result = json.loads(raw.decode("utf-8"))
            except (UnicodeDecodeError, json.JSONDecodeError) as exc:
                raise RuntimeError("Conversions API returned invalid JSON") from exc
            if not isinstance(result, dict):
                raise RuntimeError("Conversions API returned an invalid response shape")
            return result
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"Conversions API returned HTTP {exc.code}") from exc
    except (urllib.error.URLError, TimeoutError, OSError) as exc:
        raise RuntimeError("Conversions API request failed") from exc


def _load_payload(path):
    try:
        size = path.stat().st_size
        if size > MAX_INPUT_BYTES:
            raise ValueError("conversion input exceeds the safe size limit")
        with path.open("rb") as source:
            raw = source.read(MAX_INPUT_BYTES + 1)
        if len(raw) > MAX_INPUT_BYTES:
            raise ValueError("conversion input exceeds the safe size limit")
        value = json.loads(raw.decode("utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ValueError("conversion input is unavailable or invalid") from exc
    return value


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("--now-ms", type=int)
    parser.add_argument("--remote-validate", action="store_true")
    parser.add_argument("--pixel-id-env", default="OPENAI_ADS_PIXEL_ID")
    parser.add_argument("--api-key-env", default="OPENAI_ADS_CONVERSIONS_API_KEY")
    args = parser.parse_args(argv)

    try:
        payload = _load_payload(args.input)
        errors = validate_payload(payload, args.now_ms)
        events = payload.get("events", []) if isinstance(payload, dict) else []
        result = {"valid": not errors, "errors": errors, "event_count": len(events), "validate_only": True}
        if errors:
            print(json.dumps(result, indent=2, sort_keys=True))
            return 1
        if args.remote_validate:
            result["provider_response"] = remote_validate(
                payload, os.environ.get(args.pixel_id_env), os.environ.get(args.api_key_env),
            )
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0
    except (ValueError, RuntimeError):
        print("error: conversion validation failed", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
