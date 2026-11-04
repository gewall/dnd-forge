import type { FieldValues, Path, UseFormSetError } from "react-hook-form";

import type { ActionResult } from "@/lib/api/errors";

/**
 * Puts a failed action's field errors on the form and returns the message to
 * show above it (null when every error landed on a field).
 */
export function applyFailure<T extends FieldValues>(
  result: Extract<ActionResult, { ok: false }>,
  fields: readonly Path<T>[],
  setError: UseFormSetError<T>,
): string {
  let placed = false;
  for (const [field, message] of Object.entries(result.fieldErrors)) {
    if ((fields as readonly string[]).includes(field)) {
      setError(field as Path<T>, { message });
      placed = true;
    }
  }
  return placed && result.status === 422 ? "" : result.message;
}
