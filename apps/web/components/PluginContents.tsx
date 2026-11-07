import DetailDisclosure from "@/components/DetailDisclosure";
import InstructionMarkdown from "@/components/InstructionMarkdown";
import { readReleaseFiles } from "@/lib/releaseFiles";
import type { RegistryEntry } from "@/lib/registry";

const groups = [
  { key: "skills", module: "skills", label: "skills" },
  { key: "agents", module: "agents", label: "agents" },
  { key: "tools", module: "tools-mcp", label: "tools" },
] as const;

type DependencyLock = {
  root: { module: string; id: string; version: string };
  components: { module: string; id: string; version: string }[];
};

export default async function PluginContents({ entry }: { entry: RegistryEntry }) {
  const hooks = entry.includes?.hooks ?? [];
  const lockPath = entry.artifact?.self_contained ? entry.artifact.dependency_lock : null;
  const files = await readReleaseFiles("plugins", entry, [
    ...(lockPath ? [lockPath] : []),
    ...hooks.map((hook) => `hooks/${hook}.md`),
  ]);
  let lock: DependencyLock | undefined;
  if (files && lockPath) {
    const source = files.get(lockPath);
    if (!source) throw new Error(`Missing dependency lock in ${entry.id}@${entry.latest}`);
    lock = JSON.parse(source) as DependencyLock;
    if (lock.root.module !== "plugins" || lock.root.id !== entry.id || lock.root.version !== entry.latest) {
      throw new Error(`Dependency lock does not match ${entry.id}@${entry.latest}`);
    }
  }

  const isReference = entry.usability.availability === "template-only";
  const groupPrefix = isReference ? "Referenced" : lock ? "Bundled" : "Declared";
  const componentGroups = groups.map(({ key, module, label }) => ({
    module,
    label,
    items: lock && !isReference
      ? lock.components.filter((component) => component.module === module)
      : (entry.includes?.[key] ?? []).map((id) => ({ id, version: undefined })),
  })).filter(({ items }) => items.length);

  return (
    <DetailDisclosure
      show={isReference ? "View reference contents" : "View package contents"}
      hide={isReference ? "Hide reference contents" : "Hide package contents"}
      detail={`v${entry.latest}`}
    >
      {isReference ? <p>This reference template describes how the components fit together.</p> : null}
      {componentGroups.map(({ module, label, items }) => (
        <section key={module}>
          <h2>{groupPrefix} {label}</h2>
          <ul>
            {items.map(({ id, version }) => (
              <li key={id}>
                <a href={`/${module}/${id}`}>{id}</a>
                {version ? <> <span className="meta detail-disclosure-meta">v{version}</span></> : null}
              </li>
            ))}
          </ul>
        </section>
      ))}
      {hooks.length ? (
        <section>
          <h2>Packaged hooks</h2>
          {hooks.map((hook) => {
            const source = files?.get(`hooks/${hook}.md`);
            return source ? (
              <details className="detail-disclosure-nested" key={hook}>
                <summary>{hook}</summary>
                <div className="detail-disclosure-document"><InstructionMarkdown source={source} /></div>
              </details>
            ) : <p key={hook}>{hook}</p>;
          })}
        </section>
      ) : null}
      {!componentGroups.length && !hooks.length ? <p>No components or hooks are documented for this release.</p> : null}
    </DetailDisclosure>
  );
}
