import type { Metadata } from "next";
import { Suspense } from "react";

import { Wizard, WizardSkeleton } from "@/features/builder/wizard";
import { getBackgrounds, getClasses, getRaces } from "@/features/gamedata/data";

export const metadata: Metadata = {
  title: "Buat karakter",
  description: "Pilih ras, kelas, dan latar belakang, lalu beri nama dan penampilan karaktermu.",
};

export default async function BuilderPage() {
  const [races, classes, backgrounds] = await Promise.all([getRaces(), getClasses(), getBackgrounds()]);

  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <h1 className="text-4xl font-semibold tracking-tight">Buat karakter</h1>
      <p className="mt-2 text-muted-foreground">
        Lima langkah. Pilihanmu tersimpan di perangkat ini, jadi aman kalau halaman dimuat ulang.
      </p>
      <div className="mt-8">
        {/* The wizard reads ?race=&class=&background= from encyclopedia links. */}
        <Suspense fallback={<WizardSkeleton />}>
          <Wizard data={{ races, classes, backgrounds }} mode={{ kind: "create" }} />
        </Suspense>
      </div>
    </div>
  );
}
