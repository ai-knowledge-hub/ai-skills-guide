import { createHash } from "node:crypto";
import { execFile } from "node:child_process";
import { promises as fs } from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";
import { gunzipSync, gzipSync } from "node:zlib";

const repoRoot = path.resolve(process.env.REPOSITORY_ROOT ?? path.resolve(process.cwd(), "..", ".."));
const publicRoot = path.resolve(process.env.PUBLIC_ROOT ?? path.resolve(process.cwd(), "public"));
const releaseStoreRoot = path.resolve(process.env.RELEASE_STORE_ROOT ?? path.join(repoRoot, "releases"));
const validatorRoot = path.resolve(process.env.RELEASE_VALIDATOR_ROOT ?? repoRoot);
const execFileAsync = promisify(execFile);

const modules = [
  { dir: "skills", manifest: "skill.yaml", index: "skills-index.json" },
  { dir: "agents", manifest: "agent.yaml", index: "agents-index.json" },
  { dir: "tools-mcp", manifest: "tool.yaml", index: "tools-index.json" },
  { dir: "plugins", manifest: "plugin.yaml", index: "plugins-index.json" }
];
const releaseCatalog = new Map();
const sourceEntries = new Map();
const currentReleaseEntries = new Map();
let retainedReleaseValidationUnavailable = false;
const transientPackageEntries = new Set([
  ".DS_Store",
  ".mypy_cache",
  ".pytest_cache",
  ".ruff_cache",
  "Thumbs.db",
  "__pycache__",
  "node_modules"
]);

async function main() {
  await fs.mkdir(publicRoot, { recursive: true });
  await loadSourceEntries();

  // Rebuild generated static directories so manifest/artifact URLs stay in sync.
  for (const generatedDir of ["skills", "agents", "tools-mcp", "plugins", "artifacts", "release-manifests", "registry"]) {
    await fs.rm(path.join(publicRoot, generatedDir), { recursive: true, force: true });
  }

  for (const moduleConfig of modules) {
    await publishModuleManifests(moduleConfig);
  }
  await validateRetainedReleases();
  for (const moduleConfig of modules) {
    await restoreModuleReleases(moduleConfig);
  }
  await publishRegistryIndexes();
}

async function loadSourceEntries() {
  for (const moduleConfig of modules) {
    const indexPath = path.join(repoRoot, "registry", moduleConfig.index);
    const index = JSON.parse(await fs.readFile(indexPath, "utf-8"));
    for (const entry of index.skills ?? []) {
      const { latest: _latest, versions: _versions, ...projection } = entry;
      sourceEntries.set(`${moduleConfig.dir}:${entry.id}`, {
        latest: entry.latest,
        projection,
        bytes: Buffer.from(`${JSON.stringify(projection, null, 2)}\n`)
      });
    }
  }
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

    const sourceEntry = sourceEntries.get(`${dir}:${id}`);
    if (!sourceEntry || sourceEntry.latest !== version) {
      throw new Error(`Current registry projection is missing for ${dir}:${id}@${version}`);
    }
    const manifestBytes = await fs.readFile(manifestPath);
    const artifactBytes = await createPackageTarball(dir, entryDir, sourceEntry.projection, version);
    const releaseDir = path.join(releaseStoreRoot, dir, ...id.split("/"), version);
    await persistImmutableRelease(
      releaseDir,
      manifest,
      manifestBytes,
      artifactBytes,
      sourceEntry.bytes,
      `${id}@${version}`
    );
  }
}

