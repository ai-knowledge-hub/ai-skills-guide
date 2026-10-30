#!/usr/bin/env python3
"""Provider-verifying readiness driver for an account-scoped OpenAI Ads API key."""

import datetime as dt
import hashlib
import json
import os
import sys

from openai_ads_client import AdsClient, AdsClientError

METHOD = "api-key"
FLOW = "credential-binding"
BINDING = "OPENAI_ADS_API_KEY"
# The provider documents account binding for Ads API keys, not OAuth-style scopes.
SCOPES = []


def _read_request():
    raw = sys.stdin.buffer.read(64 * 1024 + 1)
    if len(raw) > 64 * 1024:
        raise ValueError("request exceeds the safe size limit")
    value = json.loads(raw or b"{}")
    if not isinstance(value, dict):
        raise ValueError("request must be an object")
    return value


def _account_identity(account):
    account_id = account.get("id")
    if not isinstance(account_id, str) or not account_id.strip():
        raise ValueError("provider did not attest an ad account identifier")
    status = account.get("status")
    if status is not None and not isinstance(status, str):
        raise ValueError("provider returned invalid account status")
    return account_id.strip(), status or "unknown"


def run_driver(operation, request, *, env=None, now=None, client_factory=AdsClient):
    env = env if env is not None else os.environ
    now = now or (lambda: dt.datetime.now(dt.timezone.utc))
    if operation == "credential":
        raise ValueError("credential is resolved by a runtime-owned environment binding")
    if operation not in {"bootstrap", "status"}:
        raise ValueError("unsupported authentication driver operation")

    client = client_factory(mode="live", api_key=env.get(BINDING))
    account_id, account_status = _account_identity(client.get_account())
    if operation == "bootstrap":
        if request.get("method") != METHOD or request.get("flow") != FLOW:
            raise ValueError("bootstrap request does not match the packaged authentication contract")
        if request.get("requested_scopes") != SCOPES or request.get("credential_bindings") != [BINDING]:
            raise ValueError("bootstrap request does not match the packaged authentication contract")
        expected = request.get("expected_account")
        if expected and expected != account_id:
            raise ValueError("selected OpenAI Ads account does not match expected account")
        return {"completed": True, "account": account_id, "scopes": SCOPES}

    observed_at = now()
    if observed_at.tzinfo is None:
        observed_at = observed_at.replace(tzinfo=dt.timezone.utc)
    expiry = observed_at.astimezone(dt.timezone.utc) + dt.timedelta(minutes=5)
    commitment = hashlib.sha256(f"{account_id}\0{account_status}".encode()).hexdigest()
    return {
        "authenticated": True,
        "revoked": False,
        "principal": f"openai-ads-account:{account_id}",
        "account": account_id,
        "scopes": SCOPES,
        "provider": "openai-advertiser-api",
        "tier": "live",
        "endpoint": "api.ads.openai.com",
        "region": "global",
        "api_version": "v1",
        "attestation_reference": f"openai-ads:ad-account:{commitment}",
        "attestation_expires_at": expiry.isoformat().replace("+00:00", "Z"),
    }


def main():
    method, operation = (sys.argv[1:] + [None, None])[:2]
    if method != METHOD:
        raise ValueError("unsupported authentication method")
    result = run_driver(operation, _read_request())
    sys.stdout.write(json.dumps(result, separators=(",", ":")) + "\n")


if __name__ == "__main__":
    try:
        main()
    except (AdsClientError, ValueError, json.JSONDecodeError):
        sys.stderr.write('{"code":"AUTH_VERIFICATION_FAILED","message":"OpenAI Ads authentication verification failed"}\n')
        raise SystemExit(1)
