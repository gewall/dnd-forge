import "server-only";

import { redirect } from "next/navigation";

import { requestApi } from "@/lib/auth/session";

export { requestApi as adminApi };

/**
 * Data of an admin read. 401/403 go back to the gate (session ended or role
 * revoked mid-visit), 404 and 422 (malformed id) become null, anything else throws.
 */
export function adminData<T>(res: { data?: T; response: Response }, what: string): T | null {
  const { status } = res.response;
  if (status === 401) redirect("/login?next=/admin");
  if (status === 403) redirect("/");
  if (status === 404 || status === 422) return null;
  if (!res.response.ok || res.data === undefined) throw new Error(`${what}: API menjawab ${status}`);
  return res.data;
}

/** First value of a search param. */
export function param(value: string | string[] | undefined): string | undefined {
  const v = Array.isArray(value) ? value[0] : value;
  return v?.trim() || undefined;
}
