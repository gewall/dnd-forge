import Link from "next/link";
import { Suspense } from "react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { adminApi, adminData, param } from "@/features/admin/api";
import { NewButton } from "@/features/admin/content/lists";
import { AdminHeader, formatDate } from "@/features/admin/page-header";

export const metadata = { title: "Artikel panduan" };

type Props = PageProps<"/admin/guides">;

export default function GuidesAdminPage({ searchParams }: Props) {
  return (
    <>
      <AdminHeader
        title="Artikel panduan"
        description="Draf tidak tampil di situs sampai diterbitkan."
        actions={<NewButton href="/admin/guides/new" label="Tulis artikel" />}
      />
      <Suspense fallback={<Skeleton className="h-80 w-full" />}>
        <Articles searchParams={searchParams} />
      </Suspense>
    </>
  );
}

async function Articles({ searchParams }: Pick<Props, "searchParams">) {
  const sp = await searchParams;
  const categoryId = param(sp.categoryId);
  const status = param(sp.status);
  const query: { categoryId?: string; status?: "DRAFT" | "PUBLISHED" } = {};
  if (categoryId) query.categoryId = categoryId;
  if (status === "DRAFT" || status === "PUBLISHED") query.status = status;

  const api = await adminApi();
  const [list, cats] = await Promise.all([
    api.GET("/v1/admin/guides", { params: { query } }),
    api.GET("/v1/admin/guide-categories"),
  ]);
  const articles = adminData(list, "artikel")?.items ?? [];
  const categories = adminData(cats, "kategori")?.items ?? [];
  const catTitle = new Map(categories.map((c) => [c.id, c.title]));

  return (
    <>
      <form className="mb-6 flex flex-wrap items-end gap-3">
        <div>
          <label htmlFor="gc" className="mb-1 block text-sm font-medium">
            Kategori
          </label>
          <select id="gc" name="categoryId" defaultValue={categoryId ?? ""} className="h-8 rounded-lg border border-input bg-background px-2 text-sm">
            <option value="">Semua</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.title}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="gs" className="mb-1 block text-sm font-medium">
            Status
          </label>
          <select id="gs" name="status" defaultValue={status ?? ""} className="h-8 rounded-lg border border-input bg-background px-2 text-sm">
            <option value="">Semua</option>
            <option value="DRAFT">Draf</option>
            <option value="PUBLISHED">Terbit</option>
          </select>
        </div>
        <Button type="submit" variant="secondary">
          Terapkan
        </Button>
      </form>
      {articles.length === 0 ? (
        <p className="text-muted-foreground">Belum ada artikel. Mulai dari kategori Dasar Aturan.</p>
      ) : (
        <div className="rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Judul</TableHead>
                <TableHead>Kategori</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Terbit</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {articles.map((a) => (
                <TableRow key={a.id}>
                  <TableCell>
                    <Link href={`/admin/guides/${a.id}`} className="font-medium hover:text-primary hover:underline">
                      {a.title}
                    </Link>
                  </TableCell>
                  <TableCell>{catTitle.get(a.categoryId) ?? "-"}</TableCell>
                  <TableCell>{a.status === "PUBLISHED" ? "Terbit" : <span className="text-muted-foreground">Draf</span>}</TableCell>
                  <TableCell>{formatDate(a.publishedAt)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </>
  );
}
