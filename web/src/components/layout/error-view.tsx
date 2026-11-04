"use client";

import Link from "next/link";
import { useEffect } from "react";

import { Button } from "@/components/ui/button";

/** Body of every error boundary: what failed, and the two ways out. */
export function ErrorView({
  error,
  retry,
  title = "Halaman ini gagal dimuat",
  homeHref = "/",
  homeLabel = "Ke beranda",
}: {
  error: unknown;
  retry: () => void;
  title?: string;
  homeHref?: string;
  homeLabel?: string;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  const digest = typeof error === "object" && error && "digest" in error ? String((error as { digest?: unknown }).digest ?? "") : "";

  return (
    <div role="alert" className="mx-auto max-w-xl px-4 py-24 sm:px-6">
      <h1 className="text-3xl font-semibold">{title}</h1>
      <p className="mt-3 text-muted-foreground">
        Server data tidak menjawab dengan benar. Muat ulang bagian ini; kalau tetap gagal, coba lagi beberapa menit lagi.
      </p>
      <div className="mt-6 flex flex-wrap gap-2">
        <Button onClick={() => retry()}>Muat ulang</Button>
        <Button asChild variant="outline">
          <Link href={homeHref}>{homeLabel}</Link>
        </Button>
      </div>
      {digest && <p className="mt-6 font-mono text-xs text-muted-foreground">Kode kejadian: {digest}</p>}
    </div>
  );
}
