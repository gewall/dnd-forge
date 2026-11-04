"use server";

import { refresh } from "next/cache";
import { z } from "zod";

import { type ActionResult, toResult } from "@/lib/api/errors";
import type { components } from "@/lib/api/schema";
import { requestApi } from "@/lib/auth/session";

type Portrait = components["schemas"]["PortraitDTO"];
type Lore = components["schemas"]["LoreDTO"];

const id = z.uuid();
const portraitInput = z.object({
  style: z.string().min(1).max(40),
  prompt: z.string().max(300).optional(),
  // Defaults mirror the API schema.
  aspectRatio: z.enum(["1:1", "3:4", "4:3"]).default("1:1"),
});
const loreInput = z.object({
  tone: z.string().min(1).max(40),
  length: z.string().max(20).default("medium"),
  language: z.string().max(10).default("id"),
  prompt: z.string().max(500).optional(),
});

function invalid(): Extract<ActionResult, { ok: false }> {
  return { ok: false, status: 422, message: "Isian belum lengkap.", fieldErrors: {} };
}

function withoutEmpty<T extends { prompt?: string }>(input: T): T {
  const prompt = input.prompt?.trim();
  return prompt ? { ...input, prompt } : { ...input, prompt: undefined };
}

/** Queues a portrait (202 + PENDING); the panel then polls /api/portraits/{id}. */
export async function requestPortrait(
  characterId: string,
  input: z.input<typeof portraitInput>,
): Promise<ActionResult<Portrait>> {
  const parsed = portraitInput.safeParse(input);
  if (!id.safeParse(characterId).success || !parsed.success) return invalid();
  return toResult(
    await (await requestApi()).POST("/v1/characters/{id}/portraits", {
      params: { path: { id: characterId } },
      body: withoutEmpty(parsed.data),
    }),
  );
}

/** Queues a lore draft; the panel polls /api/lore/{id}. */
export async function requestLore(characterId: string, input: z.input<typeof loreInput>): Promise<ActionResult<Lore>> {
  const parsed = loreInput.safeParse(input);
  if (!id.safeParse(characterId).success || !parsed.success) return invalid();
  return toResult(
    await (await requestApi()).POST("/v1/characters/{id}/lore", {
      params: { path: { id: characterId } },
      body: withoutEmpty(parsed.data),
    }),
  );
}

/** Copies a finished lore draft into the character's backstory. */
export async function applyLore(loreId: string): Promise<ActionResult> {
  if (!id.safeParse(loreId).success) return invalid();
  const result = toResult(await (await requestApi()).POST("/v1/lore/{id}/apply", { params: { path: { id: loreId } } }));
  if (result.ok) refresh();
  return result;
}
