#!/usr/bin/env python3
import re, sys

sql = sys.stdin.read().lower()
blocked = [r"\bdrop\b", r"\bdelete\b", r"\btruncate\b", r"\bupdate\b", r"\binsert\b"]
violations = [p for p in blocked if re.search(p, sql)]
if violations:
    print("unsafe")
    sys.exit(1)
print("safe")
