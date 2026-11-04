"use client";

import { ErrorView } from "@/components/layout/error-view";

import "./globals.css";

// Replaces the root layout when it fails, so it brings its own <html>.
export default function GlobalError(props: { error: unknown; retry: () => void }) {
  return (
    <html lang="id" className="dark">
      <body className="min-h-full bg-background font-sans text-foreground antialiased">
        <ErrorView {...props} title="DnD Forge sedang bermasalah" />
      </body>
    </html>
  );
}
