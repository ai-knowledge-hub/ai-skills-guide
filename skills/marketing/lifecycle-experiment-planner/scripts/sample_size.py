#!/usr/bin/env python3
import json, math, sys

# Approximate two-proportion sample size with fixed z-scores.
Z_ALPHA = 1.96
Z_BETA = 0.84

if __name__ == "__main__":
    p = float(sys.argv[1]) if len(sys.argv) > 1 else 0.05
    mde = float(sys.argv[2]) if len(sys.argv) > 2 else 0.01
    pooled = p * (1 - p)
    n = 2 * ((Z_ALPHA + Z_BETA) ** 2) * pooled / (mde ** 2)
    print(int(math.ceil(n)))
