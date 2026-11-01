import type { RegistryEntry } from "./registry";

export function projectCurrentCatalogEntry<T extends RegistryEntry>(storedEntry: T): T;
