import type { Metadata } from "next";

import { EncyclopediaNav, EntryIndex } from "@/features/gamedata/components";
import { getRaces } from "@/features/gamedata/data";

export const metadata: Metadata = {
  title: "Ras",
  description: "Sembilan ras dari SRD 5.2 beserta trait dan subrasnya.",
};

export default async function RacesPage() {
  const races = await getRaces();

  return (
    <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
      <h1 className="text-4xl font-semibold tracking-tight">Ras</h1>
      <p className="mt-3 max-w-2xl text-lg text-muted-foreground">
        Ras menentukan asal-usul karaktermu: bentuk tubuh, umur, dan kemampuan bawaan yang dibawa sejak lahir.
      </p>
      <div className="mt-8">
        <EncyclopediaNav current="/races" />
        <EntryIndex
          empty="Daftar ras belum bisa dimuat."
          entries={races.map((r) => ({
            href: `/races/${r.slug}`,
            name: r.name,
            description: r.description,
            facts: r.subraces?.length ? [`${r.subraces.length} pilihan subras`] : undefined,
          }))}
        />
      </div>
    </div>
  );
}
