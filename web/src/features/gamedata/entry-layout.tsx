import Link from "next/link";

import { type Crumb, PageBreadcrumb } from "@/components/layout/page-breadcrumb";
import { Button } from "@/components/ui/button";

/** Shared frame of a race, class or background page. */
export function EntryLayout({
  crumbs,
  name,
  description,
  imageUrl,
  facts,
  builderHref,
  builderLabel,
  children,
}: {
  crumbs: Crumb[];
  name: string;
  description: string;
  imageUrl?: string;
  facts?: { label: string; value: string }[];
  builderHref: string;
  builderLabel: string;
  children: React.ReactNode;
}) {
  return (
    <article>
      <PageBreadcrumb items={crumbs} />
      <div className="mt-6 grid gap-10 lg:grid-cols-[1fr_18rem]">
        <div className="min-w-0">
          <h1 className="text-4xl font-semibold tracking-tight">{name}</h1>
          <p className="mt-4 max-w-2xl font-serif text-lg leading-relaxed">{description}</p>
          <div className="mt-10 max-w-2xl space-y-12">{children}</div>
        </div>
        <aside className="space-y-5 lg:sticky lg:top-20 lg:self-start">
          {imageUrl && (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={imageUrl} alt="" className="aspect-[4/5] w-full rounded-xl border object-cover" />
          )}
          {facts && facts.length > 0 && (
            <dl className="divide-y rounded-xl border bg-card">
              {facts.map((f) => (
                <div key={f.label} className="flex justify-between gap-4 px-4 py-3">
                  <dt className="text-muted-foreground">{f.label}</dt>
                  <dd className="text-right font-medium">{f.value}</dd>
                </div>
              ))}
            </dl>
          )}
          <Button asChild size="lg" className="h-10 w-full text-base">
            <Link href={builderHref}>{builderLabel}</Link>
          </Button>
        </aside>
      </div>
    </article>
  );
}

export function EntrySection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h2 className="mb-5 text-2xl font-semibold">{title}</h2>
      {children}
    </section>
  );
}
