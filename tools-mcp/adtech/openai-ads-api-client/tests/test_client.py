import copy
import datetime as dt
import importlib.util
import io
import json
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from email.message import Message
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


CLIENT = load_module("openai_ads_client", ROOT / "scripts" / "openai_ads_client.py")
CONVERSIONS = load_module("validate_conversion_events", ROOT / "scripts" / "validate_conversion_events.py")
AUTH = load_module("openai_ads_auth_driver", ROOT / "scripts" / "openai_ads_auth_driver.py")


class Handler(BaseHTTPRequestHandler):
    last_request = None

    def do_GET(self):
        Handler.last_request = {"method": "GET", "path": self.path, "authorization": self.headers.get("Authorization")}
        if self.path == "/redirect":
            self.send_response(302)
            self.send_header("Location", "/ad_account")
            self.end_headers()
            return
        if self.path == "/oversized":
            body = json.dumps({"value": "x" * CLIENT.MAX_RESPONSE_BYTES}).encode()
        else:
            body = json.dumps({"id": "adacct_live_test", "status": "active"}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length))
        Handler.last_request = {"method": "POST", "path": self.path, "body": body}
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"validated": True}).encode())

    def log_message(self, *_):
        return


class FakeResponse(io.BytesIO):
    def __init__(self, body, content_type="application/json"):
        super().__init__(body)
        self.headers = Message()
        self.headers["Content-Type"] = content_type

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.close()


class FakeOpener:
    def __init__(self, response):
        self.response = response
        self.request = None

    def open(self, request, timeout):
        self.request = request
        return self.response


