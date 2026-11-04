import "server-only";

import { type ApiClient, api } from "./client";

/**
 * Client for public, cacheable reads. Returns null when the API is not
 * configured (CI builds have no secrets); callers then render empty content
 * instead of failing the prerender. Deploys always have the env set.
 */
export function publicApi(): ApiClient | null {
  if (!process.env.API_BASE_URL || !process.env.INTERNAL_API_KEY) return null;
  return api();
}

/**
 * Result of a public read: data, null for 404, and an exception for anything
 * else so a transient API failure is never stored in the cache.
 */
export function orNotFound<T>(res: { data?: T; response: Response }, what: string): T | null {
  if (res.response.status === 404) return null;
  if (!res.response.ok || res.data === undefined) {
    throw new Error(`${what}: API menjawab ${res.response.status}`);
  }
  return res.data;
}
