import type { Metadata } from "next";
import Link from "next/link";

import { ArticleList, GuideSearchForm } from "@/features/guide/components";
import { getGuideCategories } from "@/features/guide/data";

export const metadata: Metadata = {
  title: "Panduan bermain",
  description: "Aturan dasar, membuat karakter, combat, sihir, dan petualangan, dijelaskan dalam bahasa Indonesia.",
};

export default async function GuidePage() {
  const categories = await getGuideCategories();

  return (
    <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
      <h1 className="text-4xl font-semibold tracking-tight">Panduan bermain</h1>
      <p className="mt-3 max-w-2xl text-lg text-muted-foreground">
        Mulai dari dasar aturan kalau baru pertama main. Sisanya bisa dibaca saat kamu butuh di tengah permainan.
      </p>
      <div className="mt-6">
        <GuideSearchForm />
      </div>

      {categories.length === 0 ? (
        <p className="mt-12 text-muted-foreground">Panduan belum bisa dimuat.</p>
      ) : (
        <div className="mt-12 space-y-12">
          {categories.map((c) => {
            const articles = c.articles ?? [];
            return (
              <section key={c.id} aria-labelledby={`cat-${c.slug}`}>
                <h2 id={`cat-${c.slug}`} className="text-2xl font-semibold">
                  <Link href={`/guide/${c.slug}`} className="hover:text-primary">
                    {c.title}
                  </Link>
                </h2>
                {c.description && <p className="mt-1 text-muted-foreground">{c.description}</p>}
                {articles.length > 0 ? (
                  <div className="mt-3">
                    <ArticleList category={c.slug} articles={articles} />
                  </div>
                ) : (
                  <p className="mt-3 text-sm text-muted-foreground">Artikel untuk bagian ini sedang ditulis.</p>
                )}
              </section>
            );
          })}
        </div>
      )}
    </div>
  );
}
