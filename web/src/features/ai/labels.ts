// Indonesian labels for option ids from the API. Unknown ids fall back to the
// API description, then the id itself, so a new option still shows up.
const styleLabels: Record<string, string> = {
  "oil-painting": "Lukisan minyak",
  watercolor: "Cat air",
  anime: "Anime",
  realistic: "Realistis",
  comic: "Komik",
  "ink-sketch": "Sketsa tinta",
};

const toneLabels: Record<string, string> = {
  epic: "Epik",
  heroic: "Heroik",
  dark: "Kelam",
  tragic: "Tragis",
  humor: "Jenaka",
};

const lengthLabels: Record<string, string> = { short: "Pendek", medium: "Sedang", long: "Panjang" };
const languageLabels: Record<string, string> = { id: "Bahasa Indonesia", en: "English" };

const tables = { style: styleLabels, tone: toneLabels, length: lengthLabels, language: languageLabels };

export function optionLabel(kind: keyof typeof tables, id: string, description?: string): string {
  return tables[kind][id] ?? (description?.trim() || id);
}
