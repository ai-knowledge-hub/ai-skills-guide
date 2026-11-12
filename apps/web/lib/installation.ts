import type { ModuleKey, RegistryEntry } from "./registry";

export type RuntimeKey = "codex" | "claude" | "generic";

export type InstallationGuidance = {
  adapters: string[];
  constraints: { source: string; platforms: string[]; runtimes: string[] }[];
  platforms: string[] | null;
  bundledModules: ModuleKey[];
  unavailable: string[];
};

export const installerRelease = {
  version: "v0.4.0-alpha.1",
  url: "https://github.com/ai-knowledge-hub/ai-skills-guide/releases/tag/v0.4.0-alpha.1",
  assets: "https://github.com/ai-knowledge-hub/ai-skills-guide/releases/download/v0.4.0-alpha.1",
};

export function moduleDirectory(module: ModuleKey) {
  return module === "tools" ? "tools-mcp" : module;
}

export function executionRuntimesFor(guidance: InstallationGuidance, runtime: RuntimeKey) {
  const available = new Set(["native", runtime]);
  const selected: string[] = [];
  // Satisfy narrower declarations first so a shared allowed environment can
  // also satisfy broader alternatives elsewhere in the closure.
  const constraints = [...guidance.constraints].sort((a, b) => a.runtimes.length - b.runtimes.length);
  for (const constraint of constraints) {
    if (constraint.runtimes.some((value) => available.has(value))) continue;
    const environment = constraint.runtimes[0];
    if (environment) { selected.push(environment); available.add(environment); }
  }
  return selected;
}

export function installationModuleTarget(module: ModuleKey, runtime: RuntimeKey) {
  const directory = moduleDirectory(module);
  return runtime === "generic" ? `./my-agent/${directory}` : `~/.${runtime}/${directory}`;
}

export function installationTarget(module: ModuleKey, id: string, runtime: RuntimeKey) {
  return `${installationModuleTarget(module, runtime)}/${id}`;
}
