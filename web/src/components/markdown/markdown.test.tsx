import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { extractToc } from "@/features/guide/text";

import { Markdown } from "./markdown";

describe("Markdown", () => {
  it("gives headings the same ids as the table of contents", () => {
    const md = "## Melempar dadu\n\n### Advantage & *Disadvantage*\n\n## Melempar dadu\n";
    const html = renderToStaticMarkup(<Markdown source={md} />);
    for (const item of extractToc(md)) {
      expect(html).toContain(`id="${item.id}"`);
    }
  });

  it("does not render raw HTML from the source", () => {
    const html = renderToStaticMarkup(<Markdown source={'Halo <script>alert(1)</script> <img src=x onerror="x">'} />);
    // Shown as escaped text at most, never as elements.
    expect(html).not.toContain("<script");
    expect(html).not.toContain("<img");
  });

  it("opens external links in a new tab", () => {
    const html = renderToStaticMarkup(<Markdown source="[SRD](https://www.dndbeyond.com/srd)" />);
    expect(html).toContain('target="_blank"');
    expect(html).toContain('rel="noopener noreferrer"');
  });

  it("renders GFM tables", () => {
    const html = renderToStaticMarkup(<Markdown source={"| Dadu | Sisi |\n| --- | --- |\n| d20 | 20 |"} />);
    expect(html).toContain("<table");
    expect(html).toContain("<td");
  });
});
