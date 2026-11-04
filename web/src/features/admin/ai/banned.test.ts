import { describe, expect, it } from "vitest";

import { matchBanned, normalizeWord, splitWords } from "./banned";

describe("normalizeWord", () => {
  it("lowercases and collapses whitespace", () => {
    expect(normalizeWord("  Darah   Segar ")).toBe("darah segar");
  });
});

describe("splitWords", () => {
  it("splits on commas and new lines and drops repeats", () => {
    expect(splitWords("gore, Gore\nnsfw,, \n darah segar")).toEqual(["gore", "nsfw", "darah segar"]);
  });
});

describe("matchBanned", () => {
  const words = ["ass", "darah segar", "c++"];

  it("matches whole words only", () => {
    expect(matchBanned("A wizard class", words)).toEqual([]);
    expect(matchBanned("kick ASS!", words)).toEqual(["ass"]);
  });

  it("matches phrases across extra spaces", () => {
    expect(matchBanned("penuh  darah\tsegar.", words)).toEqual(["darah segar"]);
  });

  it("treats regex characters literally", () => {
    expect(matchBanned("belajar c++ dulu", words)).toEqual(["c++"]);
  });

  it("uses Unicode letters as word characters", () => {
    expect(matchBanned("éass", words)).toEqual([]);
  });
});
