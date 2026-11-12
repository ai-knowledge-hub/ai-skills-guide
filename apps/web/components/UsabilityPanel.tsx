/** @jsxImportSource react */
import type { RegistryEntry } from "@/lib/registry";
import UsabilityBadge from "@/components/UsabilityBadge";
import { availabilityDescriptions, resourceFormLabels } from "@/lib/cataloguePresentation";

export default function UsabilityPanel({ usability, compatibility, catalogueCompatibility }: { usability: RegistryEntry["usability"]; compatibility?: RegistryEntry["compatibility"]; catalogueCompatibility?: RegistryEntry["catalogueCompatibility"] }) {
  return (
    <article id="usability" className="card detail-panel" aria-labelledby="usability-heading">
      <h2 id="usability-heading">How You Can Use This</h2>
      <p><UsabilityBadge availability={usability.availability} /></p>
      <p><span className="meta">Resource form:</span> {resourceFormLabels[usability.execution]}</p>
      <p>{availabilityDescriptions[usability.availability]}</p>
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
          <p>Each program has its own setup requirements and operational evidence. Reading or copying the instructions does not verify these programs.</p>
          {usability.executable_helpers.map((helper) => (
            <section key={helper.entrypoint}>
              <p><code>{helper.entrypoint}</code> · <UsabilityBadge availability={helper.availability} /> · {resourceFormLabels[helper.execution]}</p>
              {helper.quickstart ? <p><span className="meta">First run:</span> <code>{helper.quickstart}</code></p> : null}
              <ul>{helper.limitations.map((item) => <li key={item}>{item}</li>)}</ul>
            </section>
          ))}
        </>
      ) : null}
      <details className="readiness-explainer">
        <summary>What installation and verification mean</summary>
        <dl>
          <dt>File installation</dt>
          <dd>Places package files and generated configuration in the selected directories.</dd>
          <dt>Native discovery</dt>
          <dd>Your agent app finds and accepts the instructions, tools or plugin registrations.</dd>
          <dt>Mock execution</dt>
          <dd>Exercises sample fixtures. Connected data and provider effects need a separate check.</dd>
          <dt>Connected operation</dt>
          <dd>Requires current configuration, provider authentication, scoped permissions and a successful operation in your intended environment.</dd>
        </dl>
      </details>
      <h3>Declared compatibility</h3>
      {catalogueCompatibility ? (
        <>
          <p className="meta">Maintainer-reviewed information for version {catalogueCompatibility.version} · Reviewed {catalogueCompatibility.reviewed}. This is not a live provider check.</p>
          <ul>{catalogueCompatibility.details.map((detail) => <li key={detail}>{detail}</li>)}</ul>
          {catalogueCompatibility.limitations.length ? <><p className="meta">Limits</p><ul>{catalogueCompatibility.limitations.map((item) => <li key={item}>{item}</li>)}</ul></> : null}
          <p><a href={catalogueCompatibility.source.url}>{catalogueCompatibility.source.label}</a></p>
        </>
      ) : compatibility?.length ? (
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
