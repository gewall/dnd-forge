import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { GameDataTable, NewButton } from "@/features/admin/content/lists";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Latar belakang" };

export default function Page() {
  return (
    <>
      <AdminHeader title="Latar belakang" description="Termasuk yang nonaktif." actions={<NewButton href="/admin/backgrounds/new" label="Tambah latar belakang" />} />
      <Suspense fallback={<Skeleton className="h-80 w-full" />}>
        <List />
      </Suspense>
    </>
  );
}

async function List() {
  const items = adminData(await (await adminApi()).GET("/v1/admin/backgrounds"), "Latar belakang")?.items ?? [];
  return (
    <GameDataTable
      base="/admin/backgrounds"
      extraLabel="Skill"
      rows={items.map((x) => ({
        id: x.id,
        name: x.name,
        slug: x.slug,
        isActive: x.isActive,
        sortOrder: x.sortOrder,
        extra: (x.skillProficiencies ?? []).join(", "),
      }))}
    />
  );
}
