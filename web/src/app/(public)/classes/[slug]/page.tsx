import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { DetailSkeleton, TraitList } from "@/features/gamedata/components";
import { getClass } from "@/features/gamedata/data";
import { EntryLayout, EntrySection } from "@/features/gamedata/entry-layout";
import { pageMetadata } from "@/lib/seo";

export async function generateMetadata({ params }: PageProps<"/classes/[slug]">): Promise<Metadata> {
  const cls = await getClass((await params).slug);
  return cls ? pageMetadata({ title: cls.name, description: cls.description, path: `/classes/${cls.slug}`, image: cls.imageUrl || undefined }) : {};
}

export default function ClassPage({ params }: PageProps<"/classes/[slug]">) {
  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <Suspense fallback={<DetailSkeleton />}>
        <ClassDetail params={params} />
      </Suspense>
    </div>
  );
}

async function ClassDetail({ params }: Pick<PageProps<"/classes/[slug]">, "params">) {
  const cls = await getClass((await params).slug);
  if (!cls) notFound();

  const features = cls.features ?? [];

  return (
    <EntryLayout
      crumbs={[{ label: "Kelas", href: "/classes" }, { label: cls.name }]}
      name={cls.name}
      description={cls.description}
      imageUrl={cls.imageUrl}
      facts={[
        { label: "Hit die", value: `d${cls.hitDie}` },
        { label: "Ability utama", value: cls.primaryAbility },
      ]}
      builderHref={`/builder?class=${cls.slug}`}
      builderLabel={`Buat karakter ${cls.name}`}
    >
      {features.length > 0 && (
        <EntrySection title="Fitur kelas">
          <TraitList traits={features} />
        </EntrySection>
      )}
    </EntryLayout>
  );
}
