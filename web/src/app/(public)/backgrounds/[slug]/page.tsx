import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { DetailSkeleton } from "@/features/gamedata/components";
import { getBackground } from "@/features/gamedata/data";
import { EntryLayout, EntrySection } from "@/features/gamedata/entry-layout";
import { pageMetadata } from "@/lib/seo";

export async function generateMetadata({ params }: PageProps<"/backgrounds/[slug]">): Promise<Metadata> {
  const bg = await getBackground((await params).slug);
  return bg ? pageMetadata({ title: bg.name, description: bg.description, path: `/backgrounds/${bg.slug}`, image: bg.imageUrl || undefined }) : {};
}

export default function BackgroundPage({ params }: PageProps<"/backgrounds/[slug]">) {
  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <Suspense fallback={<DetailSkeleton />}>
        <BackgroundDetail params={params} />
      </Suspense>
    </div>
  );
}

async function BackgroundDetail({ params }: Pick<PageProps<"/backgrounds/[slug]">, "params">) {
  const bg = await getBackground((await params).slug);
  if (!bg) notFound();

  const skills = bg.skillProficiencies ?? [];

  return (
    <EntryLayout
      crumbs={[{ label: "Latar belakang", href: "/backgrounds" }, { label: bg.name }]}
      name={bg.name}
      description={bg.description}
      imageUrl={bg.imageUrl}
      builderHref={`/builder?background=${bg.slug}`}
      builderLabel={`Pakai latar ${bg.name}`}
    >
      {skills.length > 0 && (
        <EntrySection title="Skill yang dikuasai">
          <ul className="flex flex-wrap gap-2">
            {skills.map((s) => (
              <li key={s} className="rounded-md bg-secondary px-3 py-1 text-secondary-foreground">
                {s}
              </li>
            ))}
          </ul>
        </EntrySection>
      )}
    </EntryLayout>
  );
}
