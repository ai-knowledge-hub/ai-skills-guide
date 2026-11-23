import type { MetadataRoute } from "next";
import { familySites } from "@/lib/brand";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: { userAgent: "*", allow: "/" },
    sitemap: new URL("/sitemap.xml", familySites.skills.url).href
  };
}
