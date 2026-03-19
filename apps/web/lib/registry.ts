import fs from "node:fs/promises";
import path from "node:path";

export type ModuleKey = "skills" | "agents" | "tools";

export type VersionEntry = {
  version: string;
  released_at: string;
  manifest_url: string;
  artifact_url: string;
  sha256: string;
};

export type RegistryEntry = {
  id: string;
  name: string;
  description: string;
  category: string;
  latest: string;
  versions: VersionEntry[];
  runtimes: string[];
  tags: string[];
  readiness: "experimental" | "reviewed" | "deprecated";
  security_reviewed: boolean;
  deprecated: boolean;
  replaced_by?: string;
};

export type RegistryIndex = {
  registry_version: string;
  generated_at: string;
  skills: RegistryEntry[];
};

export type SkillEntry = RegistryEntry;

function moduleIndexPath(module: ModuleKey) {
  const base = path.resolve(process.cwd(), "..", "..", "registry");
  if (module === "skills") {
    return path.join(base, "skills-index.json");
  }
  if (module === "agents") {
    return path.join(base, "agents-index.json");
  }
  return path.join(base, "tools-index.json");
}

async function resolveRegistryPath(module: ModuleKey) {
  const primary = moduleIndexPath(module);
  if (module !== "skills") {
    return primary;
  }
  try {
    await fs.access(primary);
    return primary;
  } catch {
    return path.resolve(process.cwd(), "..", "..", "registry", "index.json");
  }
}

export async function loadRegistry(module: ModuleKey = "skills"): Promise<RegistryIndex> {
  const data = await fs.readFile(await resolveRegistryPath(module), "utf-8");
  return JSON.parse(data) as RegistryIndex;
}

export async function loadSkillsRegistry() {
  return loadRegistry("skills");
}

export async function loadAgentsRegistry() {
  return loadRegistry("agents");
}

export async function loadToolsRegistry() {
  return loadRegistry("tools");
}

export async function getEntryById(module: ModuleKey, id: string): Promise<RegistryEntry | undefined> {
  const registry = await loadRegistry(module);
  return registry.skills.find((s) => s.id === id);
}

export async function getSkillById(id: string): Promise<SkillEntry | undefined> {
  return getEntryById("skills", id);
}

export function buildInstallSnippet(skill: RegistryEntry, runtime: "codex" | "claude" | "generic") {
  const base = `./bin/skills-hub install ${skill.id}@${skill.latest}`;
  if (runtime === "generic") {
    return `${base} --runtime generic --target ./my-agent/skills`;
  }
  return `${base} --runtime ${runtime}`;
}

export function buildModuleInstallSnippet(
  module: ModuleKey,
  entry: RegistryEntry,
  runtime: "codex" | "claude" | "generic"
) {
  const moduleFlag = module === "tools" ? "tools" : module;
  const base = `./bin/skills-hub install --module ${moduleFlag} --entry ${entry.id}@${entry.latest}`;
  if (runtime === "generic") {
    const targetDir =
      module === "agents"
        ? "./my-agent/agents"
        : module === "tools"
          ? "./my-agent/tools-mcp"
          : "./my-agent/skills";
    return `${base} --runtime generic --target ${targetDir}`;
  }
  return `${base} --runtime ${runtime}`;
}

export function uniqueValues(values: string[]) {
  return [...new Set(values)].sort((a, b) => a.localeCompare(b));
}
