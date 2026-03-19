import { loadAgentsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";
import { parseMultiValue, type SearchParamValue } from "@/lib/catalogFilters";

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

  const categories = uniqueValues(registry.skills.map((s) => s.category));
  const availableTags = uniqueValues(registry.skills.flatMap((s) => s.tags));

  return (
    <main>
      <div className="nav">
        <a href="/" className="pill">Home</a>
        <a href="/skills" className="pill">Skills</a>
        <a href="/tools-mcp" className="pill">Tools &amp; MCP</a>
        <span className="pill">Agents</span>
      </div>

      <h1>Agents Catalog</h1>
      <p>Browse orchestrated agent templates composed from role, memory, skills, and tools.</p>

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={availableTags}
        basePath="/agents"
        initial={{ q, tags, categories: categoriesSelected, runtimes }}
      />
    </main>
  );
}
