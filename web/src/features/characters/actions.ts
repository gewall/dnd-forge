"use server";

import { refresh, updateTag } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { type ActionResult, toResult } from "@/lib/api/errors";
import { requestApi } from "@/lib/auth/session";

import { type CharacterRequest, limits } from "../builder/draft";
import { galleryTag } from "./tags";

// Shape check before the API call; the API validates everything again.
const requestSchema = z.object({
  name: z.string().trim().min(1).max(limits.name),
  raceId: z.uuid(),
  subraceId: z.uuid().optional(),
  classId: z.uuid(),
  backgroundId: z.uuid(),
  gender: z.string().max(limits.gender).optional(),
  appearance: z.string().max(limits.appearance).optional(),
  isPublic: z.boolean().optional(),
});

const idSchema = z.uuid();

function invalidInput(): Extract<ActionResult, { ok: false }> {
  return { ok: false, status: 422, message: "Data karakter tidak lengkap. Periksa lagi tiap langkah.", fieldErrors: {} };
}

export async function createCharacter(input: CharacterRequest): Promise<ActionResult<{ id: string }>> {
  const parsed = requestSchema.safeParse(input);
  if (!parsed.success) return invalidInput();

  const result = toResult(await (await requestApi()).POST("/v1/characters", { body: parsed.data }));
  if (!result.ok) return result;
  if (parsed.data.isPublic) updateTag(galleryTag);
  return { ok: true, data: { id: result.data.id } };
}

export async function updateCharacter(id: string, input: CharacterRequest): Promise<ActionResult> {
  const parsed = requestSchema.safeParse(input);
  if (!idSchema.safeParse(id).success || !parsed.success) return invalidInput();

  // No backstory in the body: the API keeps the stored one.
  const result = toResult(
    await (await requestApi()).PUT("/v1/characters/{id}", { params: { path: { id } }, body: parsed.data }),
  );
  if (!result.ok) return result;
  updateTag(galleryTag);
  redirect(`/characters/${id}`);
}

export async function setCharacterPublic(id: string, isPublic: boolean): Promise<ActionResult> {
  if (!idSchema.safeParse(id).success) return invalidInput();
  const client = await requestApi();

  const current = toResult(await client.GET("/v1/characters/{id}", { params: { path: { id } } }));
  if (!current.ok) return current;
  const c = current.data;

  // PUT takes the full state; backstory stays out so it is preserved.
  const body: CharacterRequest = {
    name: c.name,
    raceId: c.race.id,
    classId: c.class.id,
    backgroundId: c.background.id,
    gender: c.gender,
    appearance: c.appearance,
    isPublic,
  };
  if (c.subrace?.id) body.subraceId = c.subrace.id;

  const result = toResult(await client.PUT("/v1/characters/{id}", { params: { path: { id } }, body }));
  if (!result.ok) return result;
  updateTag(galleryTag);
  refresh();
  return { ok: true, data: undefined };
}

export async function deleteCharacter(id: string): Promise<ActionResult> {
  if (!idSchema.safeParse(id).success) return invalidInput();
  const result = toResult(await (await requestApi()).DELETE("/v1/characters/{id}", { params: { path: { id } } }));
  if (!result.ok && result.status !== 404) return result;
  updateTag(galleryTag);
  redirect("/characters");
}
