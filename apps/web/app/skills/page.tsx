import Link from "next/link";
import { loadSkillsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";

type SearchParams = {
  q?: string;
  tag?: string;
  category?: string;
  runtime?: string;
};

export default async function SkillsPage({ searchParams }: { searchParams: SearchParams }) {
  const registry = await loadSkillsRegistry();
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
        <Link href="/agents" className="pill">Agents</Link>
        <Link href="/tools-mcp" className="pill">Tools &amp; MCP</Link>
        <span className="pill">Catalog</span>
      </div>

      <h1>Skills Catalog</h1>
      <p>Filter by intent, runtime, and category to find install-ready skills.</p>

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={tags}
        basePath="/skills"
        initial={{ q, tag, category, runtime }}
      />
    </main>
  );
}
