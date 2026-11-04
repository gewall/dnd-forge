import { describe, expect, it } from "vitest";

import {
  applyPreselect,
  canOpenStep,
  type Draft,
  draftFromCharacter,
  emptyDraft,
  firstIncompleteStep,
  type GameData,
  stepComplete,
  toRequest,
  withRace,
} from "./draft";

const sub = (id: string, raceId: string, isActive = true) => ({
  id,
  raceId,
  slug: id,
  name: id,
  description: "",
  traits: [],
  visualPrompt: "",
  isActive,
});

const data: GameData = {
  races: [
    { id: "r-elf", slug: "elf", name: "Elf", description: "", traits: [], visualPrompt: "", sortOrder: 1, isActive: true, subraces: [sub("s-high", "r-elf"), sub("s-old", "r-elf", false)] },
    { id: "r-human", slug: "human", name: "Human", description: "", traits: [], visualPrompt: "", sortOrder: 2, isActive: true, subraces: [] },
  ],
  classes: [{ id: "c-wiz", slug: "wizard", name: "Wizard", description: "", features: [], hitDie: 6, primaryAbility: "Intelligence", visualPrompt: "", sortOrder: 1, isActive: true }],
  backgrounds: [{ id: "b-sage", slug: "sage", name: "Sage", description: "", skillProficiencies: [], visualPrompt: "", sortOrder: 1, isActive: true }],
};

const full: Draft = { ...emptyDraft, raceId: "r-human", classId: "c-wiz", backgroundId: "b-sage", name: "Arin" };

describe("stepComplete", () => {
  it("needs an active subrace when the race has one", () => {
    expect(stepComplete("race", { ...emptyDraft, raceId: "r-elf" }, data)).toBe(false);
    expect(stepComplete("race", { ...emptyDraft, raceId: "r-elf", subraceId: "s-old" }, data)).toBe(false);
    expect(stepComplete("race", { ...emptyDraft, raceId: "r-elf", subraceId: "s-high" }, data)).toBe(true);
    expect(stepComplete("race", { ...emptyDraft, raceId: "r-human" }, data)).toBe(true);
  });

  it("rejects ids that are not in the game data", () => {
    expect(stepComplete("class", { ...emptyDraft, classId: "c-gone" }, data)).toBe(false);
  });

  it("checks name and field lengths on the details step", () => {
    expect(stepComplete("details", { ...full, name: "   " }, data)).toBe(false);
    expect(stepComplete("details", { ...full, name: "x".repeat(61) }, data)).toBe(false);
    expect(stepComplete("details", { ...full, appearance: "x".repeat(1001) }, data)).toBe(false);
    expect(stepComplete("details", full, data)).toBe(true);
  });
});

describe("step navigation", () => {
  it("finds the first step that needs input", () => {
    expect(firstIncompleteStep(emptyDraft, data)).toBe(0);
    expect(firstIncompleteStep({ ...full, backgroundId: undefined }, data)).toBe(2);
    expect(firstIncompleteStep(full, data)).toBe(4);
  });

  it("opens a step only after the ones before it", () => {
    expect(canOpenStep(0, emptyDraft, data)).toBe(true);
    expect(canOpenStep(2, { ...emptyDraft, raceId: "r-human" }, data)).toBe(false);
    expect(canOpenStep(4, full, data)).toBe(true);
  });
});

describe("withRace and applyPreselect", () => {
  it("drops a subrace from another race", () => {
    const d = withRace({ ...emptyDraft, raceId: "r-elf", subraceId: "s-high" }, "r-human", data);
    expect(d).toMatchObject({ raceId: "r-human", subraceId: undefined });
  });

  it("maps slugs to ids and ignores unknown ones", () => {
    const d = applyPreselect(emptyDraft, data, { race: "elf", class: "wizard", background: "nope" });
    expect(d).toMatchObject({ raceId: "r-elf", classId: "c-wiz" });
    expect(d.backgroundId).toBeUndefined();
  });
});

describe("toRequest", () => {
  it("trims text, omits backstory, and sends subrace only when set", () => {
    const body = toRequest({ ...full, name: "  Arin  ", gender: " perempuan " });
    expect(body).toEqual({
      name: "Arin",
      raceId: "r-human",
      classId: "c-wiz",
      backgroundId: "b-sage",
      gender: "perempuan",
      appearance: "",
      isPublic: false,
    });
    expect("backstory" in body).toBe(false);
    expect(toRequest({ ...full, raceId: "r-elf", subraceId: "s-high" }).subraceId).toBe("s-high");
  });

  it("round-trips a saved character without a subrace", () => {
    const ref = (id: string) => ({ id, name: id, slug: id });
    const d = draftFromCharacter({
      id: "x",
      name: "Arin",
      race: ref("r-human"),
      subrace: { id: "", name: "", slug: "" },
      class: ref("c-wiz"),
      background: ref("b-sage"),
      gender: "",
      appearance: "",
      backstory: "panjang",
      isPublic: true,
      createdAt: "",
      updatedAt: "",
    });
    expect(d.subraceId).toBeUndefined();
    expect(toRequest(d).subraceId).toBeUndefined();
  });
});
