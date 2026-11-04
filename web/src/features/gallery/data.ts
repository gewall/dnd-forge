import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { galleryTag } from "@/features/characters/tags";
import { orNotFound, publicApi } from "@/lib/api/public";
import type { components } from "@/lib/api/schema";

export type GalleryPage = components["schemas"]["GalleryPage"];
export type GalleryItem = components["schemas"]["GalleryItemDTO"];

export const GALLERY_PAGE_SIZE = 24;

/**
 * One page of public characters with their latest visible portrait.
 * `before` is the previous page's nextBefore cursor.
 */
export async function getGalleryPage(before?: string, limit = GALLERY_PAGE_SIZE): Promise<GalleryPage> {
  "use cache";
  cacheLife("minutes");
  cacheTag(galleryTag);
  const client = publicApi();
  if (!client) return { items: [], nextBefore: null };
  const query = before ? { before, limit } : { limit };
  return orNotFound(await client.GET("/v1/gallery", { params: { query } }), "galeri") ?? { items: [], nextBefore: null };
}
