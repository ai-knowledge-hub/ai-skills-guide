import { parse } from "yaml";
import type { ModuleKey, RegistryEntry } from "./registry";
import type { InstallationGuidance } from "./installation";
import { moduleDirectory } from "./installation";
import { readReleaseFiles } from "./releaseFiles";

type ManifestRequirements = {
  schema_version?: string;
  execution?: RegistryEntry["execution"];
  usability?: { executable_helpers?: unknown[] };
  orchestration?: Record<string, unknown>;
};

function executionConstraint(source: string, manifest: ManifestRequirements) {
  const execution = manifest.execution;
  if (!manifest.schema_version?.startsWith("2.") || !execution
    || (!manifest.usability?.executable_helpers?.length && ["instructions", "integration-template", ""].includes(execution.kind))) return null;
  if (!Array.isArray(execution.supported_platforms) || !execution.supported_platforms.length
    || !Array.isArray(execution.supported_runtimes) || !execution.supported_runtimes.length
    || [...execution.supported_platforms, ...execution.supported_runtimes].some((value) => typeof value !== "string" || !value.trim())) {
    throw new Error("Unavailable execution declaration");
  }
  if (execution.supported_runtimes.some((value) => !/^[a-z0-9][a-z0-9.-]{1,63}$/.test(value))) throw new Error("Unavailable execution environment");
  return {
    source,
    platforms: execution.supported_platforms.map((value) => value.trim().toLowerCase()),
    runtimes: execution.supported_runtimes.map((value) => value.trim().toLowerCase()),
  };
}

// The installer admits each manifest in the actual bundled tree, not merely
// the current catalogue version of a dependency with the same ID.
export async function loadInstallationGuidance(module: ModuleKey, entry: RegistryEntry): Promise<InstallationGuidance> {
  const result: InstallationGuidance = { adapters: entry.runtimes.map((value) => value.toLowerCase()), constraints: [], platforms: null, bundledModules: [], unavailable: [] };
  const root = executionConstraint(`${module}/${entry.id}@${entry.latest}`, entry);
  if (root) result.constraints.push(root);
  if (((module === "agents" && entry.orchestration) || (module === "plugins" && entry.artifact?.self_contained))
    && entry.usability.availability !== "template-only") {
    const pattern = /^bundled\/(skills|agents|tools-mcp)\/([^/]+\/[^/]+)\/(skill|agent|tool)\.yaml$/;
    const filenames = { skills: "skill", agents: "agent", "tools-mcp": "tool" } as const;
    const expected = new Map<string, string>();
    const lockedMembers = new Set<string>();
    const actual = new Map<string, string>();
    const unavailable = (name: string) => { if (!result.unavailable.includes(name)) result.unavailable.push(name); };
    const expectMember = (directory: string, id: string, version: string) => {
      const filename = filenames[directory as keyof typeof filenames];
      if (!filename) throw new Error("Unknown dependency module");
      const path = `bundled/${directory}/${id}/${filename}.yaml`;
      if (expected.has(path) && expected.get(path) !== version) unavailable(path);
      expected.set(path, version);
      return path;
    };
    const pinnedBindings = (orchestration: Record<string, unknown>, source: string) => {
      if (!Array.isArray(orchestration.bindings)) { unavailable(source); return; }
      for (const value of orchestration.bindings) {
        if (!value || typeof value !== "object") { unavailable(source); continue; }
        const binding = value as Record<string, unknown>;
        if (binding.availability !== "resolved") continue;
        const directory = { skill: "skills", agent: "agents", tool: "tools-mcp" }[String(binding.kind)];
        if (!directory || typeof binding.package !== "string" || typeof binding.version !== "string") { unavailable(source); continue; }
        expectMember(directory, binding.package, binding.version);
      }
    };
    if (module === "agents") pinnedBindings(entry.orchestration!, "Pinned agent bindings");
    const lockPath = module === "plugins" ? entry.artifact?.dependency_lock : null;
    const files = await readReleaseFiles(moduleDirectory(module) as "agents" | "plugins", entry, (name) => {
      const match = pattern.exec(name);
      return name === lockPath || (!!match && filenames[match[1] as keyof typeof filenames] === match[3]);
    }, { optionalAll: true, maxTotalBytes: 4 * 1024 * 1024, onUnavailable: unavailable });
    if (!files) unavailable("Released dependency closure");
    if (module === "plugins") {
      try {
        const lock = JSON.parse(files?.get(lockPath ?? "") ?? "") as {
          lock_version: string; root: { module: string; id: string; version: string };
          components: { module: string; id: string; version: string; manifest_path: string }[];
        };
        if (lock.lock_version !== "1.0" || lock.root?.module !== "plugins" || lock.root.id !== entry.id
          || lock.root.version !== entry.latest || !Array.isArray(lock.components)) throw new Error("Unavailable lock");
        for (const component of lock.components) {
          if (typeof component.id !== "string" || typeof component.version !== "string"
            || expectMember(component.module, component.id, component.version) !== component.manifest_path) throw new Error("Unavailable lock member");
          if (lockedMembers.has(component.manifest_path)) throw new Error("Duplicate lock member");
          lockedMembers.add(component.manifest_path);
        }
      } catch {
        unavailable(lockPath ?? "Dependency lock");
      }
    }
    for (const [name, source] of files ?? []) {
      if (name === lockPath) continue;
      const match = pattern.exec(name)!;
      const childModule: ModuleKey = match[1] === "tools-mcp" ? "tools" : match[1] as "skills" | "agents";
      if (!result.bundledModules.includes(childModule)) result.bundledModules.push(childModule);
      try {
        const manifest = parse(source, { maxAliasCount: 100, logLevel: "error" }) as ManifestRequirements & { id: string; version: string; runtimes: string[] };
        if (!manifest || manifest.id !== match[2] || typeof manifest.version !== "string"
          || !Array.isArray(manifest.runtimes) || manifest.runtimes.some((value) => typeof value !== "string")) throw new Error("Unavailable identity or adapters");
        const adapters = manifest.runtimes.map((value) => value.toLowerCase());
        result.adapters = result.adapters.filter((value) => adapters.includes(value));
        actual.set(name, manifest.version);
        if (manifest.orchestration) pinnedBindings(manifest.orchestration, name);
        const constraint = executionConstraint(`${match[1]}/${manifest.id}@${manifest.version}`, manifest);
        if (constraint) result.constraints.push(constraint);
      } catch {
        unavailable(name);
      }
    }
    for (const [path, version] of expected) {
      if (actual.get(path) !== version) unavailable(path);
    }
    if (module === "plugins") {
      for (const path of actual.keys()) if (!lockedMembers.has(path)) unavailable(path);
    }
  }
  if (result.constraints.length) {
    result.platforms = ["linux", "macos", "windows"].filter((platform) => result.constraints.every((constraint) => constraint.platforms.includes(platform)));
  }
  return result;
}
