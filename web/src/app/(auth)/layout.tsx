import type { Metadata } from "next";

import { SiteChrome } from "@/components/layout/site-chrome";

// Personal and sign-in pages stay out of search results.
export const metadata: Metadata = { robots: { index: false, follow: false } };

export default function AuthLayout({ children }: LayoutProps<"/">) {
  return (
    <SiteChrome>
      <div className="mx-auto w-full max-w-sm px-4 py-14 sm:py-20">{children}</div>
    </SiteChrome>
  );
}
