import { XIcon } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData, param } from "@/features/admin/api";
import { AuditList, entities } from "@/features/admin/audit/view";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Audit log" };

const PAGE = 50;
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
type Props = PageProps<"/admin/audit">;

export default function AuditPage({ searchParams }: Props) {
  return (
    <>
      <AdminHeader title="Audit log" description="Setiap perubahan yang dibuat admin, terbaru dulu. Buka satu baris untuk melihat rinciannya." />
      <Suspense fallback={<Skeleton className="h-96 w-full" />}>
        <Audit searchParams={searchParams} />
      </Suspense>
    </>
  );
}

async function Audit({ searchParams }: Pick<Props, "searchParams">) {
  const sp = await searchParams;
  const entity = param(sp.entity);
  const entityId = param(sp.entityId);
  const actorId = param(sp.actorId);
  const beforeId = Number(param(sp.beforeId));

  const query: { limit: number; entity?: string; entityId?: string; actorId?: string; beforeId?: number } = { limit: PAGE };
  if (entity) query.entity = entity.slice(0, 40);
  if (entityId) query.entityId = entityId.slice(0, 100);
  if (actorId && uuid.test(actorId)) query.actorId = actorId;
  if (Number.isSafeInteger(beforeId) && beforeId > 0) query.beforeId = beforeId;

  const page = adminData(await (await adminApi()).GET("/v1/admin/audit-logs", { params: { query } }), "audit log");
  const records = page?.items ?? [];
  const filters = { ...(query.entity && { entity: query.entity }), ...(query.entityId && { entityId: query.entityId }), ...(query.actorId && { actorId: query.actorId }) };
  const last = records.at(-1);
  const nextHref = records.length === PAGE && last ? `/admin/audit?${new URLSearchParams({ ...filters, beforeId: String(last.id) })}` : null;
  const { actorId: _actor, ...withoutActor } = filters;
  const actorEmail = query.actorId ? records.find((r) => r.actorId === query.actorId)?.actorEmail : undefined;

  return (
    <>
      <form className="mb-6 flex flex-wrap items-end gap-3" role="search">
        <div>
          <label htmlFor="ae" className="mb-1 block text-sm font-medium">
            Data
          </label>
          <select id="ae" name="entity" defaultValue={query.entity ?? ""} className="h-8 rounded-lg border border-input bg-background px-2 text-sm">
            <option value="">Semua</option>
            {Object.entries(entities).map(([key, e]) => (
              <option key={key} value={key}>
                {e.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="ai" className="mb-1 block text-sm font-medium">
            ID data
          </label>
          <Input id="ai" name="entityId" defaultValue={query.entityId} placeholder="UUID" className="w-72 font-mono text-sm" />
        </div>
        {query.actorId && <input type="hidden" name="actorId" value={query.actorId} />}
        <Button type="submit" variant="secondary">
          Terapkan
        </Button>
        {Object.keys(filters).length > 0 && (
          <Button asChild variant="ghost">
            <Link href="/admin/audit">Reset</Link>
          </Button>
        )}
      </form>

      {query.actorId && (
        <p className="mb-4 flex items-center gap-2 text-sm">
          Hanya perubahan oleh <span className="font-medium">{actorEmail ?? query.actorId}</span>
          <Button asChild size="icon-xs" variant="ghost" aria-label="Hapus filter admin">
            <Link href={`/admin/audit?${new URLSearchParams(withoutActor)}`}>
              <XIcon />
            </Link>
          </Button>
        </p>
      )}

      {records.length === 0 ? <p className="text-muted-foreground">Belum ada catatan yang cocok.</p> : <AuditList records={records} />}

      <div className="mt-4 flex gap-2">
        {query.beforeId && (
          <Button asChild variant="outline">
            <Link href={`/admin/audit?${new URLSearchParams(filters)}`}>Ke yang terbaru</Link>
          </Button>
        )}
        {nextHref && (
          <Button asChild variant="outline">
            <Link href={nextHref}>Lebih lama</Link>
          </Button>
        )}
      </div>
    </>
  );
}
