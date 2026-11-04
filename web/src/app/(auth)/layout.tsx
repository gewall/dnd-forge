import { SiteChrome } from "@/components/layout/site-chrome";

export default function AuthLayout({ children }: LayoutProps<"/">) {
  return (
    <SiteChrome>
      <div className="mx-auto w-full max-w-sm px-4 py-14 sm:py-20">{children}</div>
    </SiteChrome>
  );
}
