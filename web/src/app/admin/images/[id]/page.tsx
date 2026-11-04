import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { optionLabel } from "@/features/ai/labels";
import { SafeImage } from "@/features/ai/ui";
import { adminApi, adminData } from "@/features/admin/api";
import { DeleteFiles, HideToggle } from "@/features/admin/images/controls";
import { ImageBadges } from "@/features/admin/images/view";
import { AdminHeader, formatDate } from "@/features/admin/page-header";

export const metadata = { title: "Detail portrait" };

type Props = PageProps<"/admin/images/[id]">;

const aspectClass: { [ratio: string]: string } = { "1:1": "aspect-square", "3:4": "aspect-[3/4]", "4:3": "aspect-[4/3]" };
const statusLabel: { [s: string]: string } = { SUCCEEDED: "Selesai", FAILED: "Gagal", PENDING: "Diproses" };

export default function ImagePage({ params }: Props) {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <ImageDetail params={params} />
    </Suspense>
  );
}

async function ImageDetail({ params }: Pick<Props, "params">) {
  const { id } = await params;
  const img = adminData(await (await adminApi()).GET("/v1/admin/images/{id}", { params: { path: { id } } }), "detail portrait");
  if (!img) notFound();
  const name = img.characterName || "Karakter terhapus";

  return (
    <>
      <AdminHeader
        title={name}
        description={`Portrait oleh ${img.userEmail}`}
        crumbs={[{ label: "Moderasi gambar", href: "/admin/images" }, { label: name }]}
        actions={
          <>
            <HideToggle id={img.id} isHidden={img.isHidden} hasFiles={img.hasFiles} size="default" />
            {img.hasFiles && <DeleteFiles id={img.id} disabled={img.status === "PENDING"} />}
          </>
        }
      />
      <div className="grid gap-10 lg:grid-cols-[minmax(0,28rem)_minmax(0,1fr)]">
        <div className="space-y-3">
          <SafeImage
            src={img.imageUrl || img.thumbUrl}
            alt={`Portrait ${name}`}
            className={`w-full rounded-xl border ${aspectClass[img.aspectRatio] ?? "aspect-square"} ${img.isHidden ? "opacity-70" : ""}`}
          />
          <ImageBadges image={img} />
          {!img.hasFiles && <p className="text-sm text-muted-foreground">File gambar sudah dihapus admin; portrait ini tersembunyi permanen.</p>}
        </div>

        <div className="space-y-8">
          <dl className="grid grid-cols-[8rem_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
            <dt className="text-muted-foreground">Status</dt>
            <dd>{statusLabel[img.status] ?? img.status}</dd>
            <dt className="text-muted-foreground">Pemilik</dt>
            <dd className="flex flex-wrap gap-x-3">
              <Link href={`/admin/users/${img.userId}`} className="hover:text-primary hover:underline">
                {img.userEmail}
              </Link>
              <Link href={`/admin/images?userId=${img.userId}`} className="text-primary hover:underline">
                Portrait lainnya
              </Link>
            </dd>
            <dt className="text-muted-foreground">Karakter</dt>
            <dd>
              {name}
              {img.characterId && <span className="text-muted-foreground">{img.characterPublic ? ", publik" : ", privat"}</span>}
            </dd>
            <dt className="text-muted-foreground">Gaya</dt>
            <dd>{optionLabel("style", img.style)}</dd>
            <dt className="text-muted-foreground">Rasio</dt>
            <dd className="tabular-nums">{img.aspectRatio}</dd>
            <dt className="text-muted-foreground">Model</dt>
            <dd className="font-mono text-xs leading-5 break-all">{img.model || "-"}</dd>
            <dt className="text-muted-foreground">Dibuat</dt>
            <dd>{formatDate(img.createdAt)}</dd>
            <dt className="text-muted-foreground">Selesai</dt>
            <dd>{formatDate(img.completedAt)}</dd>
          </dl>

          {img.errorMessage && (
            <section aria-labelledby="img-error">
              <h2 id="img-error" className="mb-2 font-semibold">
                Pesan gagal
              </h2>
              <p className="rounded-lg border border-destructive/40 bg-destructive/10 p-3 text-sm">{img.errorMessage}</p>
            </section>
          )}

          <section aria-labelledby="img-prompt">
            <h2 id="img-prompt" className="mb-2 font-semibold">
              Permintaan user
            </h2>
            <p className="mb-2 text-sm text-muted-foreground">Detail tambahan yang ditulis user (sudah lolos moderasi kata terlarang).</p>
            {img.prompt ? (
              <pre className="max-h-96 overflow-auto rounded-lg bg-muted p-3 font-sans text-sm whitespace-pre-wrap">{img.prompt}</pre>
            ) : (
              <p className="text-sm text-muted-foreground">User tidak menulis detail tambahan.</p>
            )}
          </section>

          <Link href={`/admin/audit?${new URLSearchParams({ entity: "generated_image", entityId: img.id })}`} className="inline-block text-sm text-primary hover:underline">
            Riwayat moderasi portrait ini
          </Link>
        </div>
      </div>
    </>
  );
}
