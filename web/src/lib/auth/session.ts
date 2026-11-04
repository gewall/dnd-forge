import "server-only";

import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";
import { cache } from "react";

import { type ApiClient, api } from "@/lib/api/client";
import { clientIpFrom } from "@/lib/api/request";

import { ACCESS_COOKIE, type UserDTO } from "./tokens";

/** API client carrying the visitor's IP and, when signed in, their access token. */
export async function requestApi(): Promise<ApiClient> {
  const [jar, h] = await Promise.all([cookies(), headers()]);
  return api({ token: jar.get(ACCESS_COOKIE)?.value, clientIp: clientIpFrom(h) });
}

/**
 * The signed-in user from GET /v1/me, or null. Role always comes from the
 * API (not the token claim), so a role change shows up at once. Deduplicated
 * per request; call it inside a <Suspense> boundary.
 */
export const getCurrentUser = cache(async (): Promise<UserDTO | null> => {
  const jar = await cookies();
  if (!jar.get(ACCESS_COOKIE)?.value) return null;
  try {
    const { data, response } = await (await requestApi()).GET("/v1/me");
    return response.ok && data ? data : null;
  } catch {
    return null;
  }
});

/** The signed-in user, or a redirect to the login page that comes back to `next`. */
export async function requireUser(next: string): Promise<UserDTO> {
  const user = await getCurrentUser();
  if (!user) redirect(`/login?next=${encodeURIComponent(next)}`);
  return user;
}
