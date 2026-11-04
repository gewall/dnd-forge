import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Suspense } from "react";

import { Logo } from "@/components/layout/logo";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { Skeleton } from "@/components/ui/skeleton";
import { AdminMobileNav, AdminNav } from "@/features/admin/nav";
import { UserNav } from "@/features/auth/user-nav";
import { getCurrentUser } from "@/lib/auth/session";

export const metadata: Metadata = { title: { default: "Admin", template: "%s · Admin DnD Forge" }, robots: { index: false } };

/** Role from GET /v1/me, so a demoted admin is turned away at once. Proxy only checks the token. */
async function AdminGate({ children }: { children: React.ReactNode }) {
  const user = await getCurrentUser();
  if (!user) redirect("/login?next=/admin");
  if (user.role !== "ADMIN") redirect("/");
  return children;
}

export default function AdminLayout({ children }: LayoutProps<"/admin">) {
  return (
    <div className="flex min-h-full flex-1 flex-col">
      <header className="sticky top-0 z-40 border-b bg-background/90 backdrop-blur">
        <div className="flex h-14 items-center gap-4 px-4 sm:px-6">
          <Suspense fallback={<span className="size-8 md:hidden" />}>
            <AdminMobileNav />
          </Suspense>
          <Logo />
          <span className="rounded-md bg-secondary px-2 py-0.5 text-sm text-secondary-foreground">Admin</span>
          <Link href="/" className="ml-4 hidden text-sm text-muted-foreground hover:text-foreground sm:inline">
            Ke situs
          </Link>
          <div className="ml-auto flex items-center gap-1.5">
            <ThemeToggle />
            <Suspense fallback={<Skeleton className="size-8 rounded-full" />}>
              <UserNav />
            </Suspense>
          </div>
        </div>
      </header>
      <div className="flex flex-1">
        <aside className="hidden w-56 shrink-0 border-r bg-sidebar p-4 md:block">
          <Suspense fallback={<Skeleton className="h-96 w-full" />}>
            <AdminNav />
          </Suspense>
        </aside>
        <main className="min-w-0 flex-1 px-4 py-8 sm:px-8">
          <Suspense fallback={<AdminSkeleton />}>
            <AdminGate>{children}</AdminGate>
          </Suspense>
        </main>
      </div>
    </div>
  );
}

function AdminSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-9 w-64" />
      <Skeleton className="h-40 w-full" />
    </div>
  );
}
