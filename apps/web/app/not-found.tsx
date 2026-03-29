export default function NotFoundPage() {
  return (
    <main>
      <h1>Item not found</h1>
      <p>We could not find that entry in the current registry snapshot.</p>
      <div className="nav">
        <a href="/" className="button button--secondary">Home</a>
        <a href="/skills" className="button button--secondary">Skills</a>
        <a href="/agents" className="button button--secondary">Agents</a>
        <a href="/plugins" className="button button--secondary">Plugins</a>
        <a href="/tools-mcp" className="button button--secondary">Tools &amp; MCP</a>
      </div>
    </main>
  );
}
