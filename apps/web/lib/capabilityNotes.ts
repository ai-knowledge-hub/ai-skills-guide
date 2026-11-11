import type { ModuleKey } from "./registry";

// Editorial explanations preserve the release's recorded classification.
// Match the entire release identity so a later implementation needs its own review.
export const capabilityNotes = [{
  module: "plugins" as ModuleKey,
  id: "marketing/chatgpt-advertising-experiment-plugin",
  version: "0.2.0",
  capability: "live-write",
  label: "Unavailable",
  explanation: "Live writes unavailable: no supported OpenAI Ads mutation adapter is included in this release.",
  source: {
    label: "Release declaration",
    url: "https://github.com/ai-knowledge-hub/ai-skills-guide/blob/0a81fa98c8c47cebab16f93bdaf1283b21ed88c4/releases/plugins/marketing/chatgpt-advertising-experiment-plugin/0.2.0/plugin.yaml",
  },
}];

export function findCapabilityNote(module: ModuleKey, id: string, version: string, capability: string) {
  return capabilityNotes.find((note) => note.module === module && note.id === id && note.version === version && note.capability === capability);
}
