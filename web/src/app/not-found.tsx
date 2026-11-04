import Link from "next/link";

import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <>
      <SiteHeader />
      <main className="mx-auto w-full max-w-xl flex-1 px-4 py-24 sm:px-6">
        <h1 className="text-3xl font-semibold">Halaman tidak ditemukan</h1>
        <p className="mt-3 text-muted-foreground">
          Alamatnya mungkin salah ketik, atau halamannya belum dibuat. Mulai lagi dari beranda atau langsung buat
          karakter.
        </p>
        <div className="mt-6 flex flex-wrap gap-3">
          <Button asChild>
            <Link href="/">Ke beranda</Link>
          </Button>
          <Button asChild variant="outline">
            <Link href="/builder">Buat karakter</Link>
          </Button>
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
