import type { components } from "@/lib/api/schema";

type Race = components["schemas"]["RaceDTO"];
type CharacterClass = components["schemas"]["ClassDTO"];
type Background = components["schemas"]["BackgroundDTO"];
export type CharacterRequest = components["schemas"]["CharacterRequest"];
export type CharacterDTO = components["schemas"]["CharacterDTO"];

export interface GameData {
  races: Race[];
  classes: CharacterClass[];
  backgrounds: Background[];
}

export interface Draft {
  raceId?: string;
  subraceId?: string;
  classId?: string;
  backgroundId?: string;
  name: string;
  gender: string;
  appearance: string;
  isPublic: boolean;
}

export const emptyDraft: Draft = { name: "", gender: "", appearance: "", isPublic: false };

export const steps = [
  { key: "race", label: "Ras" },
  { key: "class", label: "Kelas" },
  { key: "background", label: "Latar belakang" },
  { key: "details", label: "Detail" },
  { key: "summary", label: "Ringkasan" },
] as const;

export type StepKey = (typeof steps)[number]["key"];

export const limits = { name: 60, gender: 30, appearance: 1000 } as const;

export function activeSubraces(race: Race | undefined) {
  return (race?.subraces ?? []).filter((s) => s.isActive);
}

/** Whether a step's choices are filled in well enough to move past it. */
export function stepComplete(step: StepKey, draft: Draft, data: GameData): boolean {
  switch (step) {
    case "race": {
      const race = data.races.find((r) => r.id === draft.raceId);
      if (!race) return false;
      const subs = activeSubraces(race);
      return subs.length === 0 || subs.some((s) => s.id === draft.subraceId);
    }
    case "class":
      return data.classes.some((c) => c.id === draft.classId);
    case "background":
      return data.backgrounds.some((b) => b.id === draft.backgroundId);
    case "details": {
      const name = draft.name.trim();
      return (
        name.length >= 1 &&
        name.length <= limits.name &&
        draft.gender.length <= limits.gender &&
        draft.appearance.length <= limits.appearance
      );
    }
    case "summary":
      return steps.slice(0, -1).every((s) => stepComplete(s.key, draft, data));
  }
}

/** Index of the first step that still needs input (the summary when all are done). */
export function firstIncompleteStep(draft: Draft, data: GameData): number {
  const i = steps.findIndex((s) => s.key !== "summary" && !stepComplete(s.key, draft, data));
  return i === -1 ? steps.length - 1 : i;
}

/** A step can be opened once every step before it is complete. */
export function canOpenStep(index: number, draft: Draft, data: GameData): boolean {
  return steps.slice(0, index).every((s) => stepComplete(s.key, draft, data));
}

/** Picking a race resets the subrace unless it belongs to that race. */
export function withRace(draft: Draft, raceId: string, data: GameData): Draft {
  const race = data.races.find((r) => r.id === raceId);
  const keepSub = activeSubraces(race).some((s) => s.id === draft.subraceId);
  return { ...draft, raceId, subraceId: keepSub ? draft.subraceId : undefined };
}

/** Applies ?race=&class=&background= slugs from encyclopedia links. Unknown slugs are ignored. */
export function applyPreselect(
  draft: Draft,
  data: GameData,
  slugs: { race?: string | null; class?: string | null; background?: string | null },
): Draft {
  let next = draft;
  const race = slugs.race ? data.races.find((r) => r.slug === slugs.race) : undefined;
  if (race) next = withRace(next, race.id, data);
  const cls = slugs.class ? data.classes.find((c) => c.slug === slugs.class) : undefined;
  if (cls) next = { ...next, classId: cls.id };
  const bg = slugs.background ? data.backgrounds.find((b) => b.slug === slugs.background) : undefined;
  if (bg) next = { ...next, backgroundId: bg.id };
  return next;
}

/**
 * Request body for POST/PUT. Backstory is never sent: on PUT that keeps the
 * stored backstory (e.g. one applied from a lore draft).
 */
export function toRequest(draft: Draft): CharacterRequest {
  const body: CharacterRequest = {
    name: draft.name.trim(),
    raceId: draft.raceId ?? "",
    classId: draft.classId ?? "",
    backgroundId: draft.backgroundId ?? "",
    gender: draft.gender.trim(),
    appearance: draft.appearance.trim(),
    isPublic: draft.isPublic,
  };
  if (draft.subraceId) body.subraceId = draft.subraceId;
  return body;
}

export function draftFromCharacter(c: CharacterDTO): Draft {
  return {
    raceId: c.race.id,
    subraceId: c.subrace?.id || undefined,
    classId: c.class.id,
    backgroundId: c.background.id,
    name: c.name,
    gender: c.gender,
    appearance: c.appearance,
    isPublic: c.isPublic,
  };
}
