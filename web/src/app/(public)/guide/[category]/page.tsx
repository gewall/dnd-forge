import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PageBreadcrumb } from "@/components/layout/page-breadcrumb";
import { DetailSkeleton } from "@/features/gamedata/components";
import { ArticleList } from "@/features/guide/components";
import { getGuideCategory } from "@/features/guide/data";
import { pageMetadata } from "@/lib/seo";

export async function generateMetadata({ params }: PageProps<"/guide/[category]">): Promise<Metadata> {
  const category = await getGuideCategory((await params).category);
  return category ? pageMetadata({ title: category.title, description: category.description, path: `/guide/${category.slug}` }) : {};
}

export default function GuideCategoryPage({ params }: PageProps<"/guide/[category]">) {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <Suspense fallback={<DetailSkeleton />}>
        <CategoryContent params={params} />
      </Suspense>
    </div>
  );
}

async function CategoryContent({ params }: Pick<PageProps<"/guide/[category]">, "params">) {
  const category = await getGuideCategory((await params).category);
  if (!category) notFound();
  const articles = category.articles ?? [];

  return (
    <>
      <PageBreadcrumb items={[{ label: "Panduan", href: "/guide" }, { label: category.title }]} />
      <h1 className="mt-6 text-4xl font-semibold tracking-tight">{category.title}</h1>
      {category.description && <p className="mt-3 max-w-2xl text-lg text-muted-foreground">{category.description}</p>}
      <div className="mt-8">
        {articles.length > 0 ? (
          <ArticleList category={category.slug} articles={articles} />
        ) : (
          <p className="text-muted-foreground">Artikel untuk bagian ini sedang ditulis.</p>
        )}
      </div>
    </>
  );
}
