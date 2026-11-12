import type { ModuleKey, RegistryEntry } from "./registry";
import { readReleaseFiles } from "./releaseFiles";

type FirstUseNote = {
  module: ModuleKey;
  id: string;
  version: string;
  sources: string[];
  inputs?: string[];
  outputs?: string[];
  prerequisites?: string[];
  firstStep?: string;
  sample?: { runtime: string; command: string; result: string };
};

// Editorial summaries are scoped to the released files named below. They
// describe supplied procedures and examples, without asserting runtime success.
export const firstUseNotes: FirstUseNote[] = [
  {
    module: "skills", id: "marketing/meta-google-weekly-performance-review", version: "0.1.0",
    sources: ["SKILL.md", "README.md"],
    inputs: ["Meta and Google Ads account IDs", "A date range, with current and previous-period campaign metrics or export files"],
    outputs: ["Executive summary and KPI table with period deltas", "Key insights and prioritized actions", "Risks, missing data and low-confidence findings"],
    prerequisites: ["Supply real metrics. Request export files when data access is unavailable."],
    firstStep: "Read the procedure, then copy it with your inputs into an agent app or chat that supports pasted instructions. Confirm how that host uses the text. Reading and copying here require no terminal setup.",
  },
  {
    module: "agents", id: "marketing/weekly-performance-supervisor", version: "0.2.0",
    sources: ["AGENT.md", "README.md", "examples/input.json"],
    inputs: ["Current and previous reporting periods", "Required channels and the report's audience"],
  },
  {
    module: "tools", id: "analytics/ga4-mcp-connector", version: "0.2.0",
    sources: ["TOOL.md", "README.md"],
    inputs: ["An allowed GA4 property ID", "Start and end dates", "Packaged metric names, with optional dimensions and result limits"],
    outputs: ["Normalized report columns and rows", "Pagination, provider identity and retrieval freshness", "Bounded error diagnostics when a request fails"],
  },
  {
    module: "tools", id: "adtech/conversion-event-reconciler", version: "0.1.0",
    sources: ["TOOL.md", "README.md", "examples/input.json", "examples/output.json"],
    inputs: ["JSON browser_events and server_events, with optional first-party outcomes", "Pseudonymous event and order IDs"],
    outputs: ["Canonical events with deduplication and source provenance", "Outcome reconciliation, counts and warnings"],
    prerequisites: ["Python 3 for the packaged sample. Keep access tokens, raw emails and payment credentials out of event payloads."],
    sample: { runtime: "Python 3", command: "python3 scripts/reconcile_events.py examples/input.json", result: "JSON canonical events, reconciliation counts and missing-ID warnings on standard output." },
  },
  {
    module: "tools", id: "adtech/openai-ads-api-client", version: "0.2.0",
    sources: ["README.md", "TOOL.md", "tool.yaml", "examples/account.json"],
    inputs: ["A selected operation and account scope", "Packaged fixtures for mock mode, or separately configured credentials for live reads"],
    outputs: ["Bounded account, campaign, ad or insights data for the selected operation", "Local or non-persisting conversion-validation results when requested"],
    prerequisites: ["Python 3.9 or newer for mock mode. Review the tool contract before connecting an account."],
    sample: { runtime: "Python 3.9 or newer", command: "python3 scripts/openai_ads_client.py --mode mock account", result: "The packaged mock account response. This sample makes no provider call." },
  },
  {
    module: "plugins", id: "engineering/code-maintenance-plugin", version: "0.2.0",
    sources: ["README.md", "plugin.yaml", "examples/first-use-input.json"],
    inputs: ["A proposed change, its risk and a verification command"],
    outputs: ["A bounded maintenance plan with an explicit verification step"],
    prerequisites: ["Node.js 22 for the packaged sample. Merge decisions remain human-owned."],
    sample: { runtime: "Node.js 22", command: "node scripts/first_use.mjs examples/first-use-input.json", result: "A bounded plan on standard output. It does not authorize merge." },
  },
  {
    module: "plugins", id: "marketing/content-repurposing-plugin", version: "0.3.0",
    sources: ["README.md", "plugin.yaml", "examples/first-use-input.json"],
    outputs: ["article.md, social-post.txt, email-brief.json and manifest.json", "Drafts requiring human review before publication"],
    prerequisites: ["Node.js 22 for the packaged sample. Review every generated deliverable before publication."],
    sample: { runtime: "Node.js 22", command: "node scripts/first_use.mjs examples/first-use-input.json ./first-use-output", result: "Four draft files in ./first-use-output. The sample does not publish externally." },
  },
  {
    module: "plugins", id: "marketing/chatgpt-advertising-experiment-plugin", version: "0.2.0",
    sources: ["README.md", "examples/experiment.json"],
    inputs: ["The packaged experiment and conversion fixtures for the offline lab"],
    outputs: ["An offline experiment, conversion-validation and reconciliation evidence bundle"],
    prerequisites: ["Python 3.9 or newer for the offline lab. Credentials are required separately for live reads."],
    sample: { runtime: "Python 3.9 or newer", command: "python3 scripts/run_mock_lab.py", result: "output/mock-evidence-bundle.json from packaged fixtures, with no network calls." },
  },
  {
    module: "plugins", id: "agentops/harness-governance-plugin", version: "0.2.0",
    sources: ["README.md", "plugin.yaml", "examples/first-use-input.json"],
    outputs: ["A bounded decision record for a proposed harness change"],
    prerequisites: ["Node.js 22 for the packaged sample. Harness adoption requires a separate human decision."],
    sample: { runtime: "Node.js 22", command: "node scripts/first_use.mjs examples/first-use-input.json", result: "A bounded decision record on standard output. It does not authorize adoption." },
  },
  {
    module: "plugins", id: "marketing/competitive-intelligence-plugin", version: "0.2.0",
    sources: ["README.md", "plugin.yaml", "examples/first-use-input.json"],
    outputs: ["External signals labeled untrusted and requiring human review before publication"],
    prerequisites: ["Node.js 22 for the packaged sample. Review external claims before publication."],
    sample: { runtime: "Node.js 22", command: "node scripts/first_use.mjs examples/first-use-input.json", result: "Signals labeled untrusted and requiring human review before publication." },
  },
  {
    module: "plugins", id: "marketing/creative-operating-system-plugin", version: "0.2.0",
    sources: ["README.md", "plugin.yaml", "examples/first-use-input.json"],
    outputs: ["An ordered workflow with an approval owner"],
    prerequisites: ["Node.js 22 for the packaged sample. Public launch remains approval-gated."],
    sample: { runtime: "Node.js 22", command: "node scripts/first_use.mjs examples/first-use-input.json", result: "An ordered workflow and approval owner. The sample does not authorize launch." },
  },
  {
    module: "plugins", id: "security/runtime-safety-plugin", version: "0.2.0",
    sources: ["README.md", "plugin.yaml", "examples/first-use-input.json"],
    outputs: ["A conservative security decision for the supplied finding"],
    prerequisites: ["Node.js 22 for the packaged sample. Remediation requires separate review and approval."],
    sample: { runtime: "Node.js 22", command: "node scripts/first_use.mjs examples/first-use-input.json", result: "A conservative decision on standard output. The sample performs no remediation." },
  },
  {
    module: "tools", id: "adtech/ad-platform-executor-template", version: "0.2.0",
    sources: ["TOOL.md", "README.md", "examples/approved-change.json", "examples/execution-result.json"],
    inputs: ["A bounded provider-native plan with exact target and desired state", "Runtime-owned credentials, policy and a separately authorized control-plane grant"],
    outputs: ["Plan previews, execution receipts and post-write state verification", "Reconciliation or separately authorized rollback results"],
  },
];

