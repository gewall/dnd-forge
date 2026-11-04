import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { BackgroundForm } from "@/features/admin/content/game-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Ubah latar belakang" };

type Props = PageProps<"/admin/backgrounds/[id]">;

export default function Page({ params }: Props) {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <Edit params={params} />
    </Suspense>
  );
}

async function Edit({ params }: Pick<Props, "params">) {
  const { id } = await params;
  const item = adminData(await (await adminApi()).GET("/v1/admin/backgrounds/{id}", { params: { path: { id } } }), "latar belakang");
  if (!item) notFound();
  return (
    <>
      <AdminHeader title={item.name} crumbs={[{ label: "Latar belakang", href: "/admin/backgrounds" }, { label: item.name }]} />
      <BackgroundForm key={item.id + item.name + item.slug} bg={item} />
    </>
  );
}
