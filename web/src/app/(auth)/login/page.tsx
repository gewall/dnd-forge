import type { Metadata } from "next";
import { Suspense } from "react";

import { FieldSeparator } from "@/components/ui/field";
import { Skeleton } from "@/components/ui/skeleton";
import { GoogleButton } from "@/features/auth/google-button";
import { LoginForm } from "@/features/auth/login-form";

export const metadata: Metadata = { title: "Masuk" };

export default function LoginPage() {
  const googleEnabled = Boolean(process.env.GOOGLE_CLIENT_ID);

  return (
    <>
      <h1 className="text-3xl font-semibold tracking-tight">Masuk</h1>
      <p className="mt-2 text-muted-foreground">Simpan karakter dan buat portrait-nya.</p>
      <div className="mt-8 space-y-6">
        <Suspense fallback={<FormSkeleton />}>
          {googleEnabled && (
            <>
              <GoogleButton />
              <FieldSeparator>atau pakai email</FieldSeparator>
            </>
          )}
          <LoginForm />
        </Suspense>
      </div>
    </>
  );
}

function FormSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-14 w-full" />
      <Skeleton className="h-14 w-full" />
      <Skeleton className="h-10 w-full" />
    </div>
  );
}
