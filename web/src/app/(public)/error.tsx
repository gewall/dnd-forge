"use client";

import { useEffect } from "react";

import { Button } from "@/components/ui/button";

export default function PublicError({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="mx-auto max-w-xl px-4 py-24 sm:px-6">
      <h1 className="text-3xl font-semibold">Halaman ini gagal dimuat</h1>
      <p className="mt-3 text-muted-foreground">
        Server data tidak menjawab dengan benar. Muat ulang bagian ini; kalau tetap gagal, coba lagi beberapa menit
        lagi.
      </p>
      <Button className="mt-6" onClick={() => retry()}>
        Muat ulang
      </Button>
    </div>
  );
}
