import type { ReactNode } from "react";

export default function DetailDisclosure({ show, hide, detail, children }: {
  show: string;
  hide: string;
  detail: string;
  children: ReactNode;
}) {
  return (
    <details className="detail-disclosure">
      <summary>
        <span className="detail-disclosure-label">
          <span className="detail-disclosure-show">{show}</span>
          <span className="detail-disclosure-hide">{hide}</span>
        </span>
        <span className="meta detail-disclosure-meta">{detail}</span>
      </summary>
      <div className="detail-disclosure-body">{children}</div>
    </details>
  );
}
