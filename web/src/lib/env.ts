import "server-only";

// Server-only configuration. Values are read lazily so `next build` can run
// without secrets; a missing value fails the request that needs it.
type ServerEnvKey = "API_BASE_URL" | "INTERNAL_API_KEY";

export function serverEnv(key: ServerEnvKey): string {
  const value = process.env[key];
  if (!value) {
    throw new Error(`env ${key} belum diisi`);
  }
  return value;
}
