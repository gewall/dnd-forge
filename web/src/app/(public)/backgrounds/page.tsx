import type { Metadata } from "next";

import { EncyclopediaNav, EntryIndex } from "@/features/gamedata/components";
import { getBackgrounds } from "@/features/gamedata/data";

export const metadata: Metadata = {
  title: "Latar belakang",
  description: "Latar belakang dari SRD 5.2 dan skill yang dikuasai karakter karenanya.",
};

export default async function BackgroundsPage() {
  const backgrounds = await getBackgrounds();

  return (
    <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
      <h1 className="text-4xl font-semibold tracking-tight">Latar belakang</h1>
      <p className="mt-3 max-w-2xl text-lg text-muted-foreground">
        Latar belakang adalah kehidupan karaktermu sebelum berpetualang. Dari sinilah datang skill pertama yang
        dikuasainya.
      </p>
      <div className="mt-8">
        <EncyclopediaNav current="/backgrounds" />
        <EntryIndex
          empty="Daftar latar belakang belum bisa dimuat."
          entries={backgrounds.map((b) => ({
            href: `/backgrounds/${b.slug}`,
            name: b.name,
            description: b.description,
            facts: b.skillProficiencies ?? undefined,
          }))}
        />
      </div>
    </div>
  );
}
