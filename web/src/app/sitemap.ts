import type { MetadataRoute } from "next";

import { getBackgrounds, getClasses, getRaces } from "@/features/gamedata/data";
import { getGuideCategories } from "@/features/guide/data";
import { siteUrl } from "@/lib/seo";

// Public pages only. Data comes from the same cached getters as the pages,
// so admin edits that expire those tags show up here too.
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const base = siteUrl();
  const [categories, races, classes, backgrounds] = await Promise.all([getGuideCategories(), getRaces(), getClasses(), getBackgrounds()]);

  const fixed: MetadataRoute.Sitemap = [
    { url: `${base}/`, changeFrequency: "weekly", priority: 1 },
    { url: `${base}/guide`, changeFrequency: "weekly", priority: 0.9 },
    { url: `${base}/builder`, changeFrequency: "monthly", priority: 0.8 },
    { url: `${base}/races`, changeFrequency: "monthly", priority: 0.7 },
    { url: `${base}/classes`, changeFrequency: "monthly", priority: 0.7 },
    { url: `${base}/backgrounds`, changeFrequency: "monthly", priority: 0.7 },
    { url: `${base}/gallery`, changeFrequency: "daily", priority: 0.5 },
  ];

  const guide = categories.flatMap((c) => [
    { url: `${base}/guide/${c.slug}`, changeFrequency: "weekly" as const, priority: 0.7 },
    ...(c.articles ?? []).map((a) => ({
      url: `${base}/guide/${c.slug}/${a.slug}`,
      lastModified: a.publishedAt ?? undefined,
      changeFrequency: "monthly" as const,
      priority: 0.8,
    })),
  ]);

  const entries = (path: string, items: { slug: string }[]) =>
    items.map((i) => ({ url: `${base}/${path}/${i.slug}`, changeFrequency: "monthly" as const, priority: 0.6 }));

  return [...fixed, ...guide, ...entries("races", races), ...entries("classes", classes), ...entries("backgrounds", backgrounds)];
}
