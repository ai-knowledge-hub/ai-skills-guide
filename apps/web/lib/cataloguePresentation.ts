import type { RegistryEntry } from "./registry";

export const resourceFormLabels: Record<RegistryEntry["usability"]["execution"], string> = {
  instructions: "Agent instructions",
  "local-tool": "Local program",
  "remote-integration": "API or MCP integration",
  "integration-template": "Integration template",
  orchestrator: "Agent workflow",
  bundle: "Component bundle",
  documentation: "Guide",
};

export const availabilityDescriptions: Record<RegistryEntry["usability"]["availability"], string> = {
  "documentation-only": "Read or copy the guidance. Any packaged programs have separate execution requirements and evidence.",
  "template-only": "Implementation is required before the described workflow can run. Inspect and adapt the source.",
  "not-verified": "Current operational evidence for your environment is not shown here. Check the implementation, installation requirements and limits before use.",
  "setup-required": "Configuration, credentials or runtime bindings are required. Check the prerequisites and limits for your intended environment.",
  "usable-now": "Inspect the evidence for the exact operation and environment covered by this classification.",
};

export function installationExplanation(execution: RegistryEntry["usability"]["execution"]) {
  if (execution === "instructions" || execution === "documentation") {
    return "These commands copy guidance into the selected runtime directories. Confirm that your agent app discovers it. Any packaged programs need separate setup and verification.";
  }
  return "These commands install package files and generate runtime configuration where supported. Confirm discovery in your agent app, then check configuration, authentication and a first useful operation.";
}
