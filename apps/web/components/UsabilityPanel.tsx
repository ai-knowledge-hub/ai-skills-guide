import type { RegistryEntry } from "@/lib/registry";
import UsabilityBadge from "@/components/UsabilityBadge";

export default function UsabilityPanel({ usability, compatibility }: { usability: RegistryEntry["usability"]; compatibility?: RegistryEntry["compatibility"] }) {
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
      {usability.executable_helpers?.length ? (
        <>
          <h3>Packaged executable helpers</h3>
          {usability.executable_helpers.map((helper) => (
            <section key={helper.entrypoint}>
              <p><code>{helper.entrypoint}</code> · <UsabilityBadge availability={helper.availability} /> · {helper.execution}</p>
              {helper.quickstart ? <p><span className="meta">First run:</span> <code>{helper.quickstart}</code></p> : null}
              <ul>{helper.limitations.map((item) => <li key={item}>{item}</li>)}</ul>
            </section>
          ))}
        </>
      ) : null}
      <h3>Declared compatibility</h3>
      {compatibility?.length ? (
        <><p className="meta">Maintainer support declarations. Test coverage and live authentication are evaluated separately.</p>
          <ul>{compatibility.map((requirement, index) => (
            <li key={`${requirement.kind}:${requirement.name}:${index}`}>
              <strong>{requirement.name}</strong> ({requirement.kind})
              {requirement.versions?.length ? ` · versions: ${requirement.versions.join(", ")}` : ""}
              {requirement.capabilities?.length ? ` · capabilities: ${requirement.capabilities.join(", ")}` : ""}
              {requirement.notes ? <p>{requirement.notes}</p> : null}
            </li>
          ))}</ul></>
      ) : <p>Compatibility unknown: versions and capabilities have not been declared.</p>}
      <p className="meta">Classification: {usability.source}</p>
    </article>
  );
}
