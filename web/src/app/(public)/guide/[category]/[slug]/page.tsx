import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { PageBreadcrumb } from "@/components/layout/page-breadcrumb";
import { Markdown } from "@/components/markdown/markdown";
import { DetailSkeleton } from "@/features/gamedata/components";
import { ArticleList, TableOfContents } from "@/features/guide/components";
import { getArticle } from "@/features/guide/data";
import { extractToc } from "@/features/guide/text";

type Props = PageProps<"/guide/[category]/[slug]">;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { category, slug } = await params;
  const page = await getArticle(category, slug);
  if (!page) return {};
  const { article } = page;
  return {
    title: article.title,
    description: article.excerpt,
    openGraph: { type: "article", title: article.title, description: article.excerpt, images: article.coverUrl || undefined },
  };
}

export default function ArticlePage({ params }: Props) {
  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
      <Suspense fallback={<DetailSkeleton />}>
        <ArticleContent params={params} />
      </Suspense>
    </div>
  );
}

const dateFormat = new Intl.DateTimeFormat("id-ID", { dateStyle: "long", timeZone: "Asia/Jakarta" });

async function ArticleContent({ params }: Pick<Props, "params">) {
  const { category, slug } = await params;
  const page = await getArticle(category, slug);
  if (!page) notFound();

  const { article } = page;
  const related = page.related ?? [];
  const toc = extractToc(article.content);
  const categorySlug = article.categorySlug || category;

  return (
    <>
      <PageBreadcrumb
        items={[
          { label: "Panduan", href: "/guide" },
          { label: article.categoryTitle || "Kategori", href: `/guide/${categorySlug}` },
          { label: article.title },
        ]}
      />
      <div className="mt-6 grid gap-12 lg:grid-cols-[minmax(0,1fr)_14rem]">
        <article className="min-w-0 max-w-[68ch]">
          <h1 className="text-4xl leading-tight font-semibold tracking-tight">{article.title}</h1>
          {article.excerpt && <p className="mt-4 text-lg text-muted-foreground">{article.excerpt}</p>}
          {article.publishedAt && (
            <p className="mt-3 text-sm text-muted-foreground">
              Terbit {dateFormat.format(new Date(article.publishedAt))}
            </p>
          )}
          {article.coverUrl && (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={article.coverUrl} alt="" className="mt-8 aspect-[16/9] w-full rounded-xl border object-cover" />
          )}
          <Markdown source={article.content} className="mt-6" />

          {related.length > 0 && (
            <section className="mt-16 border-t pt-8" aria-labelledby="related">
              <h2 id="related" className="text-xl font-semibold">
                Baca juga
              </h2>
              <div className="mt-2">
                <ArticleList category={categorySlug} articles={related} />
              </div>
            </section>
          )}
        </article>
        <aside className="hidden lg:block">
          <div className="sticky top-20">
            <TableOfContents items={toc} />
          </div>
        </aside>
      </div>
    </>
  );
}
