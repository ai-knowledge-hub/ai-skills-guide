import DetailDisclosure from "@/components/DetailDisclosure";
import InstructionMarkdown from "@/components/InstructionMarkdown";

export default function SkillInstructions({ source, version }: { source: string; version: string }) {
  const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/.exec(source);
  const body = frontmatter ? source.slice(frontmatter[0].length) : source;

  return (
    <DetailDisclosure show="Read full instructions" hide="Hide full instructions" detail={`SKILL.md · v${version}`}>
      <InstructionMarkdown source={body} />
      {frontmatter ? (
        <details className="detail-disclosure-nested">
          <summary>View file metadata</summary>
          <pre><code>{`---\n${frontmatter[1]}\n---`}</code></pre>
        </details>
      ) : null}
    </DetailDisclosure>
  );
}
