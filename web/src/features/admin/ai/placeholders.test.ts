import { describe, expect, it } from "vitest";

import { insertPlaceholder, unknownPlaceholders, usedPlaceholders } from "./placeholders";

describe("unknownPlaceholders", () => {
  it("lists names the kind cannot use, once each", () => {
    expect(unknownPlaceholders("{race} {klass} {klass} {name}", ["race", "name"])).toEqual(["{klass}"]);
  });

  it("ignores text that is not a placeholder", () => {
    expect(unknownPlaceholders("{ race } {} {race-x}", ["race"])).toEqual([]);
  });
});

describe("usedPlaceholders", () => {
  it("collects names in the template", () => {
    expect([...usedPlaceholders("{a}, {b}\n{a}")]).toEqual(["a", "b"]);
  });
});

describe("insertPlaceholder", () => {
  it("replaces the selection and moves the caret after the token", () => {
    expect(insertPlaceholder("Halo dunia", 5, 10, "name")).toEqual({ text: "Halo {name}", caret: 11 });
  });

  it("inserts at the caret when nothing is selected", () => {
    expect(insertPlaceholder("ab", 1, 1, "x")).toEqual({ text: "a{x}b", caret: 4 });
  });
});
