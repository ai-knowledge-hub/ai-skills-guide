import { spawnSync } from "node:child_process";
import { promises as fs } from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(process.cwd(), "..", "..");
const publicRoot = path.resolve(process.cwd(), "public");

const modules = [
  { dir: "skills", manifest: "skill.yaml" },
  { dir: "agents", manifest: "agent.yaml" },
  { dir: "tools-mcp", manifest: "tool.yaml" }
];

async function main() {
  await fs.mkdir(publicRoot, { recursive: true });

  // Rebuild generated static directories so manifest/artifact URLs stay in sync.
  for (const generatedDir of ["skills", "agents", "tools-mcp", "artifacts", "registry"]) {
    await fs.rm(path.join(publicRoot, generatedDir), { recursive: true, force: true });
  }

  for (const moduleConfig of modules) {
    await publishModuleManifests(moduleConfig);
  }
  await publishRegistryIndexes();
}

async function publishModuleManifests({ dir, manifest }) {
  const moduleRoot = path.join(repoRoot, dir);
  try {
    await fs.access(moduleRoot);
  } catch {
    return;
  }

  const manifestPaths = [];
  await walk(moduleRoot, async (filePath) => {
    if (path.basename(filePath) === manifest) {
      manifestPaths.push(filePath);
    }
  });

  for (const manifestPath of manifestPaths) {
    const entryDir = path.dirname(manifestPath);
    const relManifestPath = path.relative(repoRoot, manifestPath);
    const outManifestPath = path.join(publicRoot, relManifestPath);

    await fs.mkdir(path.dirname(outManifestPath), { recursive: true });
    await fs.copyFile(manifestPath, outManifestPath);

    const manifestRaw = await fs.readFile(manifestPath, "utf-8");
    const id = extractScalar(manifestRaw, "id");
    const version = extractScalar(manifestRaw, "version");
    if (!id || !version) {
      throw new Error(`Missing id/version in manifest: ${manifestPath}`);
    }

    const artifactPath = path.join(publicRoot, "artifacts", ...id.split("/"), `${version}.tar.gz`);
    await fs.mkdir(path.dirname(artifactPath), { recursive: true });
    createTarball(entryDir, artifactPath);
  }
}

async function publishRegistryIndexes() {
  const registryRoot = path.join(repoRoot, "registry");
  const outRegistryRoot = path.join(publicRoot, "registry");
  await fs.mkdir(outRegistryRoot, { recursive: true });

  for (const fileName of ["index.json", "skills-index.json", "agents-index.json", "tools-index.json"]) {
    const inPath = path.join(registryRoot, fileName);
    try {
      await fs.access(inPath);
    } catch {
      continue;
    }
    await fs.copyFile(inPath, path.join(outRegistryRoot, fileName));
  }
}

async function walk(root, onFile) {
  const entries = await fs.readdir(root, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      await walk(fullPath, onFile);
      continue;
    }
    await onFile(fullPath);
  }
}

function extractScalar(raw, key) {
  const regex = new RegExp(`^${key}:\\s*(.+)\\s*$`, "m");
  const match = raw.match(regex);
  if (!match) {
    return "";
  }
  return match[1].trim().replace(/^['"]|['"]$/g, "");
}

function createTarball(sourceDir, outputPath) {
  const result = spawnSync("tar", ["-czf", outputPath, "-C", sourceDir, "."], {
    stdio: "pipe"
  });
  if (result.status !== 0) {
    const stderr = result.stderr?.toString("utf-8") ?? "";
    throw new Error(`tar failed for ${sourceDir}: ${stderr}`);
  }
}

await main();
