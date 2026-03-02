import { notFound } from "next/navigation";
import Link from "next/link";
import { getEntryById, loadToolsRegistry } from "@/lib/registry";

export async function generateStaticParams() {
  const registry = await loadToolsRegistry();
  return registry.skills.map((entry) => ({ id: entry.id.split("/") }));
}

export default async function ToolDetailPage({ params }: { params: { id: string[] } }) {
  const entryId = params.id.join("/");
  const entry = await getEntryById("tools", entryId);
  if (!entry) {
    notFound();
  }
  const latestVersion = entry.versions.find((v) => v.version === entry.latest) ?? entry.versions[0];

  return (
    <main>
      <div className="nav">
        <Link href="/" className="pill">Home</Link>
        <Link href="/tools-mcp" className="pill">Tools &amp; MCP</Link>
        <span className="pill">{entry.id}</span>
      </div>

      <article className="card">
        <p className="meta">{entry.category}</p>
        <h1>{entry.name}</h1>
        <p>{entry.description}</p>
        <div className="tags">
          {entry.tags.map((tag) => (
            <span key={tag} className="tag">{tag}</span>
          ))}
        </div>
      </article>

      <section className="detail-grid">
        <article className="card detail-panel">
          <h2>Metadata</h2>
          <p><span className="meta">ID:</span> {entry.id}</p>
          <p><span className="meta">Latest:</span> {entry.latest}</p>
          <p><span className="meta">Runtimes:</span> {entry.runtimes.join(", ")}</p>
          <p><span className="meta">Deprecated:</span> {String(entry.deprecated)}</p>
          {entry.replaced_by ? <p><span className="meta">Replaced by:</span> {entry.replaced_by}</p> : null}
        </article>
        <article className="card detail-panel">
          <h2>Latest Version</h2>
          <p><span className="meta">Released:</span> {latestVersion?.released_at.slice(0, 10) ?? "n/a"}</p>
          <p>
            <span className="meta">Manifest:</span>{" "}
            {latestVersion ? <a href={latestVersion.manifest_url}>{latestVersion.manifest_url}</a> : "n/a"}
          </p>
          <p>
            <span className="meta">Artifact:</span>{" "}
            {latestVersion ? <a href={latestVersion.artifact_url}>{latestVersion.artifact_url}</a> : "n/a"}
          </p>
        </article>
      </section>
    </main>
  );
}
