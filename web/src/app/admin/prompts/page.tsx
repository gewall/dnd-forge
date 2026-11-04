import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { TemplateManager } from "@/features/admin/ai/templates";
import { adminApi, adminData } from "@/features/admin/api";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Template prompt" };

export default function PromptsPage() {
  return (
    <>
      <AdminHeader
        title="Template prompt"
        description="Kerangka prompt yang diisi data karakter sebelum dikirim ke AI. Setiap simpanan menjadi versi baru; versi aktif dipakai untuk permintaan berikutnya."
      />
      <Suspense fallback={<Skeleton className="h-96 w-full" />}>
        <Templates />
      </Suspense>
    </>
  );
}

async function Templates() {
  const api = await adminApi();
  const [templates, styles, lore] = await Promise.all([
    api.GET("/v1/admin/prompt-templates"),
    api.GET("/v1/portrait-styles"),
    api.GET("/v1/lore-options"),
  ]);
  return (
    <TemplateManager
      templates={adminData(templates, "template")?.items ?? []}
      placeholders={{
        portrait: adminData(styles, "gaya portrait")?.placeholders ?? [],
        lore: adminData(lore, "opsi lore")?.placeholders ?? [],
      }}
    />
  );
}
