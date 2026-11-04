import { describe, expect, it } from "vitest";

import { extractToc, parseHeadline } from "./text";

describe("extractToc", () => {
  it("lists h2/h3 with rehype-slug ids and skips code fences", () => {
    const md = [
      "# Judul artikel",
      "## Melempar dadu",
      "Teks.",
      "### Advantage & *Disadvantage*",
      "```md",
      "## bukan heading",
      "```",
      "## Melempar dadu",
      "#### Terlalu dalam",
      "## [Ability score](/races) dan `modifier`",
    ].join("\n");
    expect(extractToc(md)).toEqual([
      { id: "melempar-dadu", text: "Melempar dadu", depth: 2 },
      { id: "advantage--disadvantage", text: "Advantage & Disadvantage", depth: 3 },
      { id: "melempar-dadu-1", text: "Melempar dadu", depth: 2 },
      { id: "ability-score-dan-modifier", text: "Ability score dan modifier", depth: 2 },
    ]);
  });

  it("returns nothing for an article without sections", () => {
    expect(extractToc("Hanya paragraf.")).toEqual([]);
  });
});

describe("parseHeadline", () => {
  it("splits on mark tags", () => {
    expect(parseHeadline("Napas <mark>naga</mark> menyembur <mark>api</mark>.")).toEqual([
      { text: "Napas ", match: false },
      { text: "naga", match: true },
      { text: " menyembur ", match: false },
      { text: "api", match: true },
      { text: ".", match: false },
    ]);
  });

  it("keeps any other markup as plain text", () => {
    expect(parseHeadline('<img src=x onerror="alert(1)"> <mark>x</mark>')).toEqual([
      { text: '<img src=x onerror="alert(1)"> ', match: false },
      { text: "x", match: true },
    ]);
  });
});
