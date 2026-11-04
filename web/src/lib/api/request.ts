/**
 * Visitor IP from proxy headers. Vercel puts the client first in
 * x-forwarded-for; x-real-ip is the fallback.
 */
export function clientIpFrom(headers: Pick<Headers, "get">): string | undefined {
  const forwarded = headers.get("x-forwarded-for");
  const first = forwarded?.split(",")[0]?.trim();
  if (first) return first;
  return headers.get("x-real-ip")?.trim() || undefined;
}
