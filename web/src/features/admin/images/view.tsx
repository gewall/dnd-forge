import { EyeOffIcon } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { SafeImage } from "@/features/ai/ui";
import type { components } from "@/lib/api/schema";

import { formatDate } from "../page-header";
import { HideToggle } from "./controls";

export type AdminImage = components["schemas"]["AdminImageDTO"];

const statuses: { [s: string]: { label: string; variant: "secondary" | "destructive" | "outline" } } = {
  SUCCEEDED: { label: "Selesai", variant: "secondary" },
  FAILED: { label: "Gagal", variant: "destructive" },
  PENDING: { label: "Diproses", variant: "outline" },
};

export function ImageBadges({ image }: { image: AdminImage }) {
  const s = statuses[image.status] ?? { label: image.status, variant: "outline" as const };
  return (
    <div className="flex flex-wrap gap-1.5">
      {image.status !== "SUCCEEDED" && <Badge variant={s.variant}>{s.label}</Badge>}
      {image.isHidden && (
        <Badge variant="outline" className="bg-background/90">
          <EyeOffIcon aria-hidden="true" />
          {image.hasFiles ? "Tersembunyi" : "File dihapus"}
        </Badge>
      )}
      {image.characterPublic && !image.isHidden && image.status === "SUCCEEDED" && (
        <Badge variant="outline" className="bg-background/90">
          Di galeri
        </Badge>
      )}
    </div>
  );
}

export function ImageGrid({ images }: { images: AdminImage[] }) {
  return (
    <ul className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5">
      {images.map((img) => (
        <li key={img.id} className="flex flex-col overflow-hidden rounded-xl border">
          <Link href={`/admin/images/${img.id}`} className="relative block focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
            <SafeImage
              src={img.thumbUrl || img.imageUrl}
              alt={`Portrait ${img.characterName || "karakter terhapus"}`}
              className={img.isHidden ? "aspect-square w-full opacity-60" : "aspect-square w-full"}
            />
            <div className="absolute top-2 left-2">
              <ImageBadges image={img} />
            </div>
          </Link>
          <div className="flex flex-1 flex-col gap-2 p-3 text-sm">
            <div className="min-w-0">
              <Link href={`/admin/images/${img.id}`} className="block truncate font-medium hover:text-primary hover:underline">
                {img.characterName || "Karakter terhapus"}
              </Link>
              <Link
                href={`/admin/images?userId=${img.userId}`}
                className="block truncate text-muted-foreground hover:text-foreground hover:underline"
                title="Semua portrait user ini"
              >
                {img.userEmail}
              </Link>
              <p className="text-muted-foreground">{formatDate(img.createdAt)}</p>
            </div>
            <div className="mt-auto">
              <HideToggle id={img.id} isHidden={img.isHidden} hasFiles={img.hasFiles} />
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
}
