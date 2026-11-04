import { ArrowRightIcon } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import type { components } from "@/lib/api/schema";

import { formatDate } from "../page-header";
import { describeDiff, formatValue } from "./diff";

type AuditRecord = components["schemas"]["RecordDTO"];

/** Entities the API audits, with where an admin edits each one. */
export const entities: { [entity: string]: { label: string; href?: (id: string) => string } } = {
  user: { label: "User", href: (id) => `/admin/users/${id}` },
  app_setting: { label: "Pengaturan", href: () => "/admin/settings" },
  guide_category: { label: "Kategori panduan", href: () => "/admin/guide-categories" },
  guide_article: { label: "Artikel panduan", href: (id) => `/admin/guides/${id}` },
  race: { label: "Ras", href: (id) => `/admin/races/${id}` },
  subrace: { label: "Subras" },
  class: { label: "Kelas", href: (id) => `/admin/classes/${id}` },
  background: { label: "Latar belakang", href: (id) => `/admin/backgrounds/${id}` },
  ai_provider: { label: "Provider AI", href: () => "/admin/ai" },
  prompt_template: { label: "Template prompt", href: () => "/admin/prompts" },
  generated_image: { label: "Portrait", href: (id) => `/admin/images/${id}` },
};

const actions: { [action: string]: { label: string; variant: "default" | "secondary" | "outline" | "destructive" } } = {
  create: { label: "Buat", variant: "secondary" },
  update: { label: "Ubah", variant: "outline" },
  activate: { label: "Aktifkan", variant: "default" },
  delete: { label: "Hapus", variant: "destructive" },
};

const fieldLabels: { [key: string]: string } = {
  role: "Peran",
  isBanned: "Diblokir",
  portraitQuota: "Kuota portrait",
  loreQuota: "Kuota lore",
  isHidden: "Disembunyikan",
};

const field = (key: string) => fieldLabels[key] ?? key;

function Diff({ diff }: { diff: unknown }) {
  const view = describeDiff(diff);
  if (view.kind === "none") return <p className="text-sm text-muted-foreground">Tidak ada rincian perubahan.</p>;
  if (view.kind === "changes") {
    return (
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
        {view.rows.map((r) => (
          <div key={r.key} className="contents">
            <dt className="text-muted-foreground">{field(r.key)}</dt>
            <dd className="flex flex-wrap items-center gap-2">
              <span className="text-muted-foreground line-through decoration-1">{formatValue(r.from)}</span>
              <ArrowRightIcon className="size-3.5 text-muted-foreground" aria-label="menjadi" />
              <span className="font-medium">{formatValue(r.to)}</span>
            </dd>
          </div>
        ))}
      </dl>
    );
  }
  return (
    <dl className="grid gap-2 text-sm sm:grid-cols-[10rem_minmax(0,1fr)]">
      {view.rows.map((r) => (
        <div key={r.key} className="contents">
          <dt className="font-mono text-xs leading-5 text-muted-foreground">{field(r.key)}</dt>
          <dd className="min-w-0">
            {r.long ? <pre className="max-h-64 overflow-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap">{r.value}</pre> : r.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

export function AuditList({ records }: { records: AuditRecord[] }) {
  return (
    <ol className="divide-y rounded-xl border">
      {records.map((r) => {
        const entity = entities[r.entity];
        const action = actions[r.action] ?? { label: r.action, variant: "outline" as const };
        const href = r.entityId && entity?.href?.(r.entityId);
        return (
          <li key={r.id}>
            <details className="group">
              <summary className="flex cursor-pointer list-none flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 hover:bg-muted/40 [&::-webkit-details-marker]:hidden">
                <Badge variant={action.variant}>{action.label}</Badge>
                <span className="font-medium">{entity?.label ?? r.entity}</span>
                {r.entityId && <span className="font-mono text-xs text-muted-foreground">{r.entityId.slice(0, 8)}</span>}
                <span className="text-sm text-muted-foreground">oleh {r.actorEmail || "sistem"}</span>
                <time dateTime={r.createdAt} className="ml-auto text-sm text-muted-foreground tabular-nums">
                  {formatDate(r.createdAt)}
                </time>
              </summary>
              <div className="space-y-4 border-t bg-muted/20 px-4 py-4">
                <Diff diff={r.diff} />
                <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm">
                  {href && r.action !== "delete" && (
                    <Link href={href} className="text-primary hover:underline">
                      Buka {entity.label.toLowerCase()}
                    </Link>
                  )}
                  {r.entityId && (
                    <Link href={`/admin/audit?${new URLSearchParams({ entity: r.entity, entityId: r.entityId })}`} className="text-primary hover:underline">
                      Riwayat data ini
                    </Link>
                  )}
                  {r.actorId && (
                    <Link href={`/admin/audit?${new URLSearchParams({ actorId: r.actorId })}`} className="text-primary hover:underline">
                      Semua perubahan oleh {r.actorEmail || "admin ini"}
                    </Link>
                  )}
                </div>
              </div>
            </details>
          </li>
        );
      })}
    </ol>
  );
}
