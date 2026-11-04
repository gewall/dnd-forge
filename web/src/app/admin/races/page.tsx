import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { GameDataTable, NewButton } from "@/features/admin/content/lists";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Ras" };

export default function Page() {
  return (
    <>
      <AdminHeader title="Ras" description="Termasuk yang nonaktif. Ras nonaktif tidak tampil di situs dan builder." actions={<NewButton href="/admin/races/new" label="Tambah ras" />} />
      <Suspense fallback={<Skeleton className="h-80 w-full" />}>
        <List />
      </Suspense>
    </>
  );
}

async function List() {
  const items = adminData(await (await adminApi()).GET("/v1/admin/races"), "Ras")?.items ?? [];
  return (
    <GameDataTable
      base="/admin/races"
      extraLabel="Subras"
      rows={items.map((x) => ({
        id: x.id,
        name: x.name,
        slug: x.slug,
        isActive: x.isActive,
        sortOrder: x.sortOrder,
        extra: `${(x.subraces ?? []).length} subras`,
      }))}
    />
  );
}
