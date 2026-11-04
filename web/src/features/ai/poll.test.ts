import { describe, expect, it } from "vitest";

import { optionLabel } from "./labels";
import { POLL_MAX_TRIES, pollStep, upsertById } from "./poll";

describe("pollStep", () => {
  it("finishes on SUCCEEDED and FAILED", () => {
    expect(pollStep({ status: "SUCCEEDED" }, 1)).toEqual({ done: true, item: { status: "SUCCEEDED" } });
    expect(pollStep({ status: "FAILED" }, 1).done).toBe(true);
  });

  it("keeps waiting while PENDING and gives up after the last try", () => {
    expect(pollStep({ status: "PENDING" }, 1)).toEqual({ done: false, timedOut: false });
    expect(pollStep({ status: "PENDING" }, POLL_MAX_TRIES)).toEqual({ done: false, timedOut: true });
  });
});

describe("upsertById", () => {
  const a = { id: "a", v: 1 };
  const b = { id: "b", v: 1 };

  it("puts a new item first", () => {
    expect(upsertById([a], b)).toEqual([b, a]);
  });

  it("replaces an existing item in place", () => {
    expect(upsertById([b, a], { id: "a", v: 2 })).toEqual([b, { id: "a", v: 2 }]);
  });
});

describe("optionLabel", () => {
  it("uses the Indonesian label, then the API description, then the id", () => {
    expect(optionLabel("style", "oil-painting")).toBe("Lukisan minyak");
    expect(optionLabel("tone", "cosmic", "Cosmic horror")).toBe("Cosmic horror");
    expect(optionLabel("length", "epic-length")).toBe("epic-length");
  });
});
