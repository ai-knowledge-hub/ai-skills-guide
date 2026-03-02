import Link from "next/link";
import { loadAgentsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";

type SearchParams = {
  q?: string;
  tag?: string;
  category?: string;
  runtime?: string;
};

export default async function AgentsPage({ searchParams }: { searchParams: SearchParams }) {
  const registry = await loadAgentsRegistry();
  const q = searchParams.q ?? "";
  const tag = searchParams.tag ?? "";
  const category = searchParams.category ?? "";
  const runtime = searchParams.runtime ?? "";

  const categories = uniqueValues(registry.skills.map((s) => s.category));
  const tags = uniqueValues(registry.skills.flatMap((s) => s.tags));

  return (
    <main>
      <div className="nav">
        <Link href="/" className="pill">Home</Link>
        <Link href="/skills" className="pill">Skills</Link>
        <Link href="/tools-mcp" className="pill">Tools &amp; MCP</Link>
        <span className="pill">Agents</span>
      </div>

      <h1>Agents Catalog</h1>
      <p>Browse orchestrated agent templates composed from role, memory, skills, and tools.</p>

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={tags}
        basePath="/agents"
        initial={{ q, tag, category, runtime }}
      />
    </main>
  );
}
