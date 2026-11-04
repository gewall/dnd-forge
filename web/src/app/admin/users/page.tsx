import Link from "next/link";
import { Suspense } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { adminApi, adminData, param } from "@/features/admin/api";
import { AdminHeader, formatDate } from "@/features/admin/page-header";

export const metadata = { title: "User" };

const PAGE = 50;
type Props = PageProps<"/admin/users">;

export default function UsersPage({ searchParams }: Props) {
  return (
    <>
      <AdminHeader title="User" description="Cari berdasarkan email atau nama, lalu buka untuk mengubah peran, blokir, atau kuota." />
      <Suspense fallback={<Skeleton className="h-96 w-full" />}>
        <Users searchParams={searchParams} />
      </Suspense>
    </>
  );
}

async function Users({ searchParams }: Pick<Props, "searchParams">) {
  const sp = await searchParams;
  const q = param(sp.q);
  const role = param(sp.role);
  const banned = param(sp.banned);
  const before = param(sp.before);

  const query: Record<string, string | number> = { limit: PAGE };
  if (q) query.q = q;
  if (role === "USER" || role === "ADMIN") query.role = role;
  if (banned === "true" || banned === "false") query.banned = banned;
  if (before) query.before = before;

  const page = adminData(await (await adminApi()).GET("/v1/admin/users", { params: { query } }), "daftar user");
  const users = page?.items ?? [];
  const last = users.at(-1);
  const nextHref =
    users.length === PAGE && last
      ? `/admin/users?${new URLSearchParams({ ...(q && { q }), ...(role && { role }), ...(banned && { banned }), before: last.createdAt })}`
      : null;

  return (
    <>
      <form className="mb-6 flex flex-wrap items-end gap-3" role="search">
        <div>
          <label htmlFor="uq" className="mb-1 block text-sm font-medium">
            Cari
          </label>
          <Input id="uq" name="q" defaultValue={q} placeholder="email atau nama" className="w-64" />
        </div>
        <div>
          <label htmlFor="ur" className="mb-1 block text-sm font-medium">
            Peran
          </label>
          <select id="ur" name="role" defaultValue={role ?? ""} className="h-8 rounded-lg border bg-background px-2 text-sm">
            <option value="">Semua</option>
            <option value="USER">Pemain</option>
            <option value="ADMIN">Admin</option>
          </select>
        </div>
        <div>
          <label htmlFor="ub" className="mb-1 block text-sm font-medium">
            Status
          </label>
          <select id="ub" name="banned" defaultValue={banned ?? ""} className="h-8 rounded-lg border bg-background px-2 text-sm">
            <option value="">Semua</option>
            <option value="false">Aktif</option>
            <option value="true">Diblokir</option>
          </select>
        </div>
        <Button type="submit" variant="secondary">
          Terapkan
        </Button>
        {(q || role || banned) && (
          <Button asChild variant="ghost">
            <Link href="/admin/users">Reset</Link>
          </Button>
        )}
      </form>

      {users.length === 0 ? (
        <p className="text-muted-foreground">Tidak ada user yang cocok.</p>
      ) : (
        <div className="rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead>
                <TableHead>Nama</TableHead>
                <TableHead>Peran</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Karakter</TableHead>
                <TableHead>Login</TableHead>
                <TableHead>Bergabung</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell>
                    <Link href={`/admin/users/${u.id}`} className="font-medium hover:text-primary hover:underline">
                      {u.email}
                    </Link>
                  </TableCell>
                  <TableCell>{u.name || "-"}</TableCell>
                  <TableCell>{u.role === "ADMIN" ? "Admin" : "Pemain"}</TableCell>
                  <TableCell>{u.isBanned ? <span className="text-destructive">Diblokir</span> : "Aktif"}</TableCell>
                  <TableCell className="text-right tabular-nums">{u.characters}</TableCell>
                  <TableCell>{[u.hasPassword && "password", u.hasGoogle && "Google"].filter(Boolean).join(", ") || "-"}</TableCell>
                  <TableCell>{formatDate(u.createdAt)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
      <div className="mt-4 flex gap-2">
        {before && (
          <Button asChild variant="outline">
            <Link href="/admin/users">Ke halaman pertama</Link>
          </Button>
        )}
        {nextHref && (
          <Button asChild variant="outline">
            <Link href={nextHref}>Halaman berikutnya</Link>
          </Button>
        )}
      </div>
    </>
  );
}