async function persistImmutableRelease(releaseDir, manifestName, manifestBytes, artifactBytes, projectionBytes, releaseID) {
  const storedManifest = path.join(releaseDir, manifestName);
  const storedArtifact = path.join(releaseDir, "package.tar.gz");
  const storedProjection = path.join(releaseDir, "registry-entry.json");
  const manifestExists = await exists(storedManifest);
  const artifactExists = await exists(storedArtifact);
  if (manifestExists !== artifactExists) {
    throw new Error(`Incomplete immutable release ${releaseID}; restore or remove the partial release before publishing`);
  }
  if (manifestExists) {
    const [previousManifest, previousArtifact] = await Promise.all([
      fs.readFile(storedManifest),
      fs.readFile(storedArtifact)
    ]);
    if (!previousManifest.equals(manifestBytes) || !archivePayloadEquals(previousArtifact, artifactBytes)) {
      throw new Error(`Immutable release collision for ${releaseID}; increment the manifest version instead of replacing published bytes`);
    }
    if (await exists(storedProjection)) {
      const previousProjection = await fs.readFile(storedProjection);
      if (!previousProjection.equals(projectionBytes)) {
        throw new Error(`Immutable registry projection collision for ${releaseID}; publish a new version`);
      }
    } else {
      await fs.writeFile(storedProjection, projectionBytes, { flag: "wx" });
    }
    return;
  }
  await fs.mkdir(releaseDir, { recursive: true });
  await fs.writeFile(storedManifest, manifestBytes, { flag: "wx" });
  await fs.writeFile(storedArtifact, artifactBytes, { flag: "wx" });
  await fs.writeFile(storedProjection, projectionBytes, { flag: "wx" });
}

function archivePayloadEquals(previousArtifact, artifactBytes) {
  try {
    return gunzipSync(previousArtifact).equals(gunzipSync(artifactBytes));
  } catch {
    return false;
  }
}

async function validateRetainedReleases() {
  try {
    const { stdout } = await execFileAsync(
      "go",
      ["run", "./cmd/release-validator", "--root", releaseStoreRoot, "--json"],
      { cwd: validatorRoot, maxBuffer: 16 * 1024 * 1024 }
    );
    const result = JSON.parse(stdout);
    for (const [key, entry] of Object.entries(result.current_entries ?? {})) {
      currentReleaseEntries.set(key, entry);
    }
  } catch (error) {
    if (error.code === "ENOENT" && process.env.VERCEL === "1") {
      retainedReleaseValidationUnavailable = true;
      console.warn("Go release validator is unavailable in Vercel; using CI-validated committed release projections.");
      return;
    }
    const detail = error.stderr || error.stdout || error.message;
    throw new Error(`Retained release validation failed:\n${detail}`);
  }
}

async function restoreModuleReleases({ dir, manifest }) {
  const moduleReleaseRoot = path.join(releaseStoreRoot, dir);
  if (!(await exists(moduleReleaseRoot))) {
    return;
  }
  const manifestPaths = [];
  await walk(moduleReleaseRoot, async (filePath) => {
    if (path.basename(filePath) === manifest) {
      manifestPaths.push(filePath);
    }
  });
  for (const manifestPath of manifestPaths) {
    const releaseDir = path.dirname(manifestPath);
    const version = path.basename(releaseDir);
    const relativeID = path.relative(moduleReleaseRoot, path.dirname(releaseDir));
    const id = relativeID.split(path.sep).join("/");
    const manifestBytes = await fs.readFile(manifestPath);
    const manifestRaw = manifestBytes.toString("utf-8");
    if (extractScalar(manifestRaw, "id") !== id || extractScalar(manifestRaw, "version") !== version) {
      throw new Error(`Release path does not match manifest identity: ${manifestPath}`);
    }
    const artifactBytes = await fs.readFile(path.join(releaseDir, "package.tar.gz"));
    const registryEntry = JSON.parse(await fs.readFile(path.join(releaseDir, "registry-entry.json"), "utf-8"));
    const currentRegistryEntry = currentReleaseEntries.get(`${dir}:${id}@${version}`) ??
      (retainedReleaseValidationUnavailable ? projectCurrentCatalogEntry(registryEntry) : undefined);
    if (!currentRegistryEntry) {
      throw new Error(`Release validator did not return a current projection for ${id}@${version}`);
    }
    const key = `${dir}:${id}`;
    const releases = releaseCatalog.get(key) ?? [];
    releases.push({
      version,
      released_at: extractScalar(manifestRaw, "released_at"),
      manifest_url: `/release-manifests/${dir}/${id}/${version}.yaml`,
      manifest_sha256: createHash("sha256").update(manifestBytes).digest("hex"),
      artifact_url: `/artifacts/${id}/${version}.tar.gz`,
      sha256: createHash("sha256").update(artifactBytes).digest("hex"),
      registry_entry: registryEntry,
      current_registry_entry: currentRegistryEntry
    });
    releaseCatalog.set(key, releases);

    await publishReleaseFile(path.join(publicRoot, "artifacts", ...id.split("/"), `${version}.tar.gz`), artifactBytes, `${id}@${version}`);
    await publishReleaseFile(path.join(publicRoot, "release-manifests", dir, ...id.split("/"), `${version}.yaml`), manifestBytes, `${id}@${version} manifest`);
  }
}

