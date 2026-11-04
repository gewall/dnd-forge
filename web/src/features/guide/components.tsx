import { SearchIcon } from "lucide-react";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

import type { ArticleSummary, SearchHit } from "./data";
import { parseHeadline, type TocItem } from "./text";

/** Plain GET form: works without JavaScript and keeps results shareable by URL. */
export function GuideSearchForm({ defaultValue = "" }: { defaultValue?: string }) {
  return (
    <form action="/guide/search" role="search" className="flex max-w-lg gap-2">
      <label htmlFor="guide-q" className="sr-only">
        Cari di panduan
      </label>
      <Input
        id="guide-q"
        name="q"
        type="search"
        minLength={2}
        maxLength={100}
        required
        defaultValue={defaultValue}
        placeholder="Cari aturan, mis. advantage"
        className="h-10 text-base"
      />
      <Button type="submit" variant="secondary" className="h-10 px-4">
        <SearchIcon />
        Cari
      </Button>
    </form>
  );
}

export function ArticleList({ category, articles }: { category: string; articles: ArticleSummary[] }) {
  return (
    <ul className="divide-y">
      {articles.map((a) => (
        <li key={a.id}>
          <Link href={`/guide/${category}/${a.slug}`} prefetch={true} className="group block py-4">
            <span className="font-heading text-lg font-semibold group-hover:text-primary">{a.title}</span>
            {a.excerpt && <span className="mt-1 line-clamp-2 block text-muted-foreground">{a.excerpt}</span>}
          </Link>
        </li>
      ))}
    </ul>
  );
}

export function SearchResults({ hits }: { hits: SearchHit[] }) {
  return (
    <ol className="divide-y">
      {hits.map((h) => (
        <li key={h.id}>
          <Link href={`/guide/${h.categorySlug}/${h.slug}`} className="group block py-5">
            <span className="text-sm text-muted-foreground">{h.categoryTitle}</span>
            <span className="mt-0.5 block font-heading text-lg font-semibold group-hover:text-primary">{h.title}</span>
            <span className="mt-1 block text-muted-foreground">
              {parseHeadline(h.headline || h.excerpt).map((part, i) =>
                part.match ? (
                  <mark key={i} className="rounded-sm bg-primary/20 px-0.5 text-foreground">
                    {part.text}
                  </mark>
                ) : (
                  <span key={i}>{part.text}</span>
                ),
              )}
            </span>
          </Link>
        </li>
      ))}
    </ol>
  );
}

export function TableOfContents({ items }: { items: TocItem[] }) {
  if (items.length < 2) return null;
  return (
    <nav aria-label="Daftar isi" className="text-sm">
      <p className="mb-3 font-medium text-foreground">Di halaman ini</p>
      <ul className="space-y-2 border-l">
        {items.map((item) => (
          <li key={item.id}>
            <a
              href={`#${item.id}`}
              className={cn(
                "-ml-px block border-l border-transparent text-muted-foreground hover:border-primary hover:text-foreground",
                item.depth === 2 ? "pl-3" : "pl-6",
              )}
            >
              {item.text}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}
