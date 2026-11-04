import "server-only";

import type { components } from "@/lib/api/schema";
import { requestApi } from "@/lib/auth/session";

export type Character = components["schemas"]["CharacterDTO"];

/** The signed-in user's characters, newest first as the API returns them. */
export async function listCharacters(): Promise<Character[] | null> {
  const { data, response } = await (await requestApi()).GET("/v1/characters");
  return response.ok ? (data?.items ?? []) : null;
}

/** One of the user's characters, or null (missing, someone else's, or a malformed id). */
export async function getCharacter(id: string): Promise<Character | null> {
  const { data, response } = await (await requestApi()).GET("/v1/characters/{id}", { params: { path: { id } } });
  if (response.ok && data) return data;
  if (response.status === 404 || response.status === 422) return null;
  throw new Error(`karakter ${id}: API menjawab ${response.status}`);
}

/** "High Elf Wizard": subrace when there is one, then class. */
export function characterTitle(c: Pick<Character, "race" | "subrace" | "class">): string {
  return `${c.subrace?.name || c.race.name} ${c.class.name}`;
}
