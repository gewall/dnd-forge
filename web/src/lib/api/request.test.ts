import { describe, expect, it } from "vitest";

import { clientIpFrom } from "./request";

describe("clientIpFrom", () => {
  it("takes the first x-forwarded-for entry", () => {
    const h = new Headers({ "x-forwarded-for": " 203.0.113.7 , 10.0.0.1" });
    expect(clientIpFrom(h)).toBe("203.0.113.7");
  });

  it("falls back to x-real-ip", () => {
    expect(clientIpFrom(new Headers({ "x-real-ip": "198.51.100.2" }))).toBe("198.51.100.2");
  });

  it("returns undefined without proxy headers", () => {
    expect(clientIpFrom(new Headers())).toBeUndefined();
  });
});
