import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { DetailSkeleton, TraitList } from "@/features/gamedata/components";
import { getRace } from "@/features/gamedata/data";
import { EntryLayout, EntrySection } from "@/features/gamedata/entry-layout";
import { pageMetadata } from "@/lib/seo";

export async function generateMetadata({ params }: PageProps<"/races/[slug]">): Promise<Metadata> {
  const race = await getRace((await params).slug);
  return race ? pageMetadata({ title: race.name, description: race.description, path: `/races/${race.slug}`, image: race.imageUrl || undefined }) : {};
}

export default function RacePage({ params }: PageProps<"/races/[slug]">) {
  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <Suspense fallback={<DetailSkeleton />}>
        <RaceDetail params={params} />
      </Suspense>
    </div>
  );
}

async function RaceDetail({ params }: Pick<PageProps<"/races/[slug]">, "params">) {
  const race = await getRace((await params).slug);
  if (!race) notFound();

  const traits = race.traits ?? [];
  const subraces = (race.subraces ?? []).filter((s) => s.isActive);

  return (
    <EntryLayout
      crumbs={[{ label: "Ras", href: "/races" }, { label: race.name }]}
      name={race.name}
      description={race.description}
      imageUrl={race.imageUrl}
      facts={subraces.length ? [{ label: "Subras", value: `${subraces.length} pilihan` }] : undefined}
      builderHref={`/builder?race=${race.slug}`}
      builderLabel={`Buat karakter ${race.name}`}
    >
      {traits.length > 0 && (
        <EntrySection title="Trait">
          <TraitList traits={traits} />
        </EntrySection>
      )}
      {subraces.length > 0 && (
        <EntrySection title="Subras">
          <p className="-mt-2 mb-5 text-muted-foreground">Pilih salah satu saat membuat karakter {race.name}.</p>
          <ul className="space-y-6">
            {subraces.map((s) => (
              <li key={s.id} className="border-l-2 border-border pl-4">
                <h3 className="text-lg font-semibold">{s.name}</h3>
                <p className="mt-1 text-muted-foreground">{s.description}</p>
                {s.traits && s.traits.length > 0 && (
                  <div className="mt-4">
                    <TraitList traits={s.traits} />
                  </div>
                )}
              </li>
            ))}
          </ul>
        </EntrySection>
      )}
    </EntryLayout>
  );
}
