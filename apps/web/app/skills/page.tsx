import Link from "next/link";
import { loadRegistry, uniqueValues } from "@/lib/registry";
import SkillsCatalogClient from "@/components/SkillsCatalogClient";

type SearchParams = {
  q?: string;
  tag?: string;
  category?: string;
  runtime?: string;
};

export default async function SkillsPage({ searchParams }: { searchParams: SearchParams }) {
  const registry = await loadRegistry();
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
        <span className="pill">Catalog</span>
      </div>

      <h1>Skills Catalog</h1>
      <p>Filter by intent, runtime, and category to find install-ready skills.</p>

      <SkillsCatalogClient
        skills={registry.skills}
        categories={categories}
        tags={tags}
        initial={{ q, tag, category, runtime }}
      />
    </main>
  );
}
