import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { ArticleForm } from "@/features/admin/content/guide-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Tulis artikel" };

export default function NewArticlePage() {
  return (
    <>
      <AdminHeader title="Tulis artikel" crumbs={[{ label: "Artikel panduan", href: "/admin/guides" }, { label: "Baru" }]} />
      <Suspense fallback={<Skeleton className="h-96 w-full" />}>
        <NewArticle />
      </Suspense>
    </>
  );
}

async function NewArticle() {
  const categories = adminData(await (await adminApi()).GET("/v1/admin/guide-categories"), "kategori")?.items ?? [];
  if (categories.length === 0) return <p className="text-muted-foreground">Buat kategori panduan dulu.</p>;
  return <ArticleForm categories={categories} />;
}
