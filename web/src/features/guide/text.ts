import GithubSlugger from "github-slugger";

export interface TocItem {
  id: string;
  text: string;
  depth: 2 | 3;
}

/** Inline Markdown to plain text, close to what rehype-slug sees in the rendered heading. */
function plainText(inline: string): string {
  return inline
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/`([^`]*)`/g, "$1")
    .replace(/(\*\*|__|\*|_|~~)/g, "")
    .trim();
}

/**
 * Table of contents (h2/h3) for an article. Ids follow rehype-slug: one
 * github-slugger over every heading in order, so duplicates get -1, -2.
 */
export function extractToc(markdown: string): TocItem[] {
  const slugger = new GithubSlugger();
  const items: TocItem[] = [];
  let fence: string | null = null;

  for (const line of markdown.split(/\r?\n/)) {
    const fenceMatch = line.match(/^\s*(```+|~~~+)/);
    if (fenceMatch) {
      const marker = fenceMatch[1][0];
      if (fence === null) fence = marker;
      else if (fence === marker) fence = null;
      continue;
    }
    if (fence !== null) continue;

    const heading = line.match(/^(#{1,6})\s+(.+?)\s*#*\s*$/);
    if (!heading) continue;
    const text = plainText(heading[2]);
    const id = slugger.slug(text);
    const depth = heading[1].length;
    if (depth === 2 || depth === 3) items.push({ id, text, depth });
  }
  return items;
}

export interface HeadlinePart {
  text: string;
  match: boolean;
}

/**
 * Splits a search headline on its <mark> tags. Everything else stays text,
 * so React escapes it; no HTML from the API is ever injected.
 */
export function parseHeadline(headline: string): HeadlinePart[] {
  const parts: HeadlinePart[] = [];
  const re = /<mark>([\s\S]*?)<\/mark>/g;
  let last = 0;
  for (let m = re.exec(headline); m !== null; m = re.exec(headline)) {
    if (m.index > last) parts.push({ text: headline.slice(last, m.index), match: false });
    if (m[1]) parts.push({ text: m[1], match: true });
    last = m.index + m[0].length;
  }
  if (last < headline.length) parts.push({ text: headline.slice(last), match: false });
  return parts;
}
