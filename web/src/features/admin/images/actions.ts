"use server";

import { refresh, updateTag } from "next/cache";
import { z } from "zod";

import { galleryTag } from "@/features/characters/tags";
import { type ActionResult, toResult } from "@/lib/api/errors";
import { requestApi } from "@/lib/auth/session";

const uuid = z.uuid();
const notFound: Extract<ActionResult, { ok: false }> = { ok: false, status: 404, message: "Portrait tidak ditemukan.", fieldErrors: {} };

// Hidden or deleted portraits leave the public gallery, so its cache expires too.
async function done(call: Promise<{ error?: unknown; response: Response }>): Promise<ActionResult> {
  const result = toResult(await call);
  if (!result.ok) return result;
  updateTag(galleryTag);
  refresh();
  return { ok: true, data: undefined };
}

export async function setImageHidden(id: string, isHidden: boolean): Promise<ActionResult> {
  if (!uuid.safeParse(id).success || typeof isHidden !== "boolean") return notFound;
  return done((await requestApi()).PATCH("/v1/admin/images/{id}", { params: { path: { id } }, body: { isHidden } }));
}

/** Deletes the files; the portrait stays on record, hidden for good. */
export async function deleteImageFiles(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return notFound;
  return done((await requestApi()).DELETE("/v1/admin/images/{id}", { params: { path: { id } } }));
}
