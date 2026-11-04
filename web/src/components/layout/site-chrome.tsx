import { SiteFooter } from "./site-footer";
import { SiteHeader } from "./site-header";

/** Header, main and footer shared by every non-admin page. */
export function SiteChrome({ children }: { children: React.ReactNode }) {
  return (
    <>
      <SiteHeader />
      <main id="konten" className="flex-1">
        {children}
      </main>
      <SiteFooter />
    </>
  );
}
