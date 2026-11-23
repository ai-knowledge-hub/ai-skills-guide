import type { MetadataRoute } from "next";
import { familySites } from "@/lib/brand";
import { loadRegistry } from "@/lib/registry";

const catalogueRoutes = [
  ["skills", "/skills"],
  ["agents", "/agents"],
  ["plugins", "/plugins"],
  ["tools", "/tools-mcp"]
] as const;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const cataloguePages = await Promise.all(catalogueRoutes.map(async ([module, route]) => {
    const registry = await loadRegistry(module);
    return [route, ...registry.skills.map(({ id }) => `${route}/${id}`)];
  }));

  return ["/", ...cataloguePages.flat()].map((path) => ({
    url: new URL(path, familySites.skills.url).href
  }));
}
