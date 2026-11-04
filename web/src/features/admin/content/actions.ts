"use server";

import { refresh, updateTag } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { gameTags } from "@/features/gamedata/data";
import { guideTag } from "@/features/guide/data";
import { type ActionResult, toResult } from "@/lib/api/errors";
import type { components } from "@/lib/api/schema";
import { requestApi } from "@/lib/auth/session";

type S = components["schemas"];

// The API validates every field and returns per-field errors; these actions
// only check ids, call it, and refresh the public caches that show the data.

const uuid = z.uuid();
function badId(): Extract<ActionResult, { ok: false }> {
  return { ok: false, status: 404, message: "Data tidak ditemukan.", fieldErrors: {} };
}

/** Runs a create/update, expires public caches, and either redirects (create) or refreshes (update). */
async function save<T extends { id: string }>(
  call: Promise<{ data?: T; error?: unknown; response: Response }>,
  tags: string[],
  afterCreate?: (created: T) => string,
): Promise<ActionResult> {
  const result = toResult(await call);
  if (!result.ok) return result;
  for (const t of tags) updateTag(t);
  if (afterCreate) redirect(afterCreate(result.data));
  refresh();
  return { ok: true, data: undefined };
}

async function remove(
  call: Promise<{ error?: unknown; response: Response }>,
  tags: string[],
  to?: string,
): Promise<ActionResult> {
  const result = toResult(await call);
  if (!result.ok) return result;
  for (const t of tags) updateTag(t);
  if (to) redirect(to);
  refresh();
  return { ok: true, data: undefined };
}

// Guide categories

export async function saveCategory(id: string | null, body: S["CategoryRequest"]): Promise<ActionResult> {
  const api = await requestApi();
  if (id === null) return save(api.POST("/v1/admin/guide-categories", { body }), [guideTag]);
  if (!uuid.safeParse(id).success) return badId();
  return save(api.PUT("/v1/admin/guide-categories/{id}", { params: { path: { id } }, body }), [guideTag]);
}

export async function deleteCategory(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return badId();
  return remove((await requestApi()).DELETE("/v1/admin/guide-categories/{id}", { params: { path: { id } } }), [guideTag]);
}

// Guide articles

export async function saveArticle(id: string | null, body: S["ArticleRequest"]): Promise<ActionResult> {
  const api = await requestApi();
  if (id === null) return save(api.POST("/v1/admin/guides", { body }), [guideTag], (a) => `/admin/guides/${a.id}`);
  if (!uuid.safeParse(id).success) return badId();
  return save(api.PUT("/v1/admin/guides/{id}", { params: { path: { id } }, body }), [guideTag]);
}

export async function deleteArticle(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return badId();
  return remove((await requestApi()).DELETE("/v1/admin/guides/{id}", { params: { path: { id } } }), [guideTag], "/admin/guides");
}

// Races and subraces

export async function saveRace(id: string | null, body: S["RaceRequest"]): Promise<ActionResult> {
  const api = await requestApi();
  if (id === null) return save(api.POST("/v1/admin/races", { body }), [gameTags.races], (r) => `/admin/races/${r.id}`);
  if (!uuid.safeParse(id).success) return badId();
  return save(api.PUT("/v1/admin/races/{id}", { params: { path: { id } }, body }), [gameTags.races]);
}

export async function deleteRace(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return badId();
  return remove((await requestApi()).DELETE("/v1/admin/races/{id}", { params: { path: { id } } }), [gameTags.races], "/admin/races");
}

export async function saveSubrace(raceId: string, id: string | null, body: S["SubraceRequest"]): Promise<ActionResult> {
  const api = await requestApi();
  if (id === null) {
    if (!uuid.safeParse(raceId).success) return badId();
    return save(api.POST("/v1/admin/races/{id}/subraces", { params: { path: { id: raceId } }, body }), [gameTags.races]);
  }
  if (!uuid.safeParse(id).success) return badId();
  return save(api.PUT("/v1/admin/subraces/{id}", { params: { path: { id } }, body }), [gameTags.races]);
}

export async function deleteSubrace(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return badId();
  return remove((await requestApi()).DELETE("/v1/admin/subraces/{id}", { params: { path: { id } } }), [gameTags.races]);
}

// Classes

export async function saveClass(id: string | null, body: S["ClassRequest"]): Promise<ActionResult> {
  const api = await requestApi();
  if (id === null) return save(api.POST("/v1/admin/classes", { body }), [gameTags.classes], (c) => `/admin/classes/${c.id}`);
  if (!uuid.safeParse(id).success) return badId();
  return save(api.PUT("/v1/admin/classes/{id}", { params: { path: { id } }, body }), [gameTags.classes]);
}

export async function deleteClass(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return badId();
  return remove((await requestApi()).DELETE("/v1/admin/classes/{id}", { params: { path: { id } } }), [gameTags.classes], "/admin/classes");
}

// Backgrounds

export async function saveBackground(id: string | null, body: S["BackgroundRequest"]): Promise<ActionResult> {
  const api = await requestApi();
  if (id === null)
    return save(api.POST("/v1/admin/backgrounds", { body }), [gameTags.backgrounds], (b) => `/admin/backgrounds/${b.id}`);
  if (!uuid.safeParse(id).success) return badId();
  return save(api.PUT("/v1/admin/backgrounds/{id}", { params: { path: { id } }, body }), [gameTags.backgrounds]);
}

export async function deleteBackground(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return badId();
  return remove(
    (await requestApi()).DELETE("/v1/admin/backgrounds/{id}", { params: { path: { id } } }),
    [gameTags.backgrounds],
    "/admin/backgrounds",
  );
}