async function publishReleaseFile(outputPath, bytes, releaseID) {
  if (await exists(outputPath)) {
    const previous = await fs.readFile(outputPath);
    if (!previous.equals(bytes)) {
      throw new Error(`Public release path collision for ${releaseID}`);
    }
    return;
  }
  await fs.mkdir(path.dirname(outputPath), { recursive: true });
  await fs.writeFile(outputPath, bytes, { flag: "wx" });
}

async function publishRegistryIndexes() {
  const registryRoot = path.join(repoRoot, "registry");
  const outRegistryRoot = path.join(publicRoot, "registry");
  await fs.mkdir(outRegistryRoot, { recursive: true });

  const indexes = [
    ["index.json", "skills"],
    ["skills-index.json", "skills"],
    ["agents-index.json", "agents"],
    ["tools-index.json", "tools-mcp"],
    ["plugins-index.json", "plugins"]
  ];
  for (const [fileName, moduleDir] of indexes) {
    const inPath = path.join(registryRoot, fileName);
    try {
      await fs.access(inPath);
    } catch {
      continue;
    }
    const index = JSON.parse(await fs.readFile(inPath, "utf-8"));
    const publishedEntries = [];
    const seen = new Set();
    for (const sourceEntry of index.skills ?? []) {
      const releases = releaseCatalog.get(`${moduleDir}:${sourceEntry.id}`) ?? [];
      if (!releases.some((release) => release.version === sourceEntry.latest)) {
        throw new Error(`No immutable release found for ${sourceEntry.id}@${sourceEntry.latest}`);
      }
      publishedEntries.push(materializeEntry(sourceEntry.latest, releases, sourceEntry.versions[0].artifact_url));
      seen.add(sourceEntry.id);
    }
    for (const [key, releases] of releaseCatalog) {
      const prefix = `${moduleDir}:`;
      if (!key.startsWith(prefix)) {
        continue;
      }
      const id = key.slice(prefix.length);
      if (seen.has(id)) {
        continue;
      }
      const latest = latestReleaseVersion(releases);
      publishedEntries.push(materializeEntry(latest, releases, "https://skills.ai-knowledge-hub.org/"));
    }
    publishedEntries.sort((left, right) => left.id.localeCompare(right.id));
    index.skills = publishedEntries;
    index.generated_at = publishedEntries.flatMap((entry) => entry.versions ?? []).reduce(
      (latest, release) => release.released_at > latest ? release.released_at : latest,
      "1970-01-01T00:00:00Z"
    );
    await fs.writeFile(
      path.join(outRegistryRoot, fileName),
      `${JSON.stringify(index, null, 2)}\n`
    );
  }
}

function materializeEntry(latest, releases, baseURL) {
  releases.sort((left, right) => compareSemverBuild(left.version, right.version));
  const latestRelease = releases.find((release) => release.version === latest);
  if (!latestRelease) {
    throw new Error(`Retained release catalog does not contain selected latest version ${latest}`);
  }
  return {
    ...latestRelease.current_registry_entry,
    latest,
    versions: releases.map(({ registry_entry: _registryEntry, current_registry_entry: _currentRegistryEntry, ...release }) => ({
      ...release,
      manifest_url: new URL(release.manifest_url, baseURL).href,
      artifact_url: new URL(release.artifact_url, baseURL).href
    }))
  };
}

