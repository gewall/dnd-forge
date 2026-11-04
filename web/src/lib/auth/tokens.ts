import { decodeJwt } from "jose";

import type { components } from "@/lib/api/schema";

export type SessionDTO = components["schemas"]["SessionDTO"];
export type UserDTO = components["schemas"]["UserDTO"];

export const ACCESS_COOKIE = "df_at";
export const REFRESH_COOKIE = "df_rt";
/** Set for a short while after a failed refresh so every request does not retry it. */
export const REFRESH_FAILED_COOKIE = "df_rf";
export const OAUTH_COOKIE = "df_oauth";

/** Refresh this many seconds before the access token expires. */
export const REFRESH_LEEWAY_SECONDS = 60;
export const REFRESH_FAILED_SECONDS = 30;

/**
 * True when the access token is missing, unreadable, or expires within the
 * leeway. Only decodes: the API verifies the signature on every call.
 */
export function needsRefresh(accessToken: string | undefined, nowSeconds: number): boolean {
  if (!accessToken) return true;
  try {
    const { exp } = decodeJwt(accessToken);
    return typeof exp !== "number" || exp - nowSeconds < REFRESH_LEEWAY_SECONDS;
  } catch {
    return true;
  }
}

/**
 * A same-site path to send the user to after signing in. Anything that could
 * leave the site (absolute URL, protocol-relative "//", backslash tricks)
 * falls back to the home page.
 */
export function safeNext(next: string | null | undefined, fallback = "/"): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.includes("\\")) {
    return fallback;
  }
  return next;
}

export interface CookieOptions {
  httpOnly: boolean;
  secure: boolean;
  sameSite: "lax";
  path: string;
  expires?: Date;
  maxAge?: number;
}

/** Writable cookie store: `cookies()` in actions/route handlers, or `response.cookies` in proxy. */
export interface CookieJar {
  set(name: string, value: string, options: CookieOptions): unknown;
  delete(name: string): unknown;
}

export function baseCookieOptions(): CookieOptions {
  return {
    httpOnly: true,
    // Browsers accept Secure cookies on http://localhost, but Safari does not.
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
  };
}

export function setSessionCookies(jar: CookieJar, session: SessionDTO): void {
  const base = baseCookieOptions();
  jar.set(ACCESS_COOKIE, session.accessToken, { ...base, expires: new Date(session.accessTokenExpiresAt) });
  jar.set(REFRESH_COOKIE, session.refreshToken, { ...base, expires: new Date(session.refreshTokenExpiresAt) });
  jar.delete(REFRESH_FAILED_COOKIE);
}

export function clearSessionCookies(jar: CookieJar): void {
  jar.delete(ACCESS_COOKIE);
  jar.delete(REFRESH_COOKIE);
  jar.delete(REFRESH_FAILED_COOKIE);
}
