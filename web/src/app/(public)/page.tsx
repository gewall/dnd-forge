import Link from "next/link";
import { Suspense } from "react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ServiceStatus } from "@/features/status/service-status";

export default function HomePage() {
  return (
    <div className="mx-auto max-w-6xl px-4 sm:px-6">
      <section className="grid gap-10 py-16 md:grid-cols-[3fr_2fr] md:py-24">
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

        <aside className="self-end rounded-xl border bg-card p-5">
          <h2 className="text-lg font-semibold">Status layanan</h2>
          <p className="mt-1 text-sm text-muted-foreground">Sambungan situs ini ke server data.</p>
          <Suspense fallback={<StatusSkeleton />}>
            <ServiceStatus />
          </Suspense>
        </aside>
      </section>
    </div>
  );
}

function StatusSkeleton() {
  return (
    <div className="mt-4 space-y-2">
      <Skeleton className="h-5 w-40" />
      <Skeleton className="h-5 w-52" />
    </div>
  );
}
