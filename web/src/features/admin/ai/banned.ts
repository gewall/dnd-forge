// Mirrors internal/moderation: words are stored lowercase with whitespace
// collapsed, and match only as whole words or phrases ("ass" does not block
// "class").

/** Normalises one entry like the API does before storing it. */
export function normalizeWord(raw: string): string {
  return raw.replace(/\s+/g, " ").trim().toLowerCase();
}

/** Splits a pasted list on commas and new lines, dropping blanks and repeats. */
export function splitWords(raw: string): string[] {
  return [...new Set(raw.split(/[,\n]/).map(normalizeWord).filter(Boolean))];
}

const escape = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/** Banned words found in text, in list order. */
export function matchBanned(text: string, words: readonly string[]): string[] {
  const t = normalizeWord(text);
  if (!t) return [];
  return words.filter((w) => new RegExp(`(^|[^\\p{L}\\p{N}])${escape(w)}($|[^\\p{L}\\p{N}])`, "u").test(t));
}
