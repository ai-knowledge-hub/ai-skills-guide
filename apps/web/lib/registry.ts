import fs from "node:fs/promises";
import path from "node:path";
import { projectCurrentCatalogEntry } from "./currentCatalogueProjection.mjs";
import { findCatalogueCompatibility, type CatalogueCompatibility } from "./catalogueCompatibility";
import { executionRuntimesFor, moduleDirectory, type InstallationGuidance } from "./installation";

export type ModuleKey = "skills" | "agents" | "tools" | "plugins";

export type ExecutionKind =
  | "instructions"
  | "script"
  | "cli"
  | "mcp-server"
  | "service"
  | "orchestrator"
  | "bundle"
  | "integration-template";

export type AuthenticationMethod =
  | "none"
  | "api-key"
  | "bearer-token"
  | "oauth-authorization-code-pkce"
  | "oauth-device-flow"
  | "oauth-client-credentials"
  | "service-account"
  | "workload-identity"
  | "brokered"
  | "custom";

export type VersionEntry = {
  version: string;
  released_at: string;
  manifest_url: string;
  manifest_sha256?: string;
  artifact_url: string;
  sha256: string;
};

export type CompatibilityRequirement = {
 kind: "api" | "sdk" | "protocol" | "runtime" | "platform";
 name: string;
 versions?: string[];
 capabilities?: string[];
 notes?: string;
};

export type RegistryEntry = {
  catalogueCompatibility?: CatalogueCompatibility;
 compatibility?: CompatibilityRequirement[];
  schema_version?: "1.1" | "2.0" | "2.1" | "2.2";
  id: string;
  name: string;
  description: string;
  category: string;
  latest: string;
  versions: VersionEntry[];
  runtimes: string[];
  tags: string[];
  readiness: "experimental" | "reviewed" | "deprecated";
  usability: {
    availability: "usable-now" | "setup-required" | "not-verified" | "template-only" | "documentation-only";
    execution: "instructions" | "local-tool" | "remote-integration" | "integration-template" | "orchestrator" | "bundle" | "documentation";
    requires_setup?: string[];
    limitations?: string[];
    quickstart?: string;
    executable_helpers?: Array<{
      entrypoint: string;
      availability: "usable-now" | "setup-required" | "not-verified";
      execution: "local-tool";
      limitations: string[];
      quickstart?: string;
    }>;
    source: "declared" | "inferred";
  };
  security_reviewed: boolean;
  deprecated: boolean;
  replaced_by?: string;
  operational?: {
    connected_system?: string;
    capabilities?: string[];
    auth_required?: string[];
    access_level?: string;
    trust_boundary?: string;
    approval_boundary?: string;
    role?: string;
    coordinates?: string[];
    autonomy_level?: string;
    outputs?: string[];
    use_when?: string;
    execution_mode?: string;
  };
  execution?: {
    kind: ExecutionKind;
    command?: string[];
    healthcheck?: string[];
    smoke_test?: string[];
    supported_platforms: ("linux" | "macos" | "windows" | "web")[];
    supported_runtimes: string[];
  };
  artifact?: {
    self_contained: boolean;
    dependency_lock: string | null;
    checksums: string | null;
    sbom: string | null;
  };
  authentication?: {
    status: "none" | "optional" | "required";
    methods: AuthenticationMethod[];
    credential_bindings: string[];
    scopes: string[];
    setup_url?: string;
    credential_storage: string;
    validation: string;
    revocation: string;
  };
  provider_dependencies?: Array<{
    tool: string;
    requirement: "required" | "optional";
    access: "read-only" | "read-write";
  }>;
  capability_readiness?: Array<{
    id: string;
    name: string;
    description: string;
    access: "offline" | "read-only" | "read-write";
    availability: "usable-now" | "setup-required" | "not-verified" | "template-only";
    requires_setup?: string[];
    limitations?: string[];
  }>;
  verification?: {
    evidence: string[];
    last_verified_at: string;
  };
  dependencies?: {
    agents?: string[];
    skills?: string[];
    tools?: string[];
    apis?: string[];
    mcp_servers?: string[];
  };
  orchestration?: Record<string, unknown>;
  includes?: {
    skills?: string[];
    agents?: string[];
    tools?: string[];
    hooks?: string[];
  };
  requires?: {
    secrets?: string[];
    approvals?: string[];
  };
};

export type RegistryIndex = {
  registry_version: "1.1" | "1.2" | "1.3";
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
  if (module === "plugins") {
    return path.join(base, "plugins-index.json");
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
  const parsed = JSON.parse(data) as { registry_version?: unknown };
  if (parsed.registry_version !== "1.1" && parsed.registry_version !== "1.2" && parsed.registry_version !== "1.3") {
    throw new Error(`Unsupported registry_version ${String(parsed.registry_version)}; upgrade this consumer or use a compatible registry snapshot.`);
  }
  // Canonical indexes preserve immutable declarations. Render their current
  // static projection, just as publication does, before decorating version notes.
  const index = parsed as RegistryIndex;
  return withCatalogueCompatibility({
    ...index,
    skills: index.skills.map(projectCurrentCatalogEntry)
  }, module);
}

// Keep release selection pure so old/new registry fixtures need no filesystem
// changes or process-wide mocks to exercise the same path as the site loader.
export function withCatalogueCompatibility(index: RegistryIndex, module: ModuleKey): RegistryIndex {
  return {
    ...index,
    skills: index.skills.map((entry) => ({
      ...entry,
      catalogueCompatibility: findCatalogueCompatibility(module, entry.id, entry.latest)
    }))
  };
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

export async function loadPluginsRegistry() {
  return loadRegistry("plugins");
}

export async function getEntryById(module: ModuleKey, id: string): Promise<RegistryEntry | undefined> {
  const registry = await loadRegistry(module);
  return registry.skills.find((s) => s.id === id);
}

export async function getSkillById(id: string): Promise<SkillEntry | undefined> {
  return getEntryById("skills", id);
}

export function buildInstallSnippet(skill: RegistryEntry, runtime: "codex" | "claude" | "generic", guidance: InstallationGuidance) {
  return buildModuleInstallSnippet("skills", skill, runtime, guidance);
}

export function buildModuleInstallSnippet(
  module: ModuleKey,
  entry: RegistryEntry,
  runtime: "codex" | "claude" | "generic",
  guidance: InstallationGuidance,
) {
  if (guidance.unavailable.length || !guidance.adapters.includes(runtime) || guidance.platforms?.length === 0) return "";
  const registry = `https://skills.ai-knowledge-hub.org/registry/${module}-index.json`;
  const base = `skills-hub install --source remote --module ${module} --registry-url ${registry} --entry ${entry.id}@${entry.latest} --runtime ${runtime}`;
  const target = runtime === "generic" ? ` --target ./my-agent/${moduleDirectory(module)}` : "";
  const environments = executionRuntimesFor(guidance, runtime).map((value) => ` --execution-runtime ${value}`).join("");
  return `${base}${target}${environments}`;
}

export function uniqueValues(values: string[]) {
  return [...new Set(values)].sort((a, b) => a.localeCompare(b));
}
