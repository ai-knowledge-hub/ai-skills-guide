import assert from "node:assert/strict";
import test from "node:test";

import { projectCurrentCatalogEntry } from "./prepare-public-assets.mjs";

test("positive capabilities expire independently of aggregate usability", () => {
  const projected = projectCurrentCatalogEntry({
    id: "marketing/experiment-plugin",
    deprecated: false,
    usability: {
      availability: "setup-required",
      execution: "bundle",
      source: "declared"
    },
    execution: { kind: "bundle" },
    authentication: { status: "none" },
    verification: {
      evidence: ["evidence://tests/mock"],
      last_verified_at: "2020-01-01T00:00:00Z"
    },
    capability_readiness: [
      { id: "mock", availability: "usable-now" },
      { id: "live-read", availability: "setup-required" },
      { id: "live-write", availability: "not-verified" }
    ]
  });

  assert.equal(projected.usability.availability, "setup-required");
  assert.equal(projected.usability.source, "inferred");
  assert.deepEqual(
    projected.capability_readiness.map((capability) => capability.availability),
    ["not-verified", "not-verified", "not-verified"]
  );
});
