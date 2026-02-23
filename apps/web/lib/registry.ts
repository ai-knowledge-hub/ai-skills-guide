import fs from "node:fs/promises";
import path from "node:path";

export type VersionEntry = {
  version: string;
  released_at: string;
  manifest_url: string;
  artifact_url: string;
  sha256: string;
};

export type SkillEntry = {
  id: string;
  name: string;
  description: string;
  category: string;
  latest: string;
  versions: VersionEntry[];
  runtimes: string[];
  tags: string[];
  deprecated: boolean;
  replaced_by?: string;
};

export type RegistryIndex = {
  registry_version: string;
  generated_at: string;
  skills: SkillEntry[];
};

function registryPath() {
  return path.resolve(process.cwd(), "..", "..", "registry", "index.json");
}

export async function loadRegistry(): Promise<RegistryIndex> {
  const data = await fs.readFile(registryPath(), "utf-8");
  return JSON.parse(data) as RegistryIndex;
}

export async function getSkillById(id: string): Promise<SkillEntry | undefined> {
  const registry = await loadRegistry();
  return registry.skills.find((s) => s.id === id);
}

export function buildInstallSnippet(skill: SkillEntry, runtime: "codex" | "claude" | "generic") {
  const base = `./bin/skills-hub install ${skill.id}@${skill.latest}`;
  if (runtime === "generic") {
    return `${base} --runtime generic --target ./my-agent/skills`;
  }
  return `${base} --runtime ${runtime}`;
}

export function uniqueValues(values: string[]) {
  return [...new Set(values)].sort((a, b) => a.localeCompare(b));
}
