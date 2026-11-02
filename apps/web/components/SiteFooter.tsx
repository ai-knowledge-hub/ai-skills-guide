import { brand } from "@/lib/brand";

export default function SiteFooter() {
  return (
    <footer className="site-footer wrap">
      <a className="footer-wordmark" href="/">{brand.name}</a>
      <p>Practical work, shared openly.</p>
      <nav aria-label="Related sites"><a href={brand.publication}>Read the publication</a><a href={brand.repository}>View the source</a></nav>
    </footer>
  );
}
