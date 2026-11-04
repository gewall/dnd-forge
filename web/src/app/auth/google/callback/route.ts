import { type NextRequest, NextResponse } from "next/server";

import { api } from "@/lib/api/client";
import { clientIpFrom } from "@/lib/api/request";
import { OAUTH_COOKIE, safeNext, setSessionCookies } from "@/lib/auth/tokens";

interface OAuthState {
  state: string;
  verifier: string;
  next: string;
}

function readState(raw: string | undefined): OAuthState | null {
  if (!raw) return null;
  try {
    const v = JSON.parse(raw) as Partial<OAuthState>;
    return typeof v.state === "string" && typeof v.verifier === "string"
      ? { state: v.state, verifier: v.verifier, next: safeNext(v.next, "/characters") }
      : null;
  } catch {
    return null;
  }
}

const errorByStatus: Record<number, string> = {
  403: "google_unverified",
  503: "google_unavailable",
};

/** Google redirects here with ?code&state; trade the code for an API session. */
export async function GET(request: NextRequest) {
  const params = request.nextUrl.searchParams;
  const saved = readState(request.cookies.get(OAUTH_COOKIE)?.value);

  const fail = (error: string) => {
    const response = NextResponse.redirect(new URL(`/login?error=${error}`, request.url));
    response.cookies.delete({ name: OAUTH_COOKIE, path: "/auth/google" });
    return response;
  };

  if (params.get("error")) return fail("google_denied");
  const code = params.get("code");
  if (!code || !saved || params.get("state") !== saved.state) return fail("google_state");

  let status = 502;
  try {
    const { data, error, response } = await api({ clientIp: clientIpFrom(request.headers) }).POST(
      "/v1/auth/google",
      { body: { code, codeVerifier: saved.verifier } },
    );
    if (response.ok && data) {
      const ok = NextResponse.redirect(new URL(saved.next, request.url));
      setSessionCookies(ok.cookies, data);
      ok.cookies.delete({ name: OAUTH_COOKIE, path: "/auth/google" });
      return ok;
    }
    status = response.status;
    // 403 is either an unverified Google email or a blocked account.
    if (status === 403 && error?.detail?.includes("blokir")) return fail("google_blocked");
  } catch {}
  return fail(errorByStatus[status] ?? "google_failed");
}
