import { publicationGuideUrl, type PublicationGuide, type PublicationResource } from "@/lib/publicationLinks";

export default function RelatedReading({ guides, module, id }: {
  guides: readonly PublicationGuide[];
  module: PublicationResource["module"];
  id: string;
}) {
  if (!guides.length) return null;
  return (
    <article id="related-reading" className="detail-panel related-reading" aria-labelledby="related-reading-heading">
      <h2 id="related-reading-heading">Related reading</h2>
      <p className="meta">From the AI Knowledge Hub publication</p>
      <ul>
        {guides.map(guide => (
          <li key={guide.slug}>
            <a href={publicationGuideUrl(guide)}>{guide.title}</a>
            <p>{guide.resources.find(resource => resource.module === module && resource.id === id)?.context}</p>
          </li>
        ))}
      </ul>
    </article>
  );
}
