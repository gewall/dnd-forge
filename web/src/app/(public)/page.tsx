import Link from "next/link";

import { Button } from "@/components/ui/button";
import { getGalleryPage } from "@/features/gallery/data";
import { GalleryCard } from "@/features/gallery/gallery-grid";

const entryPoints = [
  { href: "/guide/dasar-aturan", title: "Baru pertama main?", body: "Dadu, ability score, dan cara satu giliran berjalan." },
  { href: "/races", title: "Sembilan ras", body: "Dari Dwarf yang keras kepala sampai Tiefling berdarah iblis." },
  { href: "/classes", title: "Dua belas kelas", body: "Pejuang, penyihir, atau sesuatu di antaranya." },
];

export default async function HomePage() {
  const gallery = await getGalleryPage(undefined, 8);
  const portraits = gallery.items ?? [];

  return (
    <div className="mx-auto max-w-6xl px-4 sm:px-6">
      <section className="py-16 md:py-24">
        <div className="max-w-2xl">
          <h1 className="text-4xl leading-[1.1] font-semibold tracking-tight sm:text-5xl">
            Bangun karakter pertamamu, pahami aturannya sambil jalan.
          </h1>
          <p className="mt-5 max-w-xl text-lg leading-relaxed text-muted-foreground">
            Panduan bermain berbahasa Indonesia, ensiklopedia ras, kelas, dan latar belakang dari SRD 5.2, dan
            pembuat karakter yang menuntunmu langkah demi langkah.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Button asChild size="lg" className="h-10 px-4 text-base">
              <Link href="/builder">Buat karakter</Link>
            </Button>
            <Button asChild size="lg" variant="outline" className="h-10 px-4 text-base">
              <Link href="/guide">Baca panduan</Link>
            </Button>
          </div>
        </div>
      </section>

      <section aria-label="Mulai dari sini" className="grid gap-px overflow-hidden rounded-xl border bg-border md:grid-cols-3">
        {entryPoints.map((e) => (
          <Link key={e.href} href={e.href} className="group bg-background p-6 transition-colors hover:bg-card">
            <h2 className="text-xl font-semibold group-hover:text-primary">{e.title}</h2>
            <p className="mt-2 text-muted-foreground">{e.body}</p>
          </Link>
        ))}
      </section>

      {portraits.length > 0 && (
        <section aria-labelledby="home-gallery" className="py-16">
          <div className="flex items-baseline justify-between gap-4">
            <h2 id="home-gallery" className="text-2xl font-semibold">
              Dari galeri
            </h2>
            <Link href="/gallery" className="text-sm text-muted-foreground hover:text-foreground">
              Lihat semua
            </Link>
          </div>
          <ul className="mt-6 grid grid-cols-2 gap-x-4 gap-y-8 sm:grid-cols-4">
            {portraits.map((item) => (
              <li key={item.portraitId}>
                <GalleryCard item={item} />
              </li>
            ))}
          </ul>
        </section>
      )}
      <div className="pb-16" />
    </div>
  );
}
