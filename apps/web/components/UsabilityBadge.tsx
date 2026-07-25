import type { RegistryEntry } from "@/lib/registry";

type Availability = RegistryEntry["usability"]["availability"];

const labels: Record<Availability, string> = {
  "usable-now": "Usable now",
  "setup-required": "Setup required",
  "template-only": "Template only",
  "documentation-only": "Documentation"
};

export default function UsabilityBadge({ availability }: { availability: Availability }) {
  return (
    <span className={`catalog-status-badge usability-badge is-${availability}`}>
      {labels[availability]}
    </span>
  );
}
