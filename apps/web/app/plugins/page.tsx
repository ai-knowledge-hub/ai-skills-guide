import { loadPluginsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

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
    <main>
      <div className="nav">
        <a href="/" className="pill">Home</a>
        <a href="/skills" className="pill">Skills</a>
        <a href="/agents" className="pill">Agents</a>
        <a href="/tools-mcp" className="pill">Tools &amp; MCP</a>
        <span className="pill">Plugins</span>
      </div>

      <h1>Plugins Catalog</h1>
      <p>Browse installable bundles that package skills, agents, tools, hooks, and setup guidance into portable team capabilities.</p>

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/plugins"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
    </main>
  );
}
