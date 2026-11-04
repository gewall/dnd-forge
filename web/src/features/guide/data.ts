import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { orNotFound, publicApi } from "@/lib/api/public";
import type { components } from "@/lib/api/schema";

export type CategoryContents = components["schemas"]["CategoryContentsDTO"];
export type ArticleSummary = components["schemas"]["ArticleSummaryDTO"];
export type ArticlePage = components["schemas"]["ArticlePageDTO"];
export type SearchHit = components["schemas"]["SearchHitDTO"];

export const guideTag = "guides";

/** Categories with their published articles, in display order. */
export async function getGuideCategories(): Promise<CategoryContents[]> {
  "use cache";
  cacheLife("hours");
  cacheTag(guideTag);
  const client = publicApi();
  if (!client) return [];
  return orNotFound(await client.GET("/v1/guides/categories"), "kategori panduan")?.items ?? [];
}

export async function getGuideCategory(slug: string): Promise<CategoryContents | null> {
  const categories = await getGuideCategories();
  return categories.find((c) => c.slug === slug) ?? null;
}

export async function getArticle(category: string, slug: string): Promise<ArticlePage | null> {
  "use cache";
  cacheLife("hours");
  cacheTag(guideTag);
  const client = publicApi();
  if (!client) return null;
  return orNotFound(
    await client.GET("/v1/guides/{category}/{slug}", { params: { path: { category, slug } } }),
    `artikel ${category}/${slug}`,
  );
}

export async function searchGuides(q: string): Promise<SearchHit[]> {
  "use cache";
  cacheLife("minutes");
  cacheTag(guideTag);
  const client = publicApi();
  if (!client) return [];
  const res = await client.GET("/v1/guides/search", { params: { query: { q, limit: 20 } } });
  return orNotFound(res, "pencarian panduan")?.items ?? [];
}
