import { XIcon } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData, param } from "@/features/admin/api";
import { ImageGrid } from "@/features/admin/images/view";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Moderasi gambar" };

const PAGE = 30;
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
type Props = PageProps<"/admin/images">;

export default function ImagesPage({ searchParams }: Props) {
  return (
    <>
      <AdminHeader
        title="Moderasi gambar"
        description="Semua portrait, terbaru dulu, termasuk yang disembunyikan. Sembunyikan portrait yang tidak pantas agar hilang dari galeri dan link media."
      />
      <Suspense fallback={<Skeleton className="h-96 w-full" />}>
        <Images searchParams={searchParams} />
      </Suspense>
    </>
  );
}

async function Images({ searchParams }: Pick<Props, "searchParams">) {
  const sp = await searchParams;
  const status = param(sp.status);
  const hidden = param(sp.hidden);
  const userId = param(sp.userId);
  const before = param(sp.before);

  const query: { limit: number; status?: "PENDING" | "SUCCEEDED" | "FAILED"; hidden?: "true" | "false"; userId?: string; before?: string } = { limit: PAGE };
  if (status === "PENDING" || status === "SUCCEEDED" || status === "FAILED") query.status = status;
  if (hidden === "true" || hidden === "false") query.hidden = hidden;
  if (userId && uuid.test(userId)) query.userId = userId;
  if (before && !Number.isNaN(Date.parse(before))) query.before = before;

  const page = adminData(await (await adminApi()).GET("/v1/admin/images", { params: { query } }), "daftar portrait");
  const images = page?.items ?? [];
  const filters = {
    ...(query.status && { status: query.status }),
    ...(query.hidden && { hidden: query.hidden }),
    ...(query.userId && { userId: query.userId }),
  };
  const { userId: _user, ...withoutUser } = filters;
  const last = images.at(-1);
  const nextHref = images.length === PAGE && last ? `/admin/images?${new URLSearchParams({ ...filters, before: last.createdAt })}` : null;
  const userEmail = query.userId ? images[0]?.userEmail : undefined;

  return (
    <>
      <form className="mb-6 flex flex-wrap items-end gap-3" role="search">
        <div>
          <label htmlFor="is" className="mb-1 block text-sm font-medium">
            Status
          </label>
          <select id="is" name="status" defaultValue={query.status ?? ""} className="h-8 rounded-lg border bg-background px-2 text-sm">
            <option value="">Semua</option>
            <option value="SUCCEEDED">Selesai</option>
            <option value="FAILED">Gagal</option>
            <option value="PENDING">Diproses</option>
          </select>
        </div>
        <div>
          <label htmlFor="ih" className="mb-1 block text-sm font-medium">
            Visibilitas
          </label>
          <select id="ih" name="hidden" defaultValue={query.hidden ?? ""} className="h-8 rounded-lg border bg-background px-2 text-sm">
            <option value="">Semua</option>
            <option value="false">Tampil</option>
            <option value="true">Tersembunyi</option>
          </select>
        </div>
        {query.userId && <input type="hidden" name="userId" value={query.userId} />}
        <Button type="submit" variant="secondary">
          Terapkan
        </Button>
        {Object.keys(filters).length > 0 && (
          <Button asChild variant="ghost">
            <Link href="/admin/images">Reset</Link>
          </Button>
        )}
      </form>

      {query.userId && (
        <p className="mb-4 flex flex-wrap items-center gap-2 text-sm">
          Hanya portrait milik{" "}
          <Link href={`/admin/users/${query.userId}`} className="font-medium hover:text-primary hover:underline">
            {userEmail ?? "user ini"}
          </Link>
          <Button asChild size="icon-xs" variant="ghost" aria-label="Hapus filter user">
            <Link href={`/admin/images?${new URLSearchParams(withoutUser)}`}>
              <XIcon />
            </Link>
          </Button>
        </p>
      )}

      {images.length === 0 ? <p className="text-muted-foreground">Belum ada portrait yang cocok.</p> : <ImageGrid images={images} />}

      <div className="mt-6 flex gap-2">
        {query.before && (
          <Button asChild variant="outline">
            <Link href={`/admin/images?${new URLSearchParams(filters)}`}>Ke yang terbaru</Link>
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
