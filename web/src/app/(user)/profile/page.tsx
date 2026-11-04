import type { Metadata } from "next";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { ProfileDetails } from "@/features/auth/profile-details";

export const metadata: Metadata = { title: "Profil & kuota" };

export default function ProfilePage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <h1 className="text-3xl font-semibold tracking-tight">Profil & kuota</h1>
      <Suspense fallback={<ProfileSkeleton />}>
        <ProfileDetails />
      </Suspense>
    </div>
  );
}

function ProfileSkeleton() {
  return (
    <div className="mt-8 space-y-4">
      <Skeleton className="h-20 w-full" />
      <div className="grid gap-4 sm:grid-cols-2">
        <Skeleton className="h-32" />
        <Skeleton className="h-32" />
      </div>
    </div>
  );
}