function latestReleaseVersion(releases) {
  return [...releases]
    .sort((left, right) => compareSemverBuild(left.version, right.version))
    .at(-1).version;
}

function compareSemverBuild(left, right) {
  const leftVersion = parseSemver(left);
  const rightVersion = parseSemver(right);
  for (const key of ["major", "minor", "patch"]) {
    if (leftVersion[key] !== rightVersion[key]) {
      return leftVersion[key] - rightVersion[key];
    }
  }
  const prereleaseOrder = compareSemverIdentifiers(leftVersion.prerelease, rightVersion.prerelease, true);
  if (prereleaseOrder !== 0) {
    return prereleaseOrder;
  }
  return compareSemverIdentifiers(leftVersion.build, rightVersion.build, false);
}

function parseSemver(version) {
  const match = /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$/.exec(version);
  if (!match) {
    throw new Error(`Invalid semantic version in release store: ${version}`);
  }
  return {
    major: Number(match[1]),
    minor: Number(match[2]),
    patch: Number(match[3]),
    prerelease: match[4]?.split(".") ?? [],
    build: match[5]?.split(".") ?? []
  };
}

function compareSemverIdentifiers(left, right, releaseOutranksPrerelease) {
  if (releaseOutranksPrerelease && left.length === 0 && right.length > 0) return 1;
  if (releaseOutranksPrerelease && right.length === 0 && left.length > 0) return -1;
  for (let index = 0; index < Math.max(left.length, right.length); index += 1) {
    if (left[index] === undefined) return -1;
    if (right[index] === undefined) return 1;
    if (left[index] === right[index]) continue;
    const leftNumeric = /^[0-9]+$/.test(left[index]);
    const rightNumeric = /^[0-9]+$/.test(right[index]);
    if (leftNumeric && rightNumeric) return Number(left[index]) - Number(right[index]);
    if (leftNumeric !== rightNumeric) return leftNumeric ? -1 : 1;
    return left[index] < right[index] ? -1 : 1;
  }
  return 0;
}

export function projectCurrentCatalogEntry(storedEntry) {
  const entry = structuredClone(storedEntry);
  const hasPositiveCapability = (entry.capability_readiness ?? [])
    .some((capability) => isPositiveAvailability(capability.availability));
  if ((entry.usability?.availability === "usable-now" || hasPositiveCapability) && !hasFreshEvidence(entry)) {
    if (entry.usability?.availability === "usable-now") entry.usability.availability = "not-verified";
    entry.usability.source = "inferred";
    for (const capability of entry.capability_readiness ?? []) {
      if (isPositiveAvailability(capability.availability)) capability.availability = "not-verified";
    }
  }
  if (entry.deprecated) {
    entry.readiness = "deprecated";
    let demoted = false;
    if (isPositiveAvailability(entry.usability?.availability)) {
      entry.usability.availability = "not-verified";
      demoted = true;
    }
    for (const helper of entry.usability?.executable_helpers ?? []) {
      if (isPositiveAvailability(helper.availability)) {
        helper.availability = "not-verified";
        demoted = true;
      }
    }
    for (const capability of entry.capability_readiness ?? []) {
      if (isPositiveAvailability(capability.availability)) {
        capability.availability = "not-verified";
        demoted = true;
      }
    }
    if (demoted) entry.usability.source = "inferred";
  }
  return entry;
}

function hasFreshEvidence(entry) {
  if (!entry.verification?.evidence?.length) return false;
  const observedAt = Date.parse(entry.verification.last_verified_at);
  if (!Number.isFinite(observedAt)) return false;
  const now = Date.now();
  if (observedAt > now + 5 * 60 * 1000) return false;
  const executableKinds = new Set(["script", "cli", "mcp-server", "service", "orchestrator", "bundle"]);
  let lifetimeDays = executableKinds.has(entry.execution?.kind) ? 90 : 180;
  if (entry.authentication?.status && entry.authentication.status !== "none") lifetimeDays = 30;
  return now - observedAt <= lifetimeDays * 24 * 60 * 60 * 1000;
}

