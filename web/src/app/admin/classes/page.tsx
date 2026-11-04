import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { GameDataTable, NewButton } from "@/features/admin/content/lists";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Kelas" };

export default function Page() {
  return (
    <>
      <AdminHeader title="Kelas" description="Termasuk yang nonaktif." actions={<NewButton href="/admin/classes/new" label="Tambah kelas" />} />
      <Suspense fallback={<Skeleton className="h-80 w-full" />}>
        <List />
      </Suspense>
    </>
  );
}

async function List() {
  const items = adminData(await (await adminApi()).GET("/v1/admin/classes"), "Kelas")?.items ?? [];
  return (
    <GameDataTable
      base="/admin/classes"
      extraLabel="Hit die dan ability"
      rows={items.map((x) => ({
        id: x.id,
        name: x.name,
        slug: x.slug,
        isActive: x.isActive,
        sortOrder: x.sortOrder,
        extra: `d${x.hitDie}, ${x.primaryAbility}`,
      }))}
    />
  );
}
