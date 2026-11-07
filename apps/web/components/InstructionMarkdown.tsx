import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

export default function InstructionMarkdown({ source }: { source: string }) {
  return (
    <Markdown
      skipHtml
      remarkPlugins={[remarkGfm]}
      components={{
        h1: ({ children }) => <h2>{children}</h2>,
        h2: ({ children }) => <h3>{children}</h3>,
        h3: ({ children }) => <h4>{children}</h4>,
        h4: ({ children }) => <h5>{children}</h5>,
        h5: ({ children }) => <h6>{children}</h6>,
        a: ({ href, children }) => href && /^https?:\/\//i.test(href)
          ? <a href={href}>{children}</a>
          : <span>{children}</span>,
        img: ({ alt }) => <span>{alt}</span>,
      }}
    >
      {source}
    </Markdown>
  );
}
