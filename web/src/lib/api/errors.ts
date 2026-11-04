import type { components } from "./schema";

// Every field is optional on the wire: "type" only has a default in the schema.
export type Problem = Partial<components["schemas"]["ErrorModel"]>;

export type ActionResult<T = void> =
  | { ok: true; data: T }
  | {
      ok: false;
      status: number;
      message: string;
      fieldErrors: Record<string, string>;
    };

const fallbackMessages: Record<number, string> = {
  401: "Sesi berakhir. Masuk lagi untuk melanjutkan.",
  403: "Kamu tidak punya akses ke halaman ini.",
  404: "Data tidak ditemukan.",
  429: "Terlalu banyak permintaan. Coba lagi sebentar lagi.",
  503: "Layanan sedang tidak tersedia. Hubungi admin.",
};

const genericMessage = "Terjadi kesalahan di server. Coba lagi.";

/**
 * Field name for a Huma error location: "body.name" → "name",
 * "body.items[2].tags" → "items.2.tags". Non-body locations are kept whole.
 */
export function fieldFromLocation(location: string): string {
  const path = location.startsWith("body.") ? location.slice("body.".length) : location;
  return path.replace(/\[(\d+)\]/g, ".$1");
}

/** Converts an API problem+json body into a failed ActionResult. */
export function toFailure(status: number, problem?: Problem | null): Extract<ActionResult, { ok: false }> {
  const fieldErrors: Record<string, string> = {};
  for (const e of problem?.errors ?? []) {
    if (!e.location || !e.message) continue;
    const field = fieldFromLocation(e.location);
    fieldErrors[field] ??= e.message;
  }

  // Huma's 422 detail is the generic "validation failed"; field messages say more.
  const detail = problem?.detail?.trim();
  const useDetail = detail && !(status === 422 && detail === "validation failed");
  const message =
    (useDetail ? detail : undefined) ??
    (status === 422 ? "Periksa lagi isian yang ditandai." : undefined) ??
    fallbackMessages[status] ??
    genericMessage;

  return { ok: false, status, message, fieldErrors };
}

/**
 * Wraps an openapi-fetch result: data on success, a failure carrying the
 * API's Indonesian message otherwise.
 */
export function toResult<T>(res: {
  data?: T;
  error?: unknown;
  response: Response;
}): ActionResult<T> {
  if (res.response.ok) {
    return { ok: true, data: res.data as T };
  }
  const problem = isProblem(res.error) ? res.error : undefined;
  return toFailure(res.response.status, problem);
}

function isProblem(value: unknown): value is Problem {
  return typeof value === "object" && value !== null;
}
