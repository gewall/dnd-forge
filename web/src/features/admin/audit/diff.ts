// Audit diffs come in two shapes (see the API's audit callers):
//   changes: {"role": ["USER", "ADMIN"], "isBanned": [false, true]}  (old, new)
//   values:  the saved input, e.g. {"name": "Elf", "traits": [...]}
// Deletes have no diff.

export type Scalar = string | number | boolean | null;

export type DiffView =
  | { kind: "none" }
  | { kind: "changes"; rows: { key: string; from: Scalar; to: Scalar }[] }
  | { kind: "values"; rows: { key: string; value: string; long: boolean }[] };

const isScalar = (v: unknown): v is Scalar => v === null || ["string", "number", "boolean"].includes(typeof v);
const isPair = (v: unknown): v is [Scalar, Scalar] => Array.isArray(v) && v.length === 2 && v.every(isScalar);

/** Shows a value the way an admin would read it. */
export function formatValue(v: unknown): string {
  if (v === null || v === undefined) return "kosong";
  if (typeof v === "boolean") return v ? "ya" : "tidak";
  if (typeof v === "string") return v === "" ? "kosong" : v;
  if (typeof v === "number") return String(v);
  return JSON.stringify(v, null, 2);
}

export function describeDiff(diff: unknown): DiffView {
  if (diff === null || diff === undefined) return { kind: "none" };
  if (typeof diff !== "object" || Array.isArray(diff)) {
    return { kind: "values", rows: [{ key: "nilai", value: formatValue(diff), long: typeof diff === "object" }] };
  }
  const entries = Object.entries(diff as Record<string, unknown>);
  if (entries.length === 0) return { kind: "none" };
  if (entries.every(([, v]) => isPair(v))) {
    return { kind: "changes", rows: entries.map(([key, v]) => ({ key, from: (v as [Scalar, Scalar])[0], to: (v as [Scalar, Scalar])[1] })) };
  }
  return {
    kind: "values",
    rows: entries.map(([key, v]) => {
      const value = formatValue(v);
      return { key, value, long: !isScalar(v) || value.length > 120 || value.includes("\n") };
    }),
  };
}
