import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { RaceForm, SubraceManager } from "@/features/admin/content/game-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Ubah ras" };

type Props = PageProps<"/admin/races/[id]">;

export default function Page({ params }: Props) {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <Edit params={params} />
    </Suspense>
  );
}

async function Edit({ params }: Pick<Props, "params">) {
  const { id } = await params;
  const item = adminData(await (await adminApi()).GET("/v1/admin/races/{id}", { params: { path: { id } } }), "ras");
  if (!item) notFound();
  return (
    <>
      <AdminHeader title={item.name} crumbs={[{ label: "Ras", href: "/admin/races" }, { label: item.name }]} />
      <RaceForm key={item.id + item.name + item.slug} race={item} />
      <section className="mt-12">
        <h2 className="mb-4 text-xl font-semibold">Subras</h2>
        <SubraceManager raceId={item.id} subraces={item.subraces ?? []} />
      </section>
    </>
  );
}
