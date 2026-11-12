import Link from "next/link";
import { familySites } from "@/lib/brand";

export default function FamilyNavigation({ label, descriptions = false }: { label: string; descriptions?: boolean }) {
  return (
    <nav className={`family-nav${descriptions ? " family-nav--described" : ""}`} aria-label={label}>
      <ul>
        {Object.entries(familySites).map(([key, site]) => (
          <li key={key}>
            {key === "skills" ? (
              <Link href="/" aria-current="true">{site.label}</Link>
            ) : (
              <a href={site.url}>{site.label} <span aria-hidden="true">↗</span></a>
            )}
            {descriptions ? <p>{site.role}</p> : null}
          </li>
        ))}
      </ul>
    </nav>
  );
}
