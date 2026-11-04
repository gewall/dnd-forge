import { describe, expect, it } from "vitest";

import { codeChallenge, randomToken } from "./pkce";
import { routeArea } from "./routes";
import {
  ACCESS_COOKIE,
  type CookieJar,
  needsRefresh,
  REFRESH_COOKIE,
  REFRESH_FAILED_COOKIE,
  safeNext,
  setSessionCookies,
} from "./tokens";

function fakeJwt(payload: object): string {
  const enc = (v: object) => Buffer.from(JSON.stringify(v)).toString("base64url");
  return `${enc({ alg: "EdDSA", typ: "JWT" })}.${enc(payload)}.sig`;
}

describe("needsRefresh", () => {
  const now = 1_800_000_000;
  it("refreshes when the token is missing or unreadable", () => {
    expect(needsRefresh(undefined, now)).toBe(true);
    expect(needsRefresh("not-a-jwt", now)).toBe(true);
    expect(needsRefresh(fakeJwt({ sub: "u" }), now)).toBe(true);
  });
  it("refreshes within a minute of expiry", () => {
    expect(needsRefresh(fakeJwt({ exp: now + 59 }), now)).toBe(true);
    expect(needsRefresh(fakeJwt({ exp: now - 5 }), now)).toBe(true);
  });
  it("keeps a token with time left", () => {
    expect(needsRefresh(fakeJwt({ exp: now + 600 }), now)).toBe(false);
  });
});

describe("safeNext", () => {
  it.each([
    ["/characters/abc?tab=lore", "/characters/abc?tab=lore"],
    [null, "/"],
    ["https://evil.example", "/"],
    ["//evil.example", "/"],
    ["/\\evil.example", "/"],
    ["characters", "/"],
  ])("%s → %s", (input, expected) => {
    expect(safeNext(input)).toBe(expected);
  });
  it("uses the given fallback", () => {
    expect(safeNext(undefined, "/builder")).toBe("/builder");
  });
});

describe("setSessionCookies", () => {
  it("writes both tokens with their expiry and clears the backoff marker", () => {
    const set = new Map<string, { value: string; expires?: Date; httpOnly: boolean }>();
    const deleted: string[] = [];
    const jar: CookieJar = {
      set: (name, value, o) => set.set(name, { value, expires: o.expires, httpOnly: o.httpOnly }),
      delete: (name) => deleted.push(name),
    };
    setSessionCookies(jar, {
      accessToken: "at",
      accessTokenExpiresAt: "2026-11-04T12:15:00+07:00",
      refreshToken: "rt",
      refreshTokenExpiresAt: "2026-12-04T12:00:00+07:00",
      user: { id: "u", email: "a@b.c", name: "", role: "USER", createdAt: "2026-11-04T12:00:00+07:00", dailyQuota: null },
    });
    expect(set.get(ACCESS_COOKIE)).toEqual({ value: "at", expires: new Date("2026-11-04T05:15:00Z"), httpOnly: true });
    expect(set.get(REFRESH_COOKIE)?.value).toBe("rt");
    expect(deleted).toEqual([REFRESH_FAILED_COOKIE]);
  });
});

describe("pkce", () => {
  it("matches the RFC 7636 appendix B example", async () => {
    expect(await codeChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")).toBe(
      "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
    );
  });
  it("makes URL-safe random tokens", () => {
    const t = randomToken();
    expect(t).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(randomToken()).not.toBe(t);
  });
});

describe("routeArea", () => {
  it.each([
    ["/admin", "admin"],
    ["/admin/users", "admin"],
    ["/characters", "user"],
    ["/characters/123/edit", "user"],
    ["/profile", "user"],
    ["/login", "guest"],
    ["/register", "guest"],
    ["/builder", null],
    ["/administrator", null],
    ["/", null],
  ])("%s → %s", (path, area) => {
    expect(routeArea(path)).toBe(area);
  });
});
