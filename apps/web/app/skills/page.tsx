import { loadSkillsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";
import { getDomainKey } from "@/lib/categoryLabels";

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
    <main>
      <div className="nav">
        <a href="/" className="pill">Home</a>
        <a href="/agents" className="pill">Agents</a>
        <a href="/plugins" className="pill">Plugins</a>
        <a href="/tools-mcp" className="pill">Tools &amp; MCP</a>
        <span className="pill">Catalog</span>
      </div>

      <h1>Skills Catalog</h1>
      <p>Filter by intent, runtime, and category to find install-ready skills.</p>

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/skills"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
    </main>
  );
}
