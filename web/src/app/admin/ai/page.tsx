import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { ProviderManager } from "@/features/admin/ai/providers";
import { adminApi, adminData } from "@/features/admin/api";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Provider AI" };

export default function ProvidersPage() {
  return (
    <>
      <AdminHeader title="Provider AI" description="Layanan yang membuat portrait dan lore. Satu provider aktif per kemampuan; key tidak pernah ditampilkan utuh." />
      <Suspense fallback={<Skeleton className="h-96 w-full" />}>
        <Providers />
      </Suspense>
    </>
  );
}

async function Providers() {
  const api = await adminApi();
  const [providers, types] = await Promise.all([api.GET("/v1/admin/ai/providers"), api.GET("/v1/admin/ai/provider-types")]);
  return <ProviderManager providers={adminData(providers, "provider")?.items ?? []} types={adminData(types, "jenis provider")?.items ?? []} />;
}
