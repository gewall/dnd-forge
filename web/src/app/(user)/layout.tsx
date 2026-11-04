import { SiteChrome } from "@/components/layout/site-chrome";

export default function UserLayout({ children }: LayoutProps<"/">) {
  return <SiteChrome>{children}</SiteChrome>;
}
