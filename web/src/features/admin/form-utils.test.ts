import { describe, expect, it } from "vitest";

import { optionalInt, slugify } from "./form-utils";

describe("optionalInt", () => {
  it("treats empty as default (null)", () => {
    expect(optionalInt("")).toBeNull();
    expect(optionalInt("  ")).toBeNull();
  });
  it("parses numbers and leaves garbage as NaN for validation", () => {
    expect(optionalInt(" 12 ")).toBe(12);
    expect(optionalInt("abc")).toBeNaN();
  });
});

describe("slugify", () => {
  it.each([
    ["Dasar Aturan", "dasar-aturan"],
    ["  Combat & Kondisi!  ", "combat-kondisi"],
    ["Élfe Haute", "elfe-haute"],
    ["--x--", "x"],
  ])("%s → %s", (input, slug) => {
    expect(slugify(input)).toBe(slug);
  });
  it("caps at 64 characters", () => {
    expect(slugify("a".repeat(80))).toHaveLength(64);
  });
});
