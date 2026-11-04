import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { AdminHeader, formatDate } from "@/features/admin/page-header";
import { AccountForm } from "@/features/admin/users/account-form";
import type { components } from "@/lib/api/schema";

export const metadata = { title: "Detail user" };

type Props = PageProps<"/admin/users/[id]">;

export default function UserPage({ params }: Props) {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <UserDetail params={params} />
    </Suspense>
  );
}

async function UserDetail({ params }: Pick<Props, "params">) {
  const { id } = await params;
  const client = await adminApi();
  const [userRes, defaultsRes] = await Promise.all([
    client.GET("/v1/admin/users/{id}", { params: { path: { id } } }),
    client.GET("/v1/admin/settings/quota"),
  ]);
  const u = adminData(userRes, "detail user");
  if (!u) notFound();
  const defaults = adminData(defaultsRes, "kuota default");

  return (
    <>
      <AdminHeader
        title={u.email}
        description={u.name || undefined}
        crumbs={[{ label: "User", href: "/admin/users" }, { label: u.email }]}
      />
      <div className="grid gap-10 xl:grid-cols-[1fr_24rem]">
        <section aria-labelledby="edit">
          <h2 id="edit" className="mb-4 text-xl font-semibold">
            Akses dan kuota
          </h2>
          <AccountForm
            id={u.id}
            initial={{ role: u.role, isBanned: u.isBanned, portraitQuota: u.portraitQuota, loreQuota: u.loreQuota }}
            defaults={defaults}
          />
        </section>
        <aside className="space-y-4">
          <dl className="divide-y rounded-xl border bg-card text-sm">
            <Row label="Karakter" value={String(u.characters)} />
            <Row label="Permintaan portrait" value={String(u.portraits)} />
            <Row label="Permintaan lore" value={String(u.lores)} />
            <Row label="Login" value={[u.hasPassword && "password", u.hasGoogle && "Google"].filter(Boolean).join(", ") || "-"} />
            <Row label="Bergabung" value={formatDate(u.createdAt)} />
            <Row label="Diubah" value={formatDate(u.updatedAt)} />
          </dl>
          <div className="grid grid-cols-2 gap-3">
            <Usage title="Portrait hari ini" usage={u.portraitUsage} />
            <Usage title="Lore hari ini" usage={u.loreUsage} />
          </div>
          <ul className="space-y-1 text-sm">
            <li>
              <Link href={`/admin/images?userId=${u.id}`} className="text-primary hover:underline">
                Lihat portrait user ini
              </Link>
            </li>
            <li>
              <Link href={`/admin/audit?${new URLSearchParams({ entity: "user", entityId: u.id })}`} className="text-primary hover:underline">
                Riwayat perubahan akun
              </Link>
            </li>
            {u.role === "ADMIN" && (
              <li>
                <Link href={`/admin/audit?actorId=${u.id}`} className="text-primary hover:underline">
                  Perubahan yang dibuat admin ini
                </Link>
              </li>
            )}
          </ul>
        </aside>
      </div>
    </>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4 px-4 py-2.5">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="text-right">{value}</dd>
    </div>
  );
}

function Usage({ title, usage }: { title: string; usage: components["schemas"]["UsageDTO"] }) {
  return (
    <div className="rounded-xl border bg-card p-4">
      <p className="text-sm text-muted-foreground">{title}</p>
      <p className="mt-1 text-2xl font-semibold tabular-nums">
        {usage.used}
        <span className="text-base font-normal text-muted-foreground"> / {usage.limit}</span>
      </p>
    </div>
  );
}