class OpenAIAdsClientTests(unittest.TestCase):
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

    @staticmethod
    def conversion_event(event_type, data_type, now_ms):
        event = {"id": f"{event_type}_1", "type": event_type, "timestamp_ms": now_ms,
                 "data": {"type": data_type}}
        if event_type == "custom":
            event["custom_event_name"] = "product_compared"
        if event_type in {"app_installed", "app_opened"}:
            event["action_source"] = "mobile_app"
        return event

    @classmethod
    def setUpClass(cls):
        cls.server = HTTPServer(("127.0.0.1", 0), Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base_url = f"http://127.0.0.1:{cls.server.server_port}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def test_mock_mode_requires_no_credentials_and_covers_reads(self):
        client = CLIENT.AdsClient(mode="mock")
        self.assertEqual(client.get_account()["id"], "adacct_mock")
        self.assertEqual(client.list_campaigns()["data"][0]["id"], "cmpn_101")
        self.assertEqual(client.list_ads("adgrp_301")["data"][0]["id"], "ad_501")
        self.assertIn("data", client.get_insights("campaign", "cmpn_101"))

    def test_packaged_mock_smoke_runs_without_environment(self):
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "openai_ads_client.py"), "--mode", "mock", "account"],
            cwd=ROOT, env={}, text=True, capture_output=True, check=False,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout)["id"], "adacct_mock")

    def test_live_client_only_sends_get_requests(self):
        client = CLIENT.AdsClient(mode="live", api_key="test-key", base_url=self.base_url, allow_loopback=True)
        self.assertEqual(client.get_account()["id"], "adacct_live_test")
        self.assertEqual(Handler.last_request["method"], "GET")
        self.assertEqual(Handler.last_request["authorization"], "Bearer test-key")

    def test_malformed_advertiser_credential_never_reaches_diagnostics(self):
        secret = "secret-token-value\nsecond-line"
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "openai_ads_client.py"), "--mode", "live", "account"],
            cwd=ROOT, env={"OPENAI_ADS_API_KEY": secret}, text=True, capture_output=True, check=False,
        )
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stderr, "error: OPENAI_ADS_API_KEY is missing or malformed\n")
        self.assertNotIn(secret, result.stderr)
        self.assertNotIn("Traceback", result.stderr)

    def test_untrusted_endpoint_and_unsafe_inputs_fail_closed(self):
        with self.assertRaises(CLIENT.AdsClientError):
            CLIENT.AdsClient(mode="live", api_key="x", base_url="https://example.com/v1")
        client = CLIENT.AdsClient(mode="mock")
        for call in (lambda: client.list_campaigns(0), lambda: client.list_ads("../escape"),
                     lambda: client.get_insights("campaign", "id/escape")):
            with self.assertRaises(CLIENT.AdsClientError):
                call()

    def test_response_bounds_and_shape_are_enforced(self):
        oversized = FakeOpener(FakeResponse(b'{"value":"' + b"x" * CLIENT.MAX_RESPONSE_BYTES + b'"}'))
        with self.assertRaisesRegex(CLIENT.AdsClientError, "size limit"):
            CLIENT.AdsClient(mode="live", api_key="x", opener=oversized).get_account()
        malformed = FakeOpener(FakeResponse(b"not-json"))
        with self.assertRaisesRegex(CLIENT.AdsClientError, "invalid JSON"):
            CLIENT.AdsClient(mode="live", api_key="x", opener=malformed).get_account()

    def test_conversion_remote_mode_forces_validate_only(self):
        now_ms = int(time.time() * 1000)
        payload = {"validate_only": False, "events": [{"id": "x", "type": "lead_created",
                                                          "timestamp_ms": now_ms,
                                                          "data": {"type": "customer_action"}}]}
        response = CONVERSIONS.remote_validate(payload, "px_1", "key", endpoint=self.base_url,
                                               allow_loopback=True)
        self.assertTrue(response["validated"])
        self.assertTrue(Handler.last_request["body"]["validate_only"])
        self.assertIn("pid=px_1", Handler.last_request["path"])

    def test_conversion_payload_and_endpoint_fail_closed(self):
        self.assertEqual(CONVERSIONS.validate_payload([]), ["payload must be an object"])
        payload = {"events": [{"id": str(i), "type": "lead", "timestamp_ms": int(time.time() * 1000)}
                              for i in range(1001)]}
        self.assertIn("events cannot contain more than 1000 items", CONVERSIONS.validate_payload(payload))
        with self.assertRaisesRegex(ValueError, "not trusted"):
            CONVERSIONS.remote_validate({"events": []}, "px", "key", endpoint="https://example.com")

    def test_conversion_schema_rejects_unknown_and_oversized_values(self):
        now_ms = int(time.time() * 1000)
        base = {"id": "x", "type": "lead_created", "timestamp_ms": now_ms,
                "data": {"type": "customer_action"}}
        payload = {"unexpected": True, "events": [{**base, "secret_extra": "x"}]}
        errors = CONVERSIONS.validate_payload(payload, now_ms)
        self.assertIn("payload contains unsupported fields", errors)
        self.assertIn("events[0] contains unsupported fields", errors)
        nested = {"events": [{**base, "user": {"unknown": "x"},
                              "data": {"type": "customer_action", "unknown": "x"}}]}
        nested_errors = CONVERSIONS.validate_payload(nested, now_ms)
        self.assertIn("events[0].user contains unsupported fields", nested_errors)
        self.assertIn("events[0].data contains unsupported fields", nested_errors)
        oversized = {"events": [{**base, "oppref": "x" * 2049}]}
        self.assertIn("events[0].oppref is invalid or exceeds its limit",
                      CONVERSIONS.validate_payload(oversized, now_ms))

    def test_conversion_serialized_request_and_source_file_are_bounded(self):
        now_ms = int(time.time() * 1000)
        events = [
            {"id": f"event_{index}", "type": "lead_created", "timestamp_ms": now_ms,
             "user": {"user_agent": "x" * 1000}, "data": {"type": "customer_action"}}
            for index in range(1000)
        ]
        errors = CONVERSIONS.validate_payload({"events": events}, now_ms)
        self.assertIn("conversion request exceeds the safe serialized boundary", errors)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "oversized.json"
            path.write_bytes(b" " * (CONVERSIONS.MAX_INPUT_BYTES + 1))
            result = subprocess.run(
                [sys.executable, str(ROOT / "scripts" / "validate_conversion_events.py"), str(path)],
                cwd=ROOT, env={}, text=True, capture_output=True, check=False,
            )
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stderr, "error: conversion validation failed\n")

    def test_every_supported_event_requires_its_documented_data_shape(self):
        now_ms = int(time.time() * 1000)
        all_data_types = {"contents", "customer_action", "plan_enrollment", "custom"}
        self.assertEqual(set(self.EVENT_DATA_TYPES), CONVERSIONS.ALLOWED_EVENT_TYPES)
        for event_type, expected_data_type in self.EVENT_DATA_TYPES.items():
            with self.subTest(event_type=event_type, data_type=expected_data_type):
                event = self.conversion_event(event_type, expected_data_type, now_ms)
                self.assertEqual(CONVERSIONS.validate_payload({"events": [event]}, now_ms), [])
            for substituted_data_type in all_data_types - {expected_data_type}:
                with self.subTest(event_type=event_type, substituted_data_type=substituted_data_type):
                    event = self.conversion_event(event_type, substituted_data_type, now_ms)
                    errors = CONVERSIONS.validate_payload({"events": [event]}, now_ms)
                    self.assertIn(
                        f"events[0].data.type must be {expected_data_type} for {event_type}", errors
                    )

    def test_conversion_cross_field_rules_reject_invalid_combinations(self):
        now_ms = int(time.time() * 1000)
        for event_type in ("app_installed", "app_opened"):
            for action_source in (None, "web", "offline"):
                with self.subTest(event_type=event_type, action_source=action_source):
                    event = self.conversion_event(event_type, "customer_action", now_ms)
                    if action_source is None:
                        event.pop("action_source")
                    else:
                        event["action_source"] = action_source
                    errors = CONVERSIONS.validate_payload({"events": [event]}, now_ms)
                    self.assertIn(
                        f"events[0].action_source must be mobile_app for {event_type}", errors
                    )

        custom_event = self.conversion_event("custom", "custom", now_ms)
        custom_event["custom_event_name"] = "ORDER_CREATED"
        self.assertIn(
            "events[0].custom_event_name cannot match a standard event name",
            CONVERSIONS.validate_payload({"events": [custom_event]}, now_ms),
        )

        standard_event = self.conversion_event("order_created", "contents", now_ms)
        standard_event["custom_event_name"] = "extra_name"
        self.assertIn(
            "events[0].custom_event_name is only allowed for custom events",
            CONVERSIONS.validate_payload({"events": [standard_event]}, now_ms),
        )

        shape_mutations = (
            ("customer_action", "contents", []),
            ("customer_action", "plan_id", "plan_1"),
            ("contents", "plan_id", "plan_1"),
        )
        for data_type, field, value in shape_mutations:
            with self.subTest(data_type=data_type, field=field):
                event_type = "lead_created" if data_type == "customer_action" else "order_created"
                event = self.conversion_event(event_type, data_type, now_ms)
                event["data"][field] = copy.deepcopy(value)
                self.assertIn(
                    "events[0].data contains unsupported fields",
                    CONVERSIONS.validate_payload({"events": [event]}, now_ms),
                )

    def test_conversion_nested_provider_formats_and_currency_inheritance(self):
        now_ms = int(time.time() * 1000)
        base = self.conversion_event("lead_created", "customer_action", now_ms)
        invalid_user_values = (
            ("countries", ["12"], "events[0].user.countries contains an invalid value"),
            ("postal_codes", ["94107!"], "events[0].user.postal_codes contains an invalid value"),
            ("android_advertising_id", "x" * 36,
             "events[0].user.android_advertising_id must be a GAID UUID"),
        )
        for field, value, expected_error in invalid_user_values:
            with self.subTest(field=field):
                event = copy.deepcopy(base)
                event["user"] = {field: value}
                self.assertIn(
                    expected_error,
                    CONVERSIONS.validate_payload({"events": [event]}, now_ms),
                )

        valid_user = copy.deepcopy(base)
        valid_user["user"] = {
            "countries": ["US", "gb"],
            "postal_codes": ["94107", "SW1A 1AA", "EC1A-1BB"],
            "android_advertising_id": "38400000-8cf0-11bd-b23e-10b96e40000d",
        }
        self.assertEqual(CONVERSIONS.validate_payload({"events": [valid_user]}, now_ms), [])

        order = self.conversion_event("order_created", "contents", now_ms)
        order["data"]["contents"] = [{"id": "sku_1", "amount": 1299}]
        self.assertIn(
            "events[0].data.contents.currency or event-level currency is required with amount",
            CONVERSIONS.validate_payload({"events": [order]}, now_ms),
        )
        event_currency = copy.deepcopy(order)
        event_currency["data"]["currency"] = "USD"
        self.assertEqual(CONVERSIONS.validate_payload({"events": [event_currency]}, now_ms), [])
        item_currency = copy.deepcopy(order)
        item_currency["data"]["contents"][0]["currency"] = "EUR"
        self.assertEqual(CONVERSIONS.validate_payload({"events": [item_currency]}, now_ms), [])

    def test_malformed_conversion_credential_never_reaches_diagnostics(self):
        secret = "conversion-secret-value\nsecond-line"
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "validate_conversion_events.py"),
             str(ROOT / "examples" / "conversion-events.json"), "--now-ms", "1784937600000",
             "--remote-validate"],
            cwd=ROOT,
            env={"OPENAI_ADS_PIXEL_ID": "pixel_1", "OPENAI_ADS_CONVERSIONS_API_KEY": secret},
            text=True, capture_output=True, check=False,
        )
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stderr, "error: conversion validation failed\n")
        self.assertNotIn(secret, result.stderr)
        self.assertNotIn("Traceback", result.stderr)

    def test_conversion_fixture_is_valid_at_reference_time(self):
        payload = json.loads((ROOT / "examples" / "conversion-events.json").read_text())
        self.assertEqual(CONVERSIONS.validate_payload(payload, 1784937600000), [])

    def test_auth_driver_uses_provider_account_not_request_identity(self):
        class FakeClient:
            def __init__(self, **kwargs):
                self.kwargs = kwargs

            def get_account(self):
                return {"id": "adacct_provider", "status": "active", "name": "Private name"}

        request = {"method": "api-key", "flow": "credential-binding", "requested_scopes": [],
                   "credential_bindings": ["OPENAI_ADS_API_KEY"], "expected_account": "adacct_provider"}
        result = AUTH.run_driver("bootstrap", request, env={"OPENAI_ADS_API_KEY": "secret"},
                                 client_factory=FakeClient)
        self.assertEqual(result["account"], "adacct_provider")
        status = AUTH.run_driver("status", {"method": "api-key"}, env={"OPENAI_ADS_API_KEY": "secret"},
                                 now=lambda: dt.datetime(2026, 9, 21, tzinfo=dt.timezone.utc),
                                 client_factory=FakeClient)
        self.assertEqual(status["principal"], "openai-ads-account:adacct_provider")
        self.assertNotIn("Private name", json.dumps(status))
        self.assertEqual(status["attestation_expires_at"], "2026-09-21T00:05:00Z")

    def test_auth_driver_rejects_expected_account_mismatch(self):
        class FakeClient:
            def __init__(self, **_):
                pass

            def get_account(self):
                return {"id": "actual"}

        request = {"method": "api-key", "flow": "credential-binding", "requested_scopes": [],
                   "credential_bindings": ["OPENAI_ADS_API_KEY"], "expected_account": "other"}
        with self.assertRaisesRegex(ValueError, "does not match"):
            AUTH.run_driver("bootstrap", request, env={"OPENAI_ADS_API_KEY": "secret"},
                            client_factory=FakeClient)


if __name__ == "__main__":
    unittest.main()
