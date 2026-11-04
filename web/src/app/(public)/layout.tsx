import { SiteChrome } from "@/components/layout/site-chrome";

export default function PublicLayout({ children }: LayoutProps<"/">) {
  return <SiteChrome>{children}</SiteChrome>;
}
