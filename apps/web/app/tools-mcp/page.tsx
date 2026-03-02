import Link from "next/link";
import { loadToolsRegistry, uniqueValues } from "@/lib/registry";
import CatalogClient from "@/components/CatalogClient";

type SearchParams = {
  q?: string;
  tag?: string;
  category?: string;
  runtime?: string;
};

export default async function ToolsMcpPage({ searchParams }: { searchParams: SearchParams }) {
  const registry = await loadToolsRegistry();
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
        <Link href="/agents" className="pill">Agents</Link>
        <span className="pill">Tools &amp; MCP</span>
      </div>

      <h1>Tools &amp; MCP Catalog</h1>
      <p>Browse tool integrations and MCP adapters used by skills and agent templates.</p>

      <CatalogClient
        entries={registry.skills}
        categories={categories}
        tags={tags}
        basePath="/tools-mcp"
        initial={{ q, tag, category, runtime }}
      />
    </main>
  );
}
