import Link from "next/link";

export default function NotFoundPage() {
  return (
    <main>
      <h1>Skill not found</h1>
      <p>We could not find that skill in the current registry snapshot.</p>
      <div className="nav">
        <Link href="/" className="button button--secondary">Home</Link>
        <Link href="/skills" className="button button--secondary">Catalog</Link>
      </div>
    </main>
  );
}
