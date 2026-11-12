import { createReadStream } from "node:fs";
import path from "node:path";
import { pipeline } from "node:stream/promises";
import { createGunzip } from "node:zlib";
import { extract } from "tar-stream";
import type { RegistryEntry } from "./registry";

const maxPreviewFileBytes = 1024 * 1024;

export async function readReleaseFiles(
  module: "skills" | "agents" | "plugins" | "tools-mcp",
  { id, latest }: Pick<RegistryEntry, "id" | "latest">,
  names: string[],
) {
  // Read only requested files from the same retained archive users install.
  const releaseRoot = process.env.RELEASE_STORE_ROOT ?? path.resolve(process.cwd(), "..", "..", "releases");
  const archivePath = path.join(releaseRoot, module, id, latest, "package.tar.gz");
  const archive = extract();
  const requested = new Set(names);
  const files = new Map<string, string>();
  const decoder = new TextDecoder("utf-8", { fatal: true });

  archive.on("entry", (header, stream, next) => {
    if (!requested.has(header.name)) {
      stream.on("end", next);
      stream.resume();
      return;
    }

    if (files.has(header.name) || header.type !== "file" || !header.size || header.size > maxPreviewFileBytes) {
      next(new Error(`Invalid ${header.name} in ${id}@${latest}`));
      return;
    }

    const chunks: Buffer[] = [];
    stream.on("data", (chunk: Buffer) => chunks.push(chunk));
    stream.on("end", () => {
      try {
        files.set(header.name, decoder.decode(Buffer.concat(chunks)));
        next();
      } catch (error) {
        next(error);
      }
    });
  });

  try {
    await pipeline(createReadStream(archivePath), createGunzip(), archive);
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") return null;
    throw error;
  }
  return files;
}
