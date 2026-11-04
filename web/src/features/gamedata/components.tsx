import Link from "next/link";

import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

import type { Trait } from "./data";

const sections = [
  { href: "/races", label: "Ras" },
  { href: "/classes", label: "Kelas" },
  { href: "/backgrounds", label: "Latar belakang" },
] as const;

export type SectionHref = (typeof sections)[number]["href"];

/** Switches between the three encyclopedia sections. */
export function EncyclopediaNav({ current }: { current: SectionHref }) {
  return (
    <nav aria-label="Bagian ensiklopedia" className="flex gap-1 border-b">
      {sections.map((s) => (
        <Link
          key={s.href}
          href={s.href}
          aria-current={s.href === current ? "page" : undefined}
          className={cn(
            "-mb-px border-b-2 px-3 py-2 text-[0.95rem] transition-colors",
            s.href === current
              ? "border-primary text-foreground"
              : "border-transparent text-muted-foreground hover:text-foreground",
          )}
        >
          {s.label}
        </Link>
      ))}
    </nav>
  );
}

export interface IndexEntry {
  href: string;
  name: string;
  description: string;
  facts?: string[];
}

/** Book-index style list: name on the left, summary and facts on the right. */
export function EntryIndex({ entries, empty }: { entries: IndexEntry[]; empty: string }) {
  if (entries.length === 0) {
    return <p className="py-10 text-muted-foreground">{empty}</p>;
  }
  return (
    <ul className="divide-y">
      {entries.map((e) => (
        <li key={e.href}>
          <Link
            href={e.href}
            prefetch={true}
            className="group grid gap-x-8 gap-y-1 py-5 outline-none focus-visible:bg-muted/50 sm:grid-cols-[12rem_1fr]"
          >
            <span className="font-heading text-xl font-semibold group-hover:text-primary">{e.name}</span>
            <span>
              <span className="line-clamp-2 text-muted-foreground">{e.description}</span>
              {e.facts && e.facts.length > 0 && (
                <span className="mt-2 flex flex-wrap gap-1.5">
                  {e.facts.map((f) => (
                    <span key={f} className="rounded-md bg-secondary px-2 py-0.5 text-sm text-secondary-foreground">
                      {f}
                    </span>
                  ))}
                </span>
              )}
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

export function EntryIndexSkeleton() {
  return (
    <div className="divide-y">
      {Array.from({ length: 5 }, (_, i) => (
        <div key={i} className="grid gap-4 py-5 sm:grid-cols-[12rem_1fr]">
          <Skeleton className="h-6 w-28" />
          <Skeleton className="h-10 w-full" />
        </div>
      ))}
    </div>
  );
}

/** Racial traits or class features as a definition list. */
export function TraitList({ traits }: { traits: Trait[] }) {
  return (
    <dl className="space-y-5">
      {traits.map((t) => (
        <div key={t.name}>
          <dt className="font-heading text-lg font-semibold">{t.name}</dt>
          <dd className="mt-1 leading-relaxed text-muted-foreground">{t.description}</dd>
        </div>
      ))}
    </dl>
  );
}

export function DetailSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-10 w-56" />
      <Skeleton className="h-16 w-full max-w-2xl" />
      <Skeleton className="h-40 w-full max-w-2xl" />
    </div>
  );
}
