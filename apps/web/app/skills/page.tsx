import CatalogueArtwork from "@/components/CatalogueArtwork";
import type { Metadata } from "next";
import { loadSkillsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

export const metadata: Metadata = { title: "Skills Catalog", description: "Filter instruction packages by intent, runtime, and category, then inspect any independently classified executable helpers." };

type SearchParams = {
  q?: SearchParamValue;
  tag?: SearchParamValue;
  category?: SearchParamValue;
  runtime?: SearchParamValue;
};

export default async function SkillsPage({ searchParams = {} }: { searchParams?: SearchParams }) {
  const registry = await loadSkillsRegistry();
  const q = typeof searchParams.q === "string" ? searchParams.q : "";
  const tags = parseMultiValue(searchParams.tag);
  const categoriesSelected = parseMultiValue(searchParams.category);
  const runtimes = parseMultiValue(searchParams.runtime);

  const categories = uniqueValues(registry.skills.map((s) => getDomainKey(s.category, "/skills")));
  const availableTags = uniqueValues(registry.skills.flatMap((s) => s.tags));

  return (
    <main id="main" tabIndex={-1} className="catalogue-main">
      <CatalogueArtwork />
      <header className="catalogue-heading">
        <h1 aria-label="Skills Catalog">Skills</h1>
        <p>Filter instruction packages by intent, runtime, and category, then inspect any independently classified executable helpers.</p>
      </header>
      <div className="catalogue-content">

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/skills"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
      </div>
    </main>
  );
}
