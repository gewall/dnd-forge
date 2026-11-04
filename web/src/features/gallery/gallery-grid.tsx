"use client";

import { useState, useTransition } from "react";

import { Button } from "@/components/ui/button";
import { optionLabel } from "@/features/ai/labels";
import { SafeImage } from "@/features/ai/ui";
import type { components } from "@/lib/api/schema";

type GalleryItem = components["schemas"]["GalleryItemDTO"];

export function GalleryCard({ item }: { item: GalleryItem }) {
  return (
    <figure className="group">
      <a href={item.imageUrl} target="_blank" rel="noopener noreferrer" className="block overflow-hidden rounded-lg border">
        <SafeImage
          src={item.thumbUrl}
          alt={`Portrait ${item.characterName}, ${item.race} ${item.class}`}
          className="aspect-[3/4] w-full transition-transform duration-300 group-hover:scale-[1.02] motion-reduce:transition-none"
        />
      </a>
      <figcaption className="mt-2">
        <span className="block truncate font-heading text-lg font-semibold">{item.characterName}</span>
        <span className="block truncate text-sm text-muted-foreground">
          {item.race} {item.class}, gaya {optionLabel("style", item.style).toLowerCase()}
        </span>
      </figcaption>
    </figure>
  );
}

/** First page comes from the server; later pages load from /api/gallery with the cursor. */
export function GalleryGrid({ initial, nextBefore }: { initial: GalleryItem[]; nextBefore: string | null }) {
  const [items, setItems] = useState(initial);
  const [cursor, setCursor] = useState(nextBefore);
  const [error, setError] = useState<string | null>(null);
  const [loading, startTransition] = useTransition();

  const loadMore = () => {
    if (!cursor) return;
    setError(null);
    startTransition(async () => {
      try {
        const res = await fetch(`/api/gallery?before=${encodeURIComponent(cursor)}`);
        if (!res.ok) throw new Error();
        const page = (await res.json()) as { items: GalleryItem[] | null; nextBefore: string | null };
        const seen = new Set(items.map((i) => i.portraitId));
        setItems((list) => [...list, ...(page.items ?? []).filter((i) => !seen.has(i.portraitId))]);
        setCursor(page.nextBefore);
      } catch {
        setError("Halaman berikutnya belum bisa dimuat. Coba lagi.");
      }
    });
  };

  return (
    <>
      <ul className="grid grid-cols-2 gap-x-4 gap-y-8 sm:grid-cols-3 lg:grid-cols-4">
        {items.map((item) => (
          <li key={item.portraitId}>
            <GalleryCard item={item} />
          </li>
        ))}
      </ul>
      {(cursor || error) && (
        <div className="mt-10 flex flex-col items-center gap-2">
          {error && <p className="text-sm text-destructive">{error}</p>}
          {cursor && (
            <Button variant="outline" onClick={loadMore} disabled={loading}>
              {loading ? "Memuat…" : "Muat lagi"}
            </Button>
          )}
        </div>
      )}
    </>
  );
}
