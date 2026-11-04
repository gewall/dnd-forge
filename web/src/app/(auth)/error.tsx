"use client";

import { ErrorView } from "@/components/layout/error-view";

export default function AuthError(props: { error: unknown; retry: () => void }) {
  return <ErrorView {...props} title="Halaman masuk gagal dimuat" />;
}
