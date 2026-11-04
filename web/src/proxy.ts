import { type NextRequest, NextResponse } from "next/server";

import { clientIpFrom } from "@/lib/api/request";
import { refreshSession, verifyAccessToken } from "@/lib/auth/edge";
import { routeArea } from "@/lib/auth/routes";
import {
  ACCESS_COOKIE,
  baseCookieOptions,
  clearSessionCookies,
  needsRefresh,
  REFRESH_COOKIE,
  REFRESH_FAILED_COOKIE,
  REFRESH_FAILED_SECONDS,
  type SessionDTO,
  setSessionCookies,
} from "@/lib/auth/tokens";

/**
 * The only place that refreshes the session. Pages, actions and route
 * handlers just read the access token cookie, which this has already renewed
 * for the current request. Route rules here are for redirects only: every
 * API endpoint checks the token and role again.
 */
export async function proxy(request: NextRequest) {
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  let accessToken = request.cookies.get(ACCESS_COOKIE)?.value;

  let refreshed: SessionDTO | undefined;
  let refreshFailed: number | undefined;

  if (
    refreshToken &&
    !request.cookies.has(REFRESH_FAILED_COOKIE) &&
    needsRefresh(accessToken, Math.floor(Date.now() / 1000))
  ) {
    const outcome = await refreshSession(refreshToken, clientIpFrom(request.headers));
    if (outcome.ok) {
      refreshed = outcome.session;
      accessToken = refreshed.accessToken;
      // Hand the new tokens to this request's render as well as the browser.
      request.cookies.set(ACCESS_COOKIE, refreshed.accessToken);
      request.cookies.set(REFRESH_COOKIE, refreshed.refreshToken);
    } else {
      refreshFailed = outcome.status;
    }
  }

  const redirect = await accessRedirect(request, accessToken);
  const response = redirect
    ? NextResponse.redirect(redirect)
    : NextResponse.next({ request: { headers: request.headers } });

  if (refreshed) {
    setSessionCookies(response.cookies, refreshed);
  } else if (refreshFailed === 403) {
    // Blocked account: the session is gone for good.
    clearSessionCookies(response.cookies);
  } else if (refreshFailed !== undefined) {
    // Do not clear cookies on 401: a parallel request may have just rotated the
    // token and set fresh cookies. Back off instead so we do not retry each request.
    response.cookies.set(REFRESH_FAILED_COOKIE, "1", { ...baseCookieOptions(), maxAge: REFRESH_FAILED_SECONDS });
  }
  return response;
}

async function accessRedirect(request: NextRequest, accessToken: string | undefined): Promise<URL | null> {
  const { pathname, search } = request.nextUrl;
  const area = routeArea(pathname);
  // Server actions POST to the page they live on; they authorize themselves.
  if (!area || (request.method !== "GET" && request.method !== "HEAD")) return null;

  const claims = accessToken ? await verifyAccessToken(accessToken) : null;

  if (area === "guest") {
    return claims ? new URL("/", request.url) : null;
  }
  if (!claims) {
    const login = new URL("/login", request.url);
    login.searchParams.set("next", pathname + search);
    return login;
  }
  if (area === "admin" && claims.role !== "ADMIN") {
    return new URL("/", request.url);
  }
  return null;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|icon.svg|favicon.ico|robots.txt|sitemap.xml).*)"],
};
