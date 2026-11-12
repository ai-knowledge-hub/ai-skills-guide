import type { ReactNode } from "react";

export default function InstallationDetails({ title, children }: { title: string; children: ReactNode }) {
  return (
    <details className="resource-install" id="installation">
      <summary><h2>{title}</h2></summary>
      <div className="detail-install-lead">{children}</div>
    </details>
  );
}
