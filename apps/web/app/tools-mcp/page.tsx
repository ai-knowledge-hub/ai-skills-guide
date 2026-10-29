import type { Metadata } from "next";
import { loadToolsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

export const metadata: Metadata = { title: "Tools & MCP Catalog", description: "Tool integrations and MCP adapters used by skills and agent templates." };

type SearchParams = {
  q?: SearchParamValue;
  tag?: SearchParamValue;
  category?: SearchParamValue;
  runtime?: SearchParamValue;
};

export default async function ToolsMcpPage({ searchParams = {} }: { searchParams?: SearchParams }) {
  const registry = await loadToolsRegistry();
  const q = typeof searchParams.q === "string" ? searchParams.q : "";
  const tags = parseMultiValue(searchParams.tag);
  const categoriesSelected = parseMultiValue(searchParams.category);
  const runtimes = parseMultiValue(searchParams.runtime);

  const categories = uniqueValues(registry.skills.map((s) => getDomainKey(s.category, "/tools-mcp")));
  const availableTags = uniqueValues(registry.skills.flatMap((s) => s.tags));

  return (
    <main id="main" tabIndex={-1} className="catalogue-main">
      <header className="catalogue-heading">
        <h1 aria-label="Tools & MCP Catalog">Tools &amp; MCP</h1>
        <p>Tool integrations and MCP adapters used by skills and agent templates.</p>
      </header>
      <div className="catalogue-content">

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/tools-mcp"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
      </div>
    </main>
  );
}
