import { type NextRequest, NextResponse } from "next/server";

import { codeChallenge, randomToken } from "@/lib/auth/pkce";
import { baseCookieOptions, OAUTH_COOKIE, safeNext } from "@/lib/auth/tokens";

const AUTHORIZE_URL = "https://accounts.google.com/o/oauth2/v2/auth";

/** Starts Google sign-in: stores state + PKCE verifier, then redirects to Google. */
export async function GET(request: NextRequest) {
  const clientId = process.env.GOOGLE_CLIENT_ID;
  const redirectUri = process.env.GOOGLE_REDIRECT_URL;
  if (!clientId || !redirectUri) {
    return NextResponse.redirect(new URL("/login?error=google_unavailable", request.url));
  }

  const state = randomToken();
  const verifier = randomToken(48);
  const next = safeNext(request.nextUrl.searchParams.get("next"), "/characters");

  const url = new URL(AUTHORIZE_URL);
  url.search = new URLSearchParams({
    client_id: clientId,
    redirect_uri: redirectUri,
    response_type: "code",
    scope: "openid email profile",
    state,
    code_challenge: await codeChallenge(verifier),
    code_challenge_method: "S256",
    prompt: "select_account",
  }).toString();

  const response = NextResponse.redirect(url);
  response.cookies.set(OAUTH_COOKIE, JSON.stringify({ state, verifier, next }), {
    ...baseCookieOptions(),
    path: "/auth/google",
    maxAge: 600,
  });
  return response;
}
