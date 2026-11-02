import type { CatalogueCompatibility } from "../lib/catalogueCompatibility";
import type { RegistryEntry } from "../lib/registry";

// Page assertions follow the current release; the retained-content checks
// independently cover historical notes that must stay in the review inventory.
export function latestToolPageNotes(
  entries: readonly Pick<RegistryEntry, "id" | "latest">[],
  notes: readonly CatalogueCompatibility[]
) {
  const latest = new Map(entries.map(entry => [entry.id, entry.latest]));
  return notes.filter(note => note.module === "tools" && latest.get(note.id) === note.version);
}
