#!/usr/bin/env python3
import json
import math
import sys
from statistics import NormalDist

def parse_inputs():
    raw = sys.stdin.read().strip()
    if raw:
        payload = json.loads(raw)
        baseline_rate = float(payload.get("baseline_rate", 0.05))
        minimum_detectable_effect = float(payload.get("minimum_detectable_effect", 0.01))
        confidence_level = float(payload.get("confidence_level", 95))
        power = float(payload.get("power", 80))
        return baseline_rate, minimum_detectable_effect, confidence_level, power

    # Backward-compatible CLI usage: sample_size.py <baseline_rate> <mde>
    baseline_rate = float(sys.argv[1]) if len(sys.argv) > 1 else 0.05
    minimum_detectable_effect = float(sys.argv[2]) if len(sys.argv) > 2 else 0.01
    return baseline_rate, minimum_detectable_effect, 95.0, 80.0


def validate_inputs(baseline_rate, minimum_detectable_effect, confidence_level, power):
    if not (0 < baseline_rate < 1):
        raise ValueError("baseline_rate must be between 0 and 1.")
    if minimum_detectable_effect <= 0:
        raise ValueError("minimum_detectable_effect must be greater than 0.")
    if not (50 <= confidence_level < 100):
        raise ValueError("confidence_level must be in [50, 100).")
    if not (50 <= power < 100):
        raise ValueError("power must be in [50, 100).")


def compute_sample_size(baseline_rate, minimum_detectable_effect, confidence_level, power):
    z_alpha = NormalDist().inv_cdf(1 - (1 - (confidence_level / 100.0)) / 2)
    z_beta = NormalDist().inv_cdf(power / 100.0)
    pooled = baseline_rate * (1 - baseline_rate)
    n_total = 2 * ((z_alpha + z_beta) ** 2) * pooled / (minimum_detectable_effect ** 2)
    return int(math.ceil(n_total))

if __name__ == "__main__":
    try:
        p, mde, confidence, power = parse_inputs()
        validate_inputs(p, mde, confidence, power)
        sample_size = compute_sample_size(p, mde, confidence, power)
        print(
            json.dumps(
                {
                    "baseline_rate": p,
                    "minimum_detectable_effect": mde,
                    "confidence_level": confidence,
                    "power": power,
                    "recommended_total_sample_size": sample_size,
                }
            )
        )
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(1)
