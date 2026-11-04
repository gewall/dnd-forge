import "server-only";

import createClient from "openapi-fetch";

import { serverEnv } from "@/lib/env";
import type { paths } from "./schema";

export type ApiClient = ReturnType<typeof createClient<paths>>;

export interface ApiOptions {
  /** Access token of the signed-in user; omit for public endpoints. */
  token?: string;
  /** Visitor IP, used by the API for per-IP rate limits. */
  clientIp?: string;
}

/**
 * Typed client for the Go API. Every call carries X-Internal-Key; never call
 * the API with a bare fetch, or the internal key and client IP go missing.
 */
export function api({ token, clientIp }: ApiOptions = {}): ApiClient {
  const headers: Record<string, string> = {
    "X-Internal-Key": serverEnv("INTERNAL_API_KEY"),
  };
  if (clientIp) headers["X-Client-IP"] = clientIp;
  if (token) headers.Authorization = `Bearer ${token}`;

  return createClient<paths>({
    baseUrl: serverEnv("API_BASE_URL"),
    headers,
  });
}
