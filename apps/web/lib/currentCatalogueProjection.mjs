// Operational readiness v3: a static publication has no authoritative live
// target check. Opaque evidence references and their dates cannot promote it.
// Shared by the site loader and the constrained Vercel publication fallback.
export function projectCurrentCatalogEntry(storedEntry) {
  const entry = structuredClone(storedEntry);
  const hasPositiveCapability = (entry.capability_readiness ?? [])
    .some((capability) => isPositiveAvailability(capability.availability));
  if (entry.usability?.availability === "usable-now" || hasPositiveCapability) {
    if (entry.usability?.availability === "usable-now") entry.usability.availability = "not-verified";
    entry.usability.source = "inferred";
    for (const capability of entry.capability_readiness ?? []) {
      if (isPositiveAvailability(capability.availability)) capability.availability = "not-verified";
    }
  }
  if (entry.deprecated) {
    entry.readiness = "deprecated";
    let demoted = false;
    if (isPositiveAvailability(entry.usability?.availability)) {
      entry.usability.availability = "not-verified";
      demoted = true;
    }
    for (const helper of entry.usability?.executable_helpers ?? []) {
      if (isPositiveAvailability(helper.availability)) {
        helper.availability = "not-verified";
        demoted = true;
      }
    }
    for (const capability of entry.capability_readiness ?? []) {
      if (isPositiveAvailability(capability.availability)) {
        capability.availability = "not-verified";
        demoted = true;
      }
    }
    if (demoted) entry.usability.source = "inferred";
  }
  return entry;
}

function isPositiveAvailability(availability) {
  return availability === "usable-now" || availability === "setup-required";
}
