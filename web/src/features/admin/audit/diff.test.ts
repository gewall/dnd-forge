import { describe, expect, it } from "vitest";

import { describeDiff, formatValue } from "./diff";

describe("describeDiff", () => {
  it("has nothing to show for deletes", () => {
    expect(describeDiff(null)).toEqual({ kind: "none" });
    expect(describeDiff({})).toEqual({ kind: "none" });
  });

  it("reads [old, new] pairs as changes", () => {
    expect(describeDiff({ role: ["USER", "ADMIN"], portraitQuota: [null, 5] })).toEqual({
      kind: "changes",
      rows: [
        { key: "role", from: "USER", to: "ADMIN" },
        { key: "portraitQuota", from: null, to: 5 },
      ],
    });
  });

  it("reads saved input as values, marking nested data as long", () => {
    const view = describeDiff({ name: "Elf", speed: 30, traits: [{ name: "Darkvision" }, { name: "Fey" }] });
    expect(view.kind).toBe("values");
    if (view.kind !== "values") return;
    expect(view.rows[0]).toEqual({ key: "name", value: "Elf", long: false });
    expect(view.rows[1]).toEqual({ key: "speed", value: "30", long: false });
    expect(view.rows[2].long).toBe(true);
    expect(view.rows[2].value).toContain('"Darkvision"');
  });

  it("does not mistake a two-item list of objects for a change", () => {
    expect(describeDiff({ traits: [{ a: 1 }, { b: 2 }] }).kind).toBe("values");
  });

  it("marks long text as long", () => {
    const view = describeDiff({ content: "x".repeat(200) });
    expect(view.kind === "values" && view.rows[0].long).toBe(true);
  });
});

describe("formatValue", () => {
  it("names empty and boolean values", () => {
    expect(formatValue(null)).toBe("kosong");
    expect(formatValue("")).toBe("kosong");
    expect(formatValue(true)).toBe("ya");
    expect(formatValue(false)).toBe("tidak");
  });
});