export function findFirstUseNote(module: ModuleKey, id: string, version: string) {
  return firstUseNotes.find((note) => note.module === module && note.id === id && note.version === version);
}

// Package instructions use level-two sections. Ignore headings inside fenced
// examples so an embedded output template cannot become an input declaration.
export function instructionSection(source: string, titles: string[]) {
  const lines: string[] = [];
  let collecting = false;
  let fence: string | undefined;
  for (const line of source.split(/\r?\n/)) {
    const marker = /^\s{0,3}(`{3,}|~{3,})/.exec(line)?.[1];
    if (marker) {
      if (!fence) fence = marker;
      else if (marker[0] === fence[0] && marker.length >= fence.length) fence = undefined;
    }
    const heading = !fence && /^##\s+(.+?)\s*$/.exec(line);
    if (heading) {
      if (collecting) break;
      collecting = titles.some((title) => heading[1].toLowerCase() === title.toLowerCase());
      continue;
    }
    if (collecting) lines.push(line);
  }
  return lines.join("\n").trim();
}

function exampleFields(source: string | undefined) {
  if (!source) return "";
  try {
    const parsed: unknown = JSON.parse(source);
    if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") return "";
    return `The released example shows these fields. Check the guide for which are required.\n\n${Object.keys(parsed).map((key) => `- \`${key}\``).join("\n")}`;
  } catch {
    return "";
  }
}

const bulletList = (items: string[]) => items.map((item) => `- ${item}`).join("\n");

const defaultFirstSteps: Record<ModuleKey, string> = {
  skills: "Read the procedure and supply its documented inputs. Use the full text in an agent app or chat that supports pasted instructions, following that host's guidance.",
  agents: "Review the coordinated workflow and its runtime requirements before configuring an agent app.",
  plugins: "Inspect the bundled components and their requirements before installing files or configuring your agent app.",
  tools: "Review the setup requirements, access boundaries and release guide before connected use.",
};

export async function loadResourceFirstUse(module: ModuleKey, entry: RegistryEntry) {
  const directory = module === "tools" ? "tools-mcp" : module;
  const document = module === "skills" ? "SKILL.md" : module === "agents" ? "AGENT.md" : module === "tools" ? "TOOL.md" : "README.md";
  const manifest = module === "skills" ? "skill.yaml" : module === "agents" ? "agent.yaml" : module === "tools" ? "tool.yaml" : "plugin.yaml";
  const matchedNote = findFirstUseNote(module, entry.id, entry.latest);
  const names = [...new Set([
    document, manifest, "README.md", "examples/input.json", "examples/output.json", "examples/first-use-input.json",
    ...(matchedNote?.sources ?? []),
  ])];
  const files = await readReleaseFiles(directory, entry, names, {
    // README and examples enrich the summary; unavailable previews must not
    // hide the remaining metadata or plugin contents.
    optionalNames: names.filter((name) => name.startsWith("examples/") || name === "README.md"),
  });
  const note = matchedNote?.sources.every((source) => files?.has(source)) ? matchedNote : undefined;
  const instructions = files?.get(document) ?? "";
  const exampleInputPath = files?.has("examples/first-use-input.json") ? "examples/first-use-input.json" : "examples/input.json";
  const inputs = note?.inputs ? bulletList(note.inputs)
    : instructionSection(instructions, ["Inputs required", "Inputs"])
      || exampleFields(files?.get(exampleInputPath))
      || "Inputs are not documented for this release. Inspect its guide before use.";
  const outputs = note?.outputs ? bulletList(note.outputs)
    : entry.operational?.outputs?.length ? bulletList(entry.operational.outputs)
      : exampleFields(files?.get("examples/output.json"))
        || "Expected outputs are not documented for this release. Inspect its guide before use.";
  const requirements = note?.prerequisites
    ?? (entry.usability.requires_setup?.length ? entry.usability.requires_setup : undefined)
    ?? (entry.operational?.auth_required?.length ? entry.operational.auth_required : undefined)
    ?? (entry.requires?.secrets?.length ? entry.requires.secrets.map((binding) => `Required configuration binding: ${binding}`) : undefined)
    ?? (module === "skills" ? ["Reading and copying the procedure require no terminal setup. Check requirements for any packaged programs separately."]
      : module === "agents" && entry.usability.quickstart ? [entry.usability.quickstart]
        : ["Additional setup prerequisites are not documented here. Review the full requirements below."]);
  const isTemplate = entry.usability.availability === "template-only";
  const firstStep = note?.firstStep ?? (isTemplate
    ? "Inspect the released reference and identify the implementation and bindings to adapt before attempting execution."
    : note?.sample ? "Inspect the packaged offline sample before running it. After file installation, use the stated runtime from the installed package directory."
    : defaultFirstSteps[module]);
  const limits = [...(entry.usability.limitations ?? []), ...(entry.operational?.approval_boundary ? [entry.operational.approval_boundary] : [])];
  if (entry.usability.executable_helpers?.length) limits.push("Packaged programs have independent setup and verification requirements. Reading or copying this procedure does not verify their execution.");
  return {
    files, document, inputs, outputs, requirements, firstStep, limits,
    sample: !isTemplate ? note?.sample : undefined,
    exampleInput: files?.get(exampleInputPath),
    exampleOutput: files?.get("examples/output.json"),
    sources: [...new Set([manifest, ...(note?.sources ?? [document]), exampleInputPath, "examples/output.json"])].filter((source) => files?.has(source)),
  };
}

export type ResourceFirstUseContent = Awaited<ReturnType<typeof loadResourceFirstUse>>;
