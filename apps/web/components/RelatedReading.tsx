import { familySites } from "@/lib/brand";
import { companionGuideUrl, type CompanionGuide } from "@/lib/publicationLinks";

export default function RelatedReading({ guides }: {
  guides: readonly (CompanionGuide & { context: string })[];
}) {
  if (!guides.length) return null;
  return (
    <article id="related-reading" className="detail-panel related-reading" aria-labelledby="related-reading-heading">
      <h2 id="related-reading-heading">Related reading</h2>
      <p className="meta">Optional context for this resource</p>
      <ul>
        {guides.map(guide => (
          <li key={companionGuideUrl(guide)}>
            <p className="meta">{familySites[guide.site].label}</p>
            <a href={companionGuideUrl(guide)}>{guide.title}</a>
            <p>{guide.context}</p>
          </li>
        ))}
      </ul>
    </article>
  );
}
