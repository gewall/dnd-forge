import type { Metadata } from "next";
import { Suspense } from "react";

import { FieldSeparator } from "@/components/ui/field";
import { Skeleton } from "@/components/ui/skeleton";
import { GoogleButton } from "@/features/auth/google-button";
import { RegisterForm } from "@/features/auth/register-form";

export const metadata: Metadata = { title: "Daftar" };

export default function RegisterPage() {
  const googleEnabled = Boolean(process.env.GOOGLE_CLIENT_ID);

  return (
    <>
      <h1 className="text-3xl font-semibold tracking-tight">Buat akun</h1>
      <p className="mt-2 text-muted-foreground">
        Gratis. Karakter yang kamu buat tersimpan dan bisa dibuka dari perangkat mana saja.
      </p>
      <div className="mt-8 space-y-6">
        <Suspense fallback={<Skeleton className="h-80 w-full" />}>
          {googleEnabled && (
            <>
              <GoogleButton />
              <FieldSeparator>atau pakai email</FieldSeparator>
            </>
          )}
          <RegisterForm />
        </Suspense>
      </div>
    </>
  );
}
