import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { CategoryManager } from "@/features/admin/content/guide-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Kategori panduan" };

export default function GuideCategoriesPage() {
  return (
    <>
      <AdminHeader title="Kategori panduan" description="Urutan dan judul bagian di halaman Panduan." />
      <Suspense fallback={<Skeleton className="h-80 w-full" />}>
        <Categories />
      </Suspense>
    </>
  );
}

async function Categories() {
  const api = await adminApi();
  const [cats, articles] = await Promise.all([api.GET("/v1/admin/guide-categories"), api.GET("/v1/admin/guides")]);
  const categories = adminData(cats, "kategori")?.items ?? [];
  const counts: Record<string, number> = {};
  for (const a of adminData(articles, "artikel")?.items ?? []) counts[a.categoryId] = (counts[a.categoryId] ?? 0) + 1;
  return <CategoryManager categories={categories} counts={counts} />;
}
