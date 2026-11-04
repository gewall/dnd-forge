import "server-only";

import { cacheLife, cacheTag } from "next/cache";

import { orNotFound, publicApi } from "@/lib/api/public";
import type { components } from "@/lib/api/schema";

export type Race = components["schemas"]["RaceDTO"];
export type Subrace = components["schemas"]["SubraceDTO"];
export type CharacterClass = components["schemas"]["ClassDTO"];
export type Background = components["schemas"]["BackgroundDTO"];
export type Trait = components["schemas"]["TraitDTO"];

// Admin edits call updateTag with these tags (F6); hours is the safety net.
export const gameTags = { races: "races", classes: "classes", backgrounds: "backgrounds" } as const;

export async function getRaces(): Promise<Race[]> {
  "use cache";
  cacheLife("hours");
  cacheTag(gameTags.races);
  const client = publicApi();
  if (!client) return [];
  return orNotFound(await client.GET("/v1/races"), "daftar ras")?.items ?? [];
}

export async function getRace(slug: string): Promise<Race | null> {
  "use cache";
  cacheLife("hours");
  cacheTag(gameTags.races);
  const client = publicApi();
  if (!client) return null;
  return orNotFound(await client.GET("/v1/races/{slug}", { params: { path: { slug } } }), `ras ${slug}`);
}

export async function getClasses(): Promise<CharacterClass[]> {
  "use cache";
  cacheLife("hours");
  cacheTag(gameTags.classes);
  const client = publicApi();
  if (!client) return [];
  return orNotFound(await client.GET("/v1/classes"), "daftar kelas")?.items ?? [];
}

export async function getClass(slug: string): Promise<CharacterClass | null> {
  "use cache";
  cacheLife("hours");
  cacheTag(gameTags.classes);
  const client = publicApi();
  if (!client) return null;
  return orNotFound(await client.GET("/v1/classes/{slug}", { params: { path: { slug } } }), `kelas ${slug}`);
}

export async function getBackgrounds(): Promise<Background[]> {
  "use cache";
  cacheLife("hours");
  cacheTag(gameTags.backgrounds);
  const client = publicApi();
  if (!client) return [];
  return orNotFound(await client.GET("/v1/backgrounds"), "daftar latar belakang")?.items ?? [];
}

export async function getBackground(slug: string): Promise<Background | null> {
  "use cache";
  cacheLife("hours");
  cacheTag(gameTags.backgrounds);
  const client = publicApi();
  if (!client) return null;
  return orNotFound(
    await client.GET("/v1/backgrounds/{slug}", { params: { path: { slug } } }),
    `latar belakang ${slug}`,
  );
}
