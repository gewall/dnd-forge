import type { Metadata } from "next";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { getGalleryPage } from "@/features/gallery/data";
import { GalleryGrid } from "@/features/gallery/gallery-grid";

export const metadata: Metadata = {
  title: "Galeri",
  description: "Portrait karakter yang dibagikan para pemain, dibuat dengan AI.",
};

export default async function GalleryPage() {
  const page = await getGalleryPage();
  const items = page.items ?? [];

  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <h1 className="text-4xl font-semibold tracking-tight">Galeri</h1>
      <p className="mt-3 max-w-2xl text-lg text-muted-foreground">
        Karakter yang dibagikan pemiliknya, masing-masing dengan portrait terbarunya.
      </p>
      <div className="mt-10">
        {items.length === 0 ? (
          <div className="rounded-xl border border-dashed p-10 text-center">
            <p className="font-heading text-xl font-semibold">Galeri masih kosong</p>
            <p className="mt-2 text-muted-foreground">
              Buat karakter, beri portrait, lalu nyalakan &ldquo;Tampil di galeri publik&rdquo;.
            </p>
            <Button asChild className="mt-5">
              <Link href="/builder">Buat karakter</Link>
            </Button>
          </div>
        ) : (
          <GalleryGrid initial={items} nextBefore={page.nextBefore} />
        )}
      </div>
    </div>
  );
}
