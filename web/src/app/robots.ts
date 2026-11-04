import type { MetadataRoute } from "next";

import { siteUrl } from "@/lib/seo";

export default function robots(): MetadataRoute.Robots {
  const base = siteUrl();
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        // Signed-in areas, auth endpoints, and the BFF are not content.
        disallow: ["/admin", "/api/", "/auth/", "/characters", "/profile", "/login", "/register", "/guide/search"],
      },
    ],
    sitemap: `${base}/sitemap.xml`,
    host: base,
  };
}
