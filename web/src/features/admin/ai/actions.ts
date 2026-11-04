"use server";

import { refresh } from "next/cache";
import { z } from "zod";

import { type ActionResult, toResult } from "@/lib/api/errors";
import type { components } from "@/lib/api/schema";
import { requestApi } from "@/lib/auth/session";

type S = components["schemas"];

// Nothing here is shown on public pages, so a mutation only refreshes the
// admin page it came from. The API validates every field.

const uuid = z.uuid();
const notFound: Extract<ActionResult, { ok: false }> = { ok: false, status: 404, message: "Data tidak ditemukan.", fieldErrors: {} };

async function done<T>(call: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<ActionResult> {
  const result = toResult(await call);
  if (!result.ok) return result;
  refresh();
  return { ok: true, data: undefined };
}

function withId(id: string, fn: (id: string) => Promise<ActionResult>): Promise<ActionResult> {
  return uuid.safeParse(id).success ? fn(id) : Promise.resolve(notFound);
}

// Providers

export async function createProvider(body: S["ProviderRequest"]): Promise<ActionResult> {
  return done((await requestApi()).POST("/v1/admin/ai/providers", { body }));
}

/** An empty apiKey keeps the stored key. */
export async function updateProvider(id: string, body: S["ProviderUpdateRequest"]): Promise<ActionResult> {
  const { apiKey, ...rest } = body;
  const payload = apiKey?.trim() ? { ...rest, apiKey: apiKey.trim() } : rest;
  return withId(id, async (id) => done((await requestApi()).PUT("/v1/admin/ai/providers/{id}", { params: { path: { id } }, body: payload })));
}

export async function activateProvider(id: string): Promise<ActionResult> {
  return withId(id, async (id) => done((await requestApi()).POST("/v1/admin/ai/providers/{id}/activate", { params: { path: { id } } })));
}

export async function deleteProvider(id: string): Promise<ActionResult> {
  return withId(id, async (id) => done((await requestApi()).DELETE("/v1/admin/ai/providers/{id}", { params: { path: { id } } })));
}

/** Calls the provider with its stored key; changes nothing. */
export async function testProvider(id: string): Promise<ActionResult> {
  if (!uuid.safeParse(id).success) return notFound;
  const result = toResult(await (await requestApi()).POST("/v1/admin/ai/providers/{id}/test", { params: { path: { id } } }));
  return result.ok ? { ok: true, data: undefined } : result;
}

export async function providerModels(id: string): Promise<ActionResult<string[]>> {
  if (!uuid.safeParse(id).success) return notFound;
  const result = toResult(await (await requestApi()).GET("/v1/admin/ai/providers/{id}/models", { params: { path: { id } } }));
  return result.ok ? { ok: true, data: [...(result.data.items ?? [])].sort() } : result;
}

// Prompt templates

export async function createTemplate(body: S["TemplateRequest"]): Promise<ActionResult> {
  return done((await requestApi()).POST("/v1/admin/prompt-templates", { body }));
}

export async function activateTemplate(id: string): Promise<ActionResult> {
  return withId(id, async (id) => done((await requestApi()).POST("/v1/admin/prompt-templates/{id}/activate", { params: { path: { id } } })));
}

export async function deleteTemplate(id: string): Promise<ActionResult> {
  return withId(id, async (id) => done((await requestApi()).DELETE("/v1/admin/prompt-templates/{id}", { params: { path: { id } } })));
}

// Banned words

/** Adds each word in turn; stops at the first one the API rejects. */
export async function addBannedWords(words: string[]): Promise<ActionResult<number>> {
  if (words.length === 0 || words.length > 100) return { ok: false, status: 422, message: "Isi 1–100 kata sekaligus.", fieldErrors: {} };
  const api = await requestApi();
  let added = 0;
  for (const word of words) {
    const result = toResult(await api.POST("/v1/admin/banned-words", { body: { word } }));
    if (!result.ok) {
      if (added > 0) refresh();
      return { ...result, message: `“${word}”: ${result.message}` };
    }
    added++;
  }
  refresh();
  return { ok: true, data: added };
}

export async function removeBannedWord(word: string): Promise<ActionResult> {
  if (!word || word.length > 60) return notFound;
  return done((await requestApi()).DELETE("/v1/admin/banned-words/{word}", { params: { path: { word } } }));
}
