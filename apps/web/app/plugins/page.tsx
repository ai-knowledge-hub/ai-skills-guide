import CatalogueArtwork from "@/components/CatalogueArtwork";
import type { Metadata } from "next";
import { loadPluginsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

const introduction = "Browse installable bundles and reference templates that bring skills, agents, tools and hooks together. Each entry explains what you can install and what still needs implementation, setup or verification.";
export const metadata: Metadata = { title: "Plugins Catalog", description: introduction };

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
      <CatalogueArtwork />
      <header className="catalogue-heading">
        <h1 aria-label="Plugins Catalog">Plugins</h1>
        <p>{introduction}</p>
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
