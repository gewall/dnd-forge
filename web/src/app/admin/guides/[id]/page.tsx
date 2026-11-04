import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { ArticleForm } from "@/features/admin/content/guide-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Ubah artikel" };

type Props = PageProps<"/admin/guides/[id]">;

export default function EditArticlePage({ params }: Props) {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <EditArticle params={params} />
    </Suspense>
  );
}

async function EditArticle({ params }: Pick<Props, "params">) {
  const { id } = await params;
  const api = await adminApi();
  const [art, cats] = await Promise.all([
    api.GET("/v1/admin/guides/{id}", { params: { path: { id } } }),
    api.GET("/v1/admin/guide-categories"),
  ]);
  const article = adminData(art, "artikel");
  if (!article) notFound();
  const categories = adminData(cats, "kategori")?.items ?? [];
  const category = categories.find((c) => c.id === article.categoryId);

  return (
    <>
      <AdminHeader
        title={article.title}
        crumbs={[{ label: "Artikel panduan", href: "/admin/guides" }, { label: article.title }]}
        actions={
          article.status === "PUBLISHED" && category ? (
            <Link href={`/guide/${category.slug}/${article.slug}`} className="text-sm text-muted-foreground underline-offset-2 hover:underline">
              Lihat di situs
            </Link>
          ) : undefined
        }
      />
      <ArticleForm key={article.updatedAt} article={article} categories={categories} />
    </>
  );
}
