import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";

import { Button } from "@/components/ui/button";
import { EntryIndexSkeleton } from "@/features/gamedata/components";
import { characterTitle, listCharacters } from "@/features/characters/data";
import { requireUser } from "@/lib/auth/session";

export const metadata: Metadata = { title: "Karakter saya" };

export default function CharactersPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="text-4xl font-semibold tracking-tight">Karakter saya</h1>
        <Button asChild>
          <Link href="/builder">Buat karakter baru</Link>
        </Button>
      </div>
      <div className="mt-8">
        <Suspense fallback={<EntryIndexSkeleton />}>
          <CharacterList />
        </Suspense>
      </div>
    </div>
  );
}

const dateFormat = new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeZone: "Asia/Jakarta" });

async function CharacterList() {
  await requireUser("/characters");
  const characters = await listCharacters();

  if (characters === null) {
    return <p className="text-muted-foreground">Daftar karakter belum bisa dimuat. Muat ulang halaman ini.</p>;
  }
  if (characters.length === 0) {
    return (
      <div className="rounded-xl border border-dashed p-10 text-center">
        <p className="font-heading text-xl font-semibold">Belum ada karakter</p>
        <p className="mt-2 text-muted-foreground">Pilih ras, kelas, dan latar belakang. Lima langkah saja.</p>
        <Button asChild className="mt-5">
          <Link href="/builder">Buat karakter pertama</Link>
        </Button>
      </div>
    );
  }

  return (
    <ul className="divide-y border-y">
      {characters.map((c) => (
        <li key={c.id}>
          <Link
            href={`/characters/${c.id}`}
            className="group flex flex-wrap items-baseline justify-between gap-x-6 gap-y-1 py-5"
          >
            <span>
              <span className="font-heading text-xl font-semibold group-hover:text-primary">{c.name}</span>
              <span className="mt-0.5 block text-muted-foreground">
                {characterTitle(c)}, latar {c.background.name}
              </span>
            </span>
            <span className="flex items-center gap-3 text-sm text-muted-foreground">
              {c.isPublic && <span className="rounded-md bg-secondary px-2 py-0.5 text-secondary-foreground">Publik</span>}
              Diubah {dateFormat.format(new Date(c.updatedAt))}
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
