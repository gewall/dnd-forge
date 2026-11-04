import { afterEach, describe, expect, it, vi } from "vitest";

import { defaultImage, metaDescription, pageMetadata, siteUrl } from "./seo";

afterEach(() => vi.unstubAllEnvs());

describe("siteUrl", () => {
  it("drops a trailing slash", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://dnd.gewall.my.id/");
    expect(siteUrl()).toBe("https://dnd.gewall.my.id");
  });
});

describe("metaDescription", () => {
  it("keeps short text and collapses whitespace", () => {
    expect(metaDescription("  Elf   tinggal\ndi hutan. ")).toBe("Elf tinggal di hutan.");
  });

  it("cuts long text at a word boundary", () => {
    const text = "kata ".repeat(60);
    const out = metaDescription(text, 50)!;
    expect(out.length).toBeLessThanOrEqual(50);
    expect(out.endsWith("kata…")).toBe(true);
  });

  it("returns undefined for empty text", () => {
    expect(metaDescription("   ")).toBeUndefined();
  });
});

describe("pageMetadata", () => {
  it("sets canonical and Open Graph, with a large card when there is an image", () => {
    const m = pageMetadata({ title: "Elf", description: "Ras tua.", path: "/races/elf", image: "https://s3.example/elf.webp" });
    expect(m.alternates?.canonical).toBe("/races/elf");
    expect(m.openGraph).toMatchObject({ title: "Elf", url: "/races/elf", images: [{ url: "https://s3.example/elf.webp", alt: "Elf" }] });
    expect(m.twitter).toMatchObject({ card: "summary_large_image" });
  });

  it("falls back to the site image when the page has none", () => {
    const m = pageMetadata({ title: "Elf", path: "/races/elf" });
    expect(m.openGraph).toMatchObject({ images: [defaultImage] });
    expect(m.twitter).toMatchObject({ card: "summary_large_image", images: ["/opengraph-image"] });
  });
});
