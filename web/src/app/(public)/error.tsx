"use client";

import { ErrorView } from "@/components/layout/error-view";

export default function PublicError(props: { error: unknown; retry: () => void }) {
  return <ErrorView {...props} />;
}
