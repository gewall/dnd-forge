import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { BannedWordsManager } from "@/features/admin/ai/banned-words";
import { adminApi, adminData } from "@/features/admin/api";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Kata terlarang" };

export default function BannedWordsPage() {
  return (
    <>
      <AdminHeader title="Kata terlarang" description="Prompt portrait dan lore yang memuat salah satu kata ini ditolak sebelum dikirim ke AI." />
      <Suspense fallback={<Skeleton className="h-64 w-full" />}>
        <Words />
      </Suspense>
    </>
  );
}

async function Words() {
  const words = adminData(await (await adminApi()).GET("/v1/admin/banned-words"), "kata terlarang");
  return <BannedWordsManager words={words?.items ?? []} />;
}
