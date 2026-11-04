import Link from "next/link";

import { SiteChrome } from "@/components/layout/site-chrome";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <SiteChrome>
      <div className="mx-auto w-full max-w-xl px-4 py-24 sm:px-6">
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
      </div>
    </SiteChrome>
  );
}
