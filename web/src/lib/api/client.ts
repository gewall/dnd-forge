import "server-only";

import { serverEnv } from "@/lib/env";

import { type ApiClient, type ApiOptions, createApi } from "./base";

export type { ApiClient, ApiOptions };

/**
 * Typed client for the Go API. Every call carries X-Internal-Key; never call
 * the API with a bare fetch, or the internal key and client IP go missing.
 */
export function api(options: ApiOptions = {}): ApiClient {
  return createApi({
    baseUrl: serverEnv("API_BASE_URL"),
    internalKey: serverEnv("INTERNAL_API_KEY"),
    ...options,
  });
}
