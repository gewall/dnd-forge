import type { Metadata } from "next";

import { SiteChrome } from "@/components/layout/site-chrome";

// Personal and sign-in pages stay out of search results.
export const metadata: Metadata = { robots: { index: false, follow: false } };

export default function UserLayout({ children }: LayoutProps<"/">) {
  return <SiteChrome>{children}</SiteChrome>;
}