function isPositiveAvailability(availability) {
  return availability === "usable-now" || availability === "setup-required";
}

async function exists(filePath) {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
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

async function createPackageTarball(moduleDir, sourceDir, projection, version) {
  const members = await collectArchiveMembers(sourceDir);
  if (moduleDir === "plugins" && projection.artifact?.self_contained) {
    const components = await collectPluginClosure(projection, members);
    addArtifactMetadata(projection, version, components, members);
  }
  members.sort((left, right) => Buffer.compare(Buffer.from(left.path), Buffer.from(right.path)));
  for (let index = 1; index < members.length; index += 1) {
    if (members[index - 1].path === members[index].path) {
      throw new Error(`Duplicate package member while constructing closure: ${members[index].path}`);
    }
  }
  if (members.length === 0) {
    throw new Error(`Cannot publish an empty package: ${sourceDir}`);
  }
  const chunks = [];
  for (const member of members) {
    chunks.push(createTarHeader(member));
    if (member.data) {
      chunks.push(member.data);
      const padding = (512 - (member.data.length % 512)) % 512;
      if (padding > 0) {
        chunks.push(Buffer.alloc(padding));
      }
    }
  }
  chunks.push(Buffer.alloc(1024));
  const archive = gzipSync(Buffer.concat(chunks), { level: 9, mtime: 0 });
  archive.fill(0, 4, 8);
  archive[9] = 255;
  return archive;
}

async function collectPluginClosure(projection, members) {
  const queue = dependencyReferences(projection, true);
  const components = new Map();
  while (queue.length > 0) {
    const reference = queue.shift();
    const key = `${reference.module}:${reference.id}`;
    if (components.has(key)) {
      continue;
    }
    const sourceEntry = sourceEntries.get(key);
    if (!sourceEntry) {
      throw new Error(`Self-contained plugin is missing ${reference.module} dependency source ${reference.id}`);
    }
    const dependencyRoot = path.join(repoRoot, reference.module, ...reference.id.split("/"));
    if (!(await exists(dependencyRoot))) {
      throw new Error(`Self-contained plugin is missing ${reference.module} dependency source ${reference.id}`);
    }
    const prefix = `bundled/${reference.module}/${reference.id}`;
    await collectDirectory(dependencyRoot, prefix, members);
    const component = {
      module: reference.module,
      id: reference.id,
      version: sourceEntry.latest,
      prefix,
      projection: sourceEntry.projection
    };
    components.set(key, component);
    queue.push(...dependencyReferences(sourceEntry.projection, false));
  }
  return [...components.values()].sort((left, right) =>
    `${left.module}:${left.id}`.localeCompare(`${right.module}:${right.id}`)
  );
}

function dependencyReferences(projection, includePluginComposition) {
  const references = [];
  const add = (module, ids) => {
    for (const id of ids ?? []) {
      // Some legacy manifests list repository-relative schema or script paths
      // under dependencies.tools. Only registry identities belong in a
      // self-contained plugin closure; file dependencies remain part of their
      // owning package.
      if (id.includes("/") && sourceEntries.has(`${module}:${id}`)) {
        references.push({ module, id });
      }
    }
  };
  add("skills", projection.dependencies?.skills);
  add("agents", projection.dependencies?.agents);
  add("tools-mcp", projection.dependencies?.tools);
  if (includePluginComposition) {
    add("skills", projection.includes?.skills);
    add("agents", projection.includes?.agents);
    add("tools-mcp", projection.includes?.tools);
  }
  return references;
}

function addArtifactMetadata(projection, version, components, members) {
  const lockPath = projection.artifact?.dependency_lock;
  const checksumPath = projection.artifact?.checksums;
  const sbomPath = projection.artifact?.sbom;
  const provenancePath = projection.artifact?.provenance;
  if (!lockPath || !checksumPath || !sbomPath || !provenancePath) {
    throw new Error(`Self-contained plugin ${projection.id} must declare dependency_lock, checksums, sbom, and provenance paths`);
  }
  for (const generatedPath of [lockPath, checksumPath, sbomPath, provenancePath]) {
    const index = members.findIndex((member) => member.path === generatedPath);
    if (index >= 0) {
      members.splice(index, 1);
    }
  }

  const lockedComponents = components.map((component) => ({
    module: component.module,
    id: component.id,
    version: component.version,
    content_sha256: digestMemberSet(members, `${component.prefix}/`),
    manifest_path: `${component.prefix}/${manifestNameForModule(component.module)}`,
    dependencies: dependencyReferences(component.projection, false)
      .map((reference) => `${reference.module}:${reference.id}`)
      .sort()
  }));
  const rootReference = `plugins:${projection.id}@${version}`;
  const lock = {
    lock_version: "1.0",
    root: {
      module: "plugins",
      id: projection.id,
      version
    },
    components: lockedComponents
  };
  const lockBytes = jsonBytes(lock);
  members.push(fileMember(lockPath, lockBytes));

  const closureDigest = digestMemberSet(members);
  const componentReferences = new Map(lockedComponents.map((component) => [
    `${component.module}:${component.id}`,
    `${component.module}:${component.id}@${component.version}`
  ]));
  const rootDependencies = [...new Set(dependencyReferences(projection, true)
    .map((reference) => componentReferences.get(`${reference.module}:${reference.id}`)))]
    .filter(Boolean)
    .sort();
  const sbom = {
    bomFormat: "CycloneDX",
    specVersion: "1.5",
    version: 1,
    metadata: {
      component: { type: "application", "bom-ref": rootReference, name: projection.id, version: lock.root.version }
    },
    components: lockedComponents.map((component) => ({
      type: "library",
      "bom-ref": `${component.module}:${component.id}@${component.version}`,
      group: component.id.split("/")[0],
      name: component.id.split("/")[1],
      version: component.version,
      hashes: [{ alg: "SHA-256", content: component.content_sha256 }],
      properties: [{ name: "ai.skills.module", value: component.module }]
    })),
    dependencies: [
      { ref: rootReference, dependsOn: rootDependencies },
      ...lockedComponents.map((component) => ({
        ref: `${component.module}:${component.id}@${component.version}`,
        dependsOn: component.dependencies.map((dependency) => componentReferences.get(dependency)).sort()
      }))
    ]
  };
  members.push(fileMember(sbomPath, jsonBytes(sbom)));
  members.push(fileMember(provenancePath, jsonBytes({
    provenance_version: "1.0",
    subject: { module: "plugins", id: projection.id, version: lock.root.version, closure_sha256: closureDigest },
    builder: "ai-skills-guide/prepare-public-assets",
    materials: lockedComponents.map((component) => ({
      ref: `${component.module}:${component.id}@${component.version}`,
      sha256: component.content_sha256
    }))
  })));

  const checksumLines = members
    .filter((member) => member.type === "0" && member.path !== checksumPath)
    .sort((left, right) => Buffer.compare(Buffer.from(left.path), Buffer.from(right.path)))
    .map((member) => `${createHash("sha256").update(member.data).digest("hex")}  ${member.path}`);
  members.push(fileMember(checksumPath, Buffer.from(`${checksumLines.join("\n")}\n`)));
}

function manifestNameForModule(module) {
  return { skills: "skill.yaml", agents: "agent.yaml", "tools-mcp": "tool.yaml" }[module];
}

function digestMemberSet(members, prefix = "") {
  const lines = members
    .filter((member) => member.type === "0" && member.path.startsWith(prefix))
    .map((member) => {
      const relativePath = member.path.slice(prefix.length);
      return `${createHash("sha256").update(member.data).digest("hex")} ${member.mode.toString(8)} ${member.size} ${relativePath}`;
    })
    .sort();
  return createHash("sha256").update(`${lines.join("\n")}\n`).digest("hex");
}

function fileMember(relativePath, data) {
  return { path: relativePath, mode: 0o644, size: data.length, type: "0", data };
}

function jsonBytes(value) {
  return Buffer.from(`${JSON.stringify(value, null, 2)}\n`);
}

async function collectArchiveMembers(sourceDir) {
  const members = [];
  await collectDirectory(sourceDir, "", members);
  return members;
}

async function collectDirectory(absoluteDir, relativeDir, members) {
  const entries = await fs.readdir(absoluteDir, { withFileTypes: true });
  entries.sort((left, right) => Buffer.compare(Buffer.from(left.name), Buffer.from(right.name)));
  for (const entry of entries) {
    if (isTransientPackageEntry(entry.name)) {
      continue;
    }
    const absolutePath = path.join(absoluteDir, entry.name);
    const relativePath = relativeDir ? `${relativeDir}/${entry.name}` : entry.name;
    if (relativePath.includes("\\")) {
      throw new Error(`Package entry cannot contain a backslash: ${absolutePath}`);
    }
    const stat = await fs.lstat(absolutePath);
    if (stat.isDirectory()) {
      members.push({ path: `${relativePath}/`, mode: 0o755, size: 0, type: "5" });
      await collectDirectory(absolutePath, relativePath, members);
      continue;
    }
    if (!stat.isFile()) {
      throw new Error(`Unsupported package entry type: ${absolutePath}`);
    }
    const data = await fs.readFile(absolutePath);
    members.push({
      path: relativePath,
      mode: stat.mode & 0o111 ? 0o755 : 0o644,
      size: data.length,
      type: "0",
      data
    });
  }
}

function isTransientPackageEntry(name) {
  return transientPackageEntries.has(name) || name.endsWith(".pyc") || name.endsWith(".pyo");
}

function createTarHeader(member) {
  const header = Buffer.alloc(512);
  const { name, prefix } = splitTarPath(member.path);
  writeString(header, 0, 100, name);
  writeOctal(header, 100, 8, member.mode);
  writeOctal(header, 108, 8, 0);
  writeOctal(header, 116, 8, 0);
  writeOctal(header, 124, 12, member.size);
  writeOctal(header, 136, 12, 0);
  header.fill(0x20, 148, 156);
  writeString(header, 156, 1, member.type);
  writeString(header, 257, 6, "ustar\0");
  writeString(header, 263, 2, "00");
  writeString(header, 345, 155, prefix);
  const checksum = header.reduce((sum, value) => sum + value, 0);
  writeString(header, 148, 8, `${checksum.toString(8).padStart(6, "0")}\0 `);
  return header;
}

function splitTarPath(relativePath) {
  if (Buffer.byteLength(relativePath) <= 100) {
    return { name: relativePath, prefix: "" };
  }
  const directory = relativePath.endsWith("/");
  const corePath = directory ? relativePath.slice(0, -1) : relativePath;
  for (let separator = corePath.lastIndexOf("/"); separator > 0; separator = corePath.lastIndexOf("/", separator - 1)) {
    const prefix = corePath.slice(0, separator);
    const suffix = `${corePath.slice(separator + 1)}${directory ? "/" : ""}`;
    if (Buffer.byteLength(prefix) <= 155 && Buffer.byteLength(suffix) <= 100) {
      return { name: suffix, prefix };
    }
  }
  throw new Error(`Archive path exceeds USTAR limits: ${relativePath}`);
}

function writeString(buffer, offset, length, value) {
  const encoded = Buffer.from(value, "utf-8");
  if (encoded.length > length) {
    throw new Error(`Tar header value is too long: ${value}`);
  }
  encoded.copy(buffer, offset);
}

function writeOctal(buffer, offset, length, value) {
  const encoded = `${value.toString(8).padStart(length - 1, "0")}\0`;
  writeString(buffer, offset, length, encoded);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  await main();
}
