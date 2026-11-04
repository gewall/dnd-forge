"use client";

import { ErrorView } from "@/components/layout/error-view";

export default function UserError(props: { error: unknown; retry: () => void }) {
  return <ErrorView {...props} title="Data akunmu gagal dimuat" />;
}
