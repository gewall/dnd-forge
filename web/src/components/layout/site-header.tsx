import Link from "next/link";

import { Button } from "@/components/ui/button";

import { Logo } from "./logo";
import { MobileNav } from "./mobile-nav";
import { mainNav } from "./nav";
import { ThemeToggle } from "./theme-toggle";

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/90 backdrop-blur supports-[backdrop-filter]:bg-background/75">
      <div className="mx-auto flex h-14 max-w-6xl items-center gap-4 px-4 sm:px-6">
        <MobileNav />
        <Logo />
        <nav aria-label="Navigasi utama" className="ml-4 hidden items-center gap-1 md:flex">
          {mainNav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="rounded-md px-2.5 py-1.5 text-[0.95rem] text-muted-foreground transition-colors hover:text-foreground"
            >
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-1.5">
          <ThemeToggle />
          <Button asChild variant="ghost" className="hidden sm:inline-flex">
            <Link href="/login">Masuk</Link>
          </Button>
          <Button asChild className="hidden md:inline-flex">
            <Link href="/builder">Buat karakter</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
