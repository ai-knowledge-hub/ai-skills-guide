import { loadToolsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

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
    <main>
      <div className="nav">
        <a href="/" className="pill">Home</a>
        <a href="/skills" className="pill">Skills</a>
        <a href="/agents" className="pill">Agents</a>
        <span className="pill">Tools &amp; MCP</span>
      </div>

      <h1>Tools &amp; MCP Catalog</h1>
      <p>Browse tool integrations and MCP adapters used by skills and agent templates.</p>

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/tools-mcp"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
    </main>
  );
}
