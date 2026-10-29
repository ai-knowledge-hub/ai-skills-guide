import type { Metadata } from "next";
import { loadAgentsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

export const metadata: Metadata = { title: "Agents Catalog", description: "Orchestrated agent templates composed from roles, memory, skills and tools." };

type SearchParams = {
  q?: SearchParamValue;
  tag?: SearchParamValue;
  category?: SearchParamValue;
  runtime?: SearchParamValue;
};

export default async function AgentsPage({ searchParams = {} }: { searchParams?: SearchParams }) {
  const registry = await loadAgentsRegistry();
  const q = typeof searchParams.q === "string" ? searchParams.q : "";
  const tags = parseMultiValue(searchParams.tag);
  const categoriesSelected = parseMultiValue(searchParams.category);
  const runtimes = parseMultiValue(searchParams.runtime);

  const categories = uniqueValues(registry.skills.map((s) => getDomainKey(s.category, "/agents")));
  const availableTags = uniqueValues(registry.skills.flatMap((s) => s.tags));

  return (
    <main id="main" tabIndex={-1} className="catalogue-main">
      <header className="catalogue-heading">
        <h1 aria-label="Agents Catalog">Agents</h1>
        <p>Orchestrated agent templates composed from roles, memory, skills and tools.</p>
      </header>
      <div className="catalogue-content">

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/agents"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
      </div>
    </main>
  );
}
