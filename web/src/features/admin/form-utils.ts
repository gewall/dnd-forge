/** "" → null (use the default), otherwise the integer typed in; NaN stays for validation. */
export function optionalInt(value: string): number | null {
  const v = value.trim();
  return v === "" ? null : Number(v);
}

/** Slug the way the API accepts it: lowercase a–z, 0–9 and single dashes. */
export function slugify(text: string): string {
  return text
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64);
}
