import type { Metadata } from "next";
import { loadPluginsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

export const metadata: Metadata = { title: "Plugins Catalog", description: "Installable bundles of skills, agents, tools, hooks and setup guidance." };

type SearchParams = {
  q?: SearchParamValue;
  tag?: SearchParamValue;
  category?: SearchParamValue;
  runtime?: SearchParamValue;
};

export default async function PluginsPage({ searchParams = {} }: { searchParams?: SearchParams }) {
  const registry = await loadPluginsRegistry();
  const q = typeof searchParams.q === "string" ? searchParams.q : "";
  const tags = parseMultiValue(searchParams.tag);
  const categoriesSelected = parseMultiValue(searchParams.category);
  const runtimes = parseMultiValue(searchParams.runtime);

  const categories = uniqueValues(registry.skills.map((s) => getDomainKey(s.category, "/plugins")));
  const availableTags = uniqueValues(registry.skills.flatMap((s) => s.tags));

  return (
    <main id="main" tabIndex={-1} className="catalogue-main">
      <header className="catalogue-heading">
        <h1 aria-label="Plugins Catalog">Plugins</h1>
        <p>Installable bundles of skills, agents, tools, hooks and setup guidance.</p>
      </header>
      <div className="catalogue-content">

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/plugins"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
      </div>
    </main>
  );
}
