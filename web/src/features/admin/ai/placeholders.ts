// Mirrors the API's placeholder syntax (internal/prompt/build.go): {name},
// where name is letters, digits, or underscores.
const placeholderRE = /\{(\w+)\}/g;

/** Placeholders in a template that its kind cannot use, each listed once. */
export function unknownPlaceholders(template: string, known: readonly string[]): string[] {
  const allowed = new Set(known);
  const bad = new Set<string>();
  for (const m of template.matchAll(placeholderRE)) if (!allowed.has(m[1])) bad.add(m[0]);
  return [...bad];
}

/** Placeholders a template actually uses, each listed once. */
export function usedPlaceholders(template: string): Set<string> {
  return new Set([...template.matchAll(placeholderRE)].map((m) => m[1]));
}

/** Inserts {name} at a selection, returning the new text and caret position. */
export function insertPlaceholder(text: string, start: number, end: number, name: string): { text: string; caret: number } {
  const token = `{${name}}`;
  return { text: text.slice(0, start) + token + text.slice(end), caret: start + token.length };
}
