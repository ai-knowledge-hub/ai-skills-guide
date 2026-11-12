import type { ModuleKey, RegistryEntry } from "./registry";

export type RuntimeKey = "codex" | "claude" | "generic";

export const installerRelease = {
  version: "v0.4.0-alpha.1",
  url: "https://github.com/ai-knowledge-hub/ai-skills-guide/releases/tag/v0.4.0-alpha.1",
  assets: "https://github.com/ai-knowledge-hub/ai-skills-guide/releases/download/v0.4.0-alpha.1",
};

export function moduleDirectory(module: ModuleKey) {
  return module === "tools" ? "tools-mcp" : module;
}

export function executionRuntimeFor(entry: Pick<RegistryEntry, "execution">, runtime: RuntimeKey) {
  const supported = entry.execution?.supported_runtimes ?? [];
  if (supported.includes("native") || supported.includes(runtime)) return undefined;
  return supported[0];
}

export function installationTarget(module: ModuleKey, id: string, runtime: RuntimeKey) {
  const directory = moduleDirectory(module);
  if (runtime === "generic") return `./my-agent/${directory}/${id}`;
  return `~/.${runtime}/${directory}/${id}`;
}
