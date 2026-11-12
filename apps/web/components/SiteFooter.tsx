import { brand } from "@/lib/brand";
import FamilyNavigation from "@/components/FamilyNavigation";

export default function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="footer-inner wrap">
        <div className="footer-identity">
          <a className="footer-wordmark" href="/">{brand.name} <span>{brand.siteName}</span></a>
          <a className="text-link" href={brand.repository}>View the source <span aria-hidden="true">↗</span></a>
        </div>
        <FamilyNavigation label="Explore AI Knowledge Hub" descriptions />
      </div>
    </footer>
  );
}
