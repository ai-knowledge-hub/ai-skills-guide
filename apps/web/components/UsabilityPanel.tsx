import type { RegistryEntry } from "@/lib/registry";
import UsabilityBadge from "@/components/UsabilityBadge";

export default function UsabilityPanel({ usability }: { usability: RegistryEntry["usability"] }) {
  return (
    <article className="card detail-panel">
      <h2>How You Can Use This</h2>
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
      <p className="meta">Classification: {usability.source}</p>
    </article>
  );
}
