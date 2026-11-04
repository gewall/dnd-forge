import createClient from "openapi-fetch";

import type { paths } from "./schema";

export type ApiClient = ReturnType<typeof createClient<paths>>;

export interface ApiOptions {
  /** Access token of the signed-in user; omit for public endpoints. */
  token?: string;
  /** Visitor IP, used by the API for per-IP rate limits. */
  clientIp?: string;
}

export interface ApiConfig extends ApiOptions {
  baseUrl: string;
  internalKey: string;
}

/**
 * Typed client for the Go API that always sends X-Internal-Key. Kept free of
 * "server-only" so proxy.ts can use it; app code goes through `api()` in
 * client.ts instead.
 */
export function createApi({ baseUrl, internalKey, token, clientIp }: ApiConfig): ApiClient {
  const headers: Record<string, string> = { "X-Internal-Key": internalKey };
  if (clientIp) headers["X-Client-IP"] = clientIp;
  if (token) headers.Authorization = `Bearer ${token}`;
  return createClient<paths>({ baseUrl, headers });
}
