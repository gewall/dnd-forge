import type { Metadata } from "next";

export const siteName = "DnD Forge";

/** The generated site image (app/opengraph-image.tsx). */
export const defaultImage = { url: "/opengraph-image", width: 1200, height: 630, alt: "DnD Forge: panduan bermain dan pembuat karakter" };

/** Absolute site origin; NEXT_PUBLIC_SITE_URL in production. */
export function siteUrl(): string {
  return (process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000").replace(/\/+$/, "");
}

/** Shortens text for a meta description without cutting a word in half. */
export function metaDescription(text: string | undefined, max = 160): string | undefined {
  const t = text?.replace(/\s+/g, " ").trim();
  if (!t) return undefined;
  if (t.length <= max) return t;
  const cut = t.slice(0, max - 1);
  const space = cut.lastIndexOf(" ");
  return `${(space > max * 0.6 ? cut.slice(0, space) : cut).replace(/[\s.,;:]+$/, "")}…`;
}

/** Metadata for a public detail page: canonical URL plus Open Graph. */
export function pageMetadata({
  title,
  description,
  path,
  image,
  type = "website",
}: {
  title: string;
  description?: string;
  path: string;
  image?: string;
  type?: "website" | "article";
}): Metadata {
  const desc = metaDescription(description);
  // A page's openGraph replaces the root one whole, so the site image is
  // named here when the page has no art of its own.
  const images = image ? [{ url: image, alt: title }] : [defaultImage];
  return {
    title,
    description: desc,
    alternates: { canonical: path },
    openGraph: { type, siteName, locale: "id_ID", title, description: desc, url: path, images },
    twitter: { card: "summary_large_image", images: images.map((i) => i.url) },
  };
}
