import type { ReactNode } from "react";
import type { RegistryEntry } from "@/lib/registry";
import type { ResourceFirstUseContent } from "@/lib/resourceFirstUse";
import InstructionMarkdown from "@/components/InstructionMarkdown";
import DetailDisclosure from "@/components/DetailDisclosure";

export default function ResourceFirstUse({ entry, content, children }: {
  entry: RegistryEntry;
  content: ResourceFirstUseContent;
  children: ReactNode;
}) {
  const release = entry.versions.find((version) => version.version === entry.latest);
  return (
    <section className="resource-first-use" aria-label="First use">
      {entry.operational?.use_when ? <p className="resource-use-when">{entry.operational.use_when}</p> : null}
      <div className="resource-facts">
        <section><h2>{entry.usability.availability === "template-only" ? "Intended inputs" : "Inputs"}</h2><InstructionMarkdown source={content.inputs} /></section>
        <section><h2>{entry.usability.availability === "template-only" ? "Intended outputs" : "Outputs"}</h2><InstructionMarkdown source={content.outputs} /></section>
      </div>
      <section className="resource-requirements">
        <h2>Before you begin</h2>
        <ul>{content.requirements.map((requirement) => <li key={requirement}>{requirement}</li>)}</ul>
      </section>
      <section className="resource-first-step">
        <h2>Try this first</h2>
        <p>{content.firstStep}</p>
        {entry.deprecated ? <p>This release is deprecated. Check its replacement and migration guidance in the requirements below.</p> : null}
        {content.limits.length ? (
          <details className="first-use-limits" open>
            <summary>Limits for this step</summary>
            <ul>{content.limits.map((limit) => <li key={limit}>{limit}</li>)}</ul>
          </details>
        ) : null}
        {content.sample ? (
          <DetailDisclosure show="Inspect offline sample" hide="Hide offline sample" detail="Synthetic example">
            <p>Runtime · {content.sample.runtime}. Run from the installed package directory after file installation.</p>
            <pre><code>{content.sample.command}</code></pre>
            <p>Expected result · {content.sample.result}</p>
            <p className="meta">This is the release’s documented sample, separate from evidence of operation in your environment.</p>
            {content.exampleInput ? <details className="detail-disclosure-nested"><summary>Sample input</summary><pre><code>{content.exampleInput}</code></pre></details> : null}
            {content.exampleOutput ? <details className="detail-disclosure-nested"><summary>Expected sample output</summary><pre><code>{content.exampleOutput}</code></pre></details> : null}
          </DetailDisclosure>
        ) : children}
      </section>
      {content.unavailablePreviews.length ? <p className="meta">Preview unavailable here · {content.unavailablePreviews.join(", ")}. Open the release package for the full content.</p> : null}
      <p className="meta resource-summary-source">Summary sources · Catalogue metadata{content.sources.length ? `, ${content.sources.join(", ")}` : ""} · v{entry.latest}{release ? <> · <a href={release.artifact_url}>Release package</a></> : null}</p>
    </section>
  );
}
