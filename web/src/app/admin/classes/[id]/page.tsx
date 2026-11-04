import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { ClassForm } from "@/features/admin/content/game-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Ubah kelas" };

type Props = PageProps<"/admin/classes/[id]">;

export default function Page({ params }: Props) {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <Edit params={params} />
    </Suspense>
  );
}

async function Edit({ params }: Pick<Props, "params">) {
  const { id } = await params;
  const item = adminData(await (await adminApi()).GET("/v1/admin/classes/{id}", { params: { path: { id } } }), "kelas");
  if (!item) notFound();
  return (
    <>
      <AdminHeader title={item.name} crumbs={[{ label: "Kelas", href: "/admin/classes" }, { label: item.name }]} />
      <ClassForm key={item.id + item.name + item.slug} cls={item} />
    </>
  );
}
