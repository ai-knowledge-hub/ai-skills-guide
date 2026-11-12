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
  { optionalNames = [], onUnavailable }: { optionalNames?: string[]; onUnavailable?: (name: string) => void } = {},
) {
  // Read only requested files from the same retained archive users install.
  const releaseRoot = process.env.RELEASE_STORE_ROOT ?? path.resolve(process.cwd(), "..", "..", "releases");
  const archivePath = path.join(releaseRoot, module, id, latest, "package.tar.gz");
  const archive = extract();
  const requested = new Set(names);
  const optional = new Set(optionalNames);
  const seen = new Set<string>();
  const files = new Map<string, string>();
  const decoder = new TextDecoder("utf-8", { fatal: true });

  archive.on("entry", (header, stream, next) => {
    stream.on("error", (error) => archive.destroy(error));
    if (!requested.has(header.name)) {
      stream.on("end", next);
      stream.resume();
      return;
    }

    if (seen.has(header.name)) {
      next(new Error(`Invalid ${header.name} in ${id}@${latest}`));
      return;
    }
    seen.add(header.name);

    if (header.type !== "file" || !header.size || header.size > maxPreviewFileBytes) {
      if (optional.has(header.name)) {
        // Optional files can be valid release content without being suitable
        // for a bounded text preview. Drain them without buffering their bytes.
        onUnavailable?.(header.name);
        stream.on("end", next);
        stream.resume();
      } else {
        next(new Error(`Invalid ${header.name} in ${id}@${latest}`));
      }
      return;
    }

    const chunks: Buffer[] = [];
    stream.on("data", (chunk: Buffer) => chunks.push(chunk));
    stream.on("end", () => {
      try {
        files.set(header.name, decoder.decode(Buffer.concat(chunks)));
        next();
      } catch (error) {
        if (optional.has(header.name)) onUnavailable?.(header.name);
        next(optional.has(header.name) ? undefined : error);
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
