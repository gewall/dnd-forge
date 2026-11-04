import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PageBreadcrumb } from "@/components/layout/page-breadcrumb";
import { draftFromCharacter } from "@/features/builder/draft";
import { Wizard, WizardSkeleton } from "@/features/builder/wizard";
import { getCharacter } from "@/features/characters/data";
import { getBackgrounds, getClasses, getRaces } from "@/features/gamedata/data";
import { requireUser } from "@/lib/auth/session";

export const metadata: Metadata = { title: "Ubah karakter", robots: { index: false } };

type Props = PageProps<"/characters/[id]/edit">;

export default function EditCharacterPage({ params }: Props) {
  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <Suspense fallback={<WizardSkeleton />}>
        <EditContent params={params} />
      </Suspense>
    </div>
  );
}

async function EditContent({ params }: Pick<Props, "params">) {
  const { id } = await params;
  await requireUser(`/characters/${id}/edit`);
  const [c, races, classes, backgrounds] = await Promise.all([getCharacter(id), getRaces(), getClasses(), getBackgrounds()]);
  if (!c) notFound();

  return (
    <>
      <PageBreadcrumb
        items={[
          { label: "Karakter saya", href: "/characters" },
          { label: c.name, href: `/characters/${c.id}` },
          { label: "Ubah" },
        ]}
      />
      <h1 className="mt-6 mb-8 text-4xl font-semibold tracking-tight">Ubah {c.name}</h1>
      <Wizard data={{ races, classes, backgrounds }} mode={{ kind: "edit", id: c.id, initial: draftFromCharacter(c) }} />
    </>
  );
}
