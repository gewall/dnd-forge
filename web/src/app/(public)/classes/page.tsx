import type { Metadata } from "next";

import { EncyclopediaNav, EntryIndex } from "@/features/gamedata/components";
import { getClasses } from "@/features/gamedata/data";

export const metadata: Metadata = {
  title: "Kelas",
  description: "Dua belas kelas dari SRD 5.2: hit die, ability utama, dan fitur awalnya.",
};

export default async function ClassesPage() {
  const classes = await getClasses();

  return (
    <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
      <h1 className="text-4xl font-semibold tracking-tight">Kelas</h1>
      <p className="mt-3 max-w-2xl text-lg text-muted-foreground">
        Kelas adalah cara karaktermu bertarung dan bertahan hidup: senjata, sihir, atau keahlian yang terus berkembang
        tiap naik level.
      </p>
      <div className="mt-8">
        <EncyclopediaNav current="/classes" />
        <EntryIndex
          empty="Daftar kelas belum bisa dimuat."
          entries={classes.map((c) => ({
            href: `/classes/${c.slug}`,
            name: c.name,
            description: c.description,
            facts: [`Hit die d${c.hitDie}`, c.primaryAbility],
          }))}
        />
      </div>
    </div>
  );
}
