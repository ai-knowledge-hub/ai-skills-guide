import Link from "next/link";

export default function NotFoundPage() {
  return (
    <main>
      <h1>Item not found</h1>
      <p>We could not find that entry in the current registry snapshot.</p>
      <div className="nav">
        <Link href="/" className="button button--secondary">Home</Link>
        <Link href="/skills" className="button button--secondary">Skills</Link>
        <Link href="/agents" className="button button--secondary">Agents</Link>
        <Link href="/tools-mcp" className="button button--secondary">Tools &amp; MCP</Link>
      </div>
    </main>
  );
}
