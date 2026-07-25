import importlib.util
import json
import os
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


CLIENT = load_module("openai_ads_client", ROOT / "scripts" / "openai_ads_client.py")
CONVERSIONS = load_module("validate_conversion_events", ROOT / "scripts" / "validate_conversion_events.py")


class Handler(BaseHTTPRequestHandler):
    last_request = None

    def do_GET(self):
        Handler.last_request = {"method": "GET", "path": self.path, "authorization": self.headers.get("Authorization")}
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"id": "adacct_live_test"}).encode())

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


class OpenAIAdsClientTests(unittest.TestCase):
    def setUp(self):
        self.server = HTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.base_url = f"http://127.0.0.1:{self.server.server_port}"

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def test_mock_mode_requires_no_credentials(self):
        client = CLIENT.AdsClient(mode="mock")
        self.assertEqual(client.get_account()["id"], "adacct_mock")
        self.assertEqual(client.list_campaigns()["data"][0]["id"], "cmpn_101")

    def test_live_client_only_sends_get_requests(self):
        client = CLIENT.AdsClient(mode="live", api_key="test-key", base_url=self.base_url)
        self.assertEqual(client.get_account()["id"], "adacct_live_test")
        self.assertEqual(Handler.last_request["method"], "GET")
        self.assertEqual(Handler.last_request["authorization"], "Bearer test-key")

    def test_conversion_remote_mode_forces_validate_only(self):
        payload = {"validate_only": False, "events": [{"id": "x", "type": "lead", "timestamp_ms": 1}]}
        response = CONVERSIONS.remote_validate(payload, "px_1", "key", endpoint=self.base_url)
        self.assertTrue(response["validated"])
        self.assertTrue(Handler.last_request["body"]["validate_only"])
        self.assertIn("pid=px_1", Handler.last_request["path"])

    def test_conversion_fixture_is_valid_at_reference_time(self):
        payload = json.loads((ROOT / "examples" / "conversion-events.json").read_text())
        self.assertEqual(CONVERSIONS.validate_payload(payload, 1784937600000), [])


if __name__ == "__main__":
    unittest.main()
