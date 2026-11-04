import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { adminApi, adminData } from "@/features/admin/api";
import { AdminHeader } from "@/features/admin/page-header";
import { QuotaDefaultsForm } from "@/features/admin/users/account-form";

export const metadata = { title: "Kuota default" };

export default function SettingsPage() {
  return (
    <>
      <AdminHeader title="Kuota default" description="Batas generate harian untuk user tanpa kuota sendiri." />
      <Suspense fallback={<Skeleton className="h-40 w-full max-w-md" />}>
        <Defaults />
      </Suspense>
    </>
  );
}

async function Defaults() {
  const defaults = adminData(await (await adminApi()).GET("/v1/admin/settings/quota"), "kuota default");
  if (!defaults) return <p className="text-muted-foreground">Kuota default belum bisa dimuat.</p>;
  return <QuotaDefaultsForm initial={defaults} />;
}
