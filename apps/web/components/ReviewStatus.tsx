import type { RegistryEntry } from "@/lib/registry";
import { formatReadinessLabel } from "@/lib/categoryLabels";

export default function ReviewStatus({ entry }: { entry: RegistryEntry }) {
  return (
    <article className="card detail-panel">
      <h2>Review and lifecycle</h2>
      <p><span className="meta">Review maturity:</span> {formatReadinessLabel(entry.readiness)}</p>
      <p><span className="meta">Security review declaration:</span> {entry.security_reviewed ? "Declared" : "Not declared"}</p>
      <p><span className="meta">Lifecycle:</span> {entry.deprecated ? "Deprecated" : "Active"}</p>
      <p className="meta">These release declarations are separate from evidence that a workflow operates in your environment.</p>
    </article>
  );
}
