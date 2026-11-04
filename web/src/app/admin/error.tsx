"use client";

import { ErrorView } from "@/components/layout/error-view";

export default function AdminError(props: { error: unknown; retry: () => void }) {
  return <ErrorView {...props} title="Data admin gagal dimuat" homeHref="/admin" homeLabel="Ke ringkasan admin" />;
}
