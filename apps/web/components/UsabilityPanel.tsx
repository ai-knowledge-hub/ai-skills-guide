import type { RegistryEntry } from "@/lib/registry";
import UsabilityBadge from "@/components/UsabilityBadge";

export default function UsabilityPanel({ usability }: { usability: RegistryEntry["usability"] }) {
  return (
    <article id="usability" className="card detail-panel" aria-labelledby="usability-heading">
      <h2 id="usability-heading">How You Can Use This</h2>
      <p><UsabilityBadge availability={usability.availability} /></p>
      <p><span className="meta">Execution:</span> {usability.execution}</p>
      {usability.quickstart ? <p><span className="meta">First run:</span> <code>{usability.quickstart}</code></p> : null}
      {usability.requires_setup?.length ? (
        <><p className="meta">Setup required</p><ul>{usability.requires_setup.map((item) => <li key={item}>{item}</li>)}</ul></>
      ) : null}
      {usability.limitations?.length ? (
        <><p className="meta">Current limits</p><ul>{usability.limitations.map((item) => <li key={item}>{item}</li>)}</ul></>
      ) : null}
      <p className="meta">Classification: {usability.source}</p>
    </article>
  );
}
