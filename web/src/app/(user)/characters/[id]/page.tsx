import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PageBreadcrumb } from "@/components/layout/page-breadcrumb";
import { Button } from "@/components/ui/button";
import { DeleteCharacterButton, PublicToggle } from "@/features/characters/controls";
import { characterTitle, getCharacter } from "@/features/characters/data";
import { DetailSkeleton } from "@/features/gamedata/components";
import { requireUser } from "@/lib/auth/session";

export const metadata: Metadata = { title: "Detail karakter", robots: { index: false } };

type Props = PageProps<"/characters/[id]">;

export default function CharacterPage({ params }: Props) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <Suspense fallback={<DetailSkeleton />}>
        <CharacterDetail params={params} />
      </Suspense>
    </div>
  );
}

async function CharacterDetail({ params }: Pick<Props, "params">) {
  const { id } = await params;
  await requireUser(`/characters/${id}`);
  const c = await getCharacter(id);
  if (!c) notFound();

  return (
    <article>
      <PageBreadcrumb items={[{ label: "Karakter saya", href: "/characters" }, { label: c.name }]} />
      <header className="mt-6 flex flex-wrap items-start justify-between gap-6">
        <div>
          <h1 className="text-4xl font-semibold tracking-tight">{c.name}</h1>
          <p className="mt-2 text-lg text-muted-foreground">
            {characterTitle(c)}, latar{" "}
            <Link href={`/backgrounds/${c.background.slug}`} className="underline-offset-2 hover:underline">
              {c.background.name}
            </Link>
            {c.gender ? `, ${c.gender}` : ""}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link href={`/characters/${c.id}/edit`}>Ubah</Link>
          </Button>
          <DeleteCharacterButton id={c.id} name={c.name} />
        </div>
      </header>

      <div className="mt-6">
        <PublicToggle id={c.id} isPublic={c.isPublic} />
      </div>

      <div className="mt-10 grid gap-10 md:grid-cols-2">
        <section>
          <h2 className="text-xl font-semibold">Penampilan</h2>
          {c.appearance ? (
            <p className="mt-3 font-serif text-lg leading-relaxed whitespace-pre-line">{c.appearance}</p>
          ) : (
            <p className="mt-3 text-muted-foreground">
              Belum diisi. Tambahkan lewat{" "}
              <Link href={`/characters/${c.id}/edit`} className="underline underline-offset-2">
                Ubah
              </Link>{" "}
              supaya portrait AI lebih sesuai.
            </p>
          )}
        </section>
        <section>
          <h2 className="text-xl font-semibold">Kisah latar</h2>
          {c.backstory ? (
            <p className="mt-3 font-serif text-lg leading-relaxed whitespace-pre-line">{c.backstory}</p>
          ) : (
            <p className="mt-3 text-muted-foreground">Belum ada kisah latar untuk karakter ini.</p>
          )}
        </section>
      </div>

      <dl className="mt-10 grid gap-4 border-t pt-6 text-sm sm:grid-cols-3">
        <Ref label="Ras" name={c.subrace?.name ? `${c.race.name} (${c.subrace.name})` : c.race.name} href={`/races/${c.race.slug}`} />
        <Ref label="Kelas" name={c.class.name} href={`/classes/${c.class.slug}`} />
        <Ref label="Latar belakang" name={c.background.name} href={`/backgrounds/${c.background.slug}`} />
      </dl>
    </article>
  );
}

function Ref({ label, name, href }: { label: string; name: string; href: string }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="mt-0.5">
        <Link href={href} className="underline-offset-2 hover:underline">
          {name}
        </Link>
      </dd>
    </div>
  );
}
