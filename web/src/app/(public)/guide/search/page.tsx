import type { Metadata } from "next";
import { Suspense } from "react";

import { PageBreadcrumb } from "@/components/layout/page-breadcrumb";
import { EntryIndexSkeleton } from "@/features/gamedata/components";
import { GuideSearchForm, SearchResults } from "@/features/guide/components";
import { searchGuides } from "@/features/guide/data";

export const metadata: Metadata = { title: "Cari panduan", robots: { index: false } };

type Props = PageProps<"/guide/search">;

function queryFrom(value: string | string[] | undefined): string {
  return (Array.isArray(value) ? value[0] : value)?.trim().slice(0, 100) ?? "";
}

export default function GuideSearchPage({ searchParams }: Props) {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <PageBreadcrumb items={[{ label: "Panduan", href: "/guide" }, { label: "Cari" }]} />
      <h1 className="mt-6 text-4xl font-semibold tracking-tight">Cari panduan</h1>
      <Suspense fallback={<EntryIndexSkeleton />}>
        <Results searchParams={searchParams} />
      </Suspense>
    </div>
  );
}

async function Results({ searchParams }: Pick<Props, "searchParams">) {
  const q = queryFrom((await searchParams).q);
  const hits = q.length >= 2 ? await searchGuides(q) : [];

  return (
    <>
      <div className="mt-6">
        <GuideSearchForm defaultValue={q} />
      </div>
      <div className="mt-8">
        {q.length < 2 ? (
          <p className="text-muted-foreground">Ketik minimal dua huruf untuk mencari.</p>
        ) : hits.length === 0 ? (
          <p className="text-muted-foreground">
            Tidak ada artikel yang cocok dengan &ldquo;{q}&rdquo;. Coba kata lain, misalnya nama aturan atau kondisi.
          </p>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">{hits.length} artikel cocok</p>
            <SearchResults hits={hits} />
          </>
        )}
      </div>
    </>
  );
}
