"use client";

import { MenuIcon } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

import { cn } from "@/lib/utils";

const groups: { label: string; items: { href: string; label: string }[] }[] = [
  {
    label: "Umum",
    items: [
      { href: "/admin", label: "Ringkasan" },
      { href: "/admin/users", label: "User" },
      { href: "/admin/settings", label: "Kuota default" },
    ],
  },
  {
    label: "Konten",
    items: [
      { href: "/admin/guides", label: "Artikel panduan" },
      { href: "/admin/guide-categories", label: "Kategori panduan" },
      { href: "/admin/races", label: "Ras" },
      { href: "/admin/classes", label: "Kelas" },
      { href: "/admin/backgrounds", label: "Latar belakang" },
    ],
  },
  {
    label: "AI",
    items: [
      { href: "/admin/ai", label: "Provider" },
      { href: "/admin/prompts", label: "Template prompt" },
      { href: "/admin/banned-words", label: "Kata terlarang" },
    ],
  },
  {
    label: "Pengawasan",
    items: [
      { href: "/admin/images", label: "Moderasi gambar" },
      { href: "/admin/audit", label: "Audit log" },
    ],
  },
];

function isActive(pathname: string, href: string): boolean {
  return href === "/admin" ? pathname === "/admin" : pathname === href || pathname.startsWith(`${href}/`);
}

export function AdminNav() {
  const pathname = usePathname();
  return (
    <nav aria-label="Navigasi admin" className="space-y-6">
      {groups.map((g) => (
        <div key={g.label}>
          <p className="px-2 text-xs font-medium text-muted-foreground">{g.label}</p>
          <ul className="mt-1 space-y-0.5">
            {g.items.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  aria-current={isActive(pathname, item.href) ? "page" : undefined}
                  className={cn(
                    "block rounded-md px-2 py-1.5 text-[0.95rem] transition-colors",
                    isActive(pathname, item.href)
                      ? "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
                      : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-foreground",
                  )}
                >
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </nav>
  );
}

/** Same navigation in a sheet for small screens. */
export function AdminMobileNav() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  // Close after navigating.
  const [last, setLast] = useState(pathname);
  if (last !== pathname) {
    setLast(pathname);
    setOpen(false);
  }
  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="md:hidden" aria-label="Buka menu admin">
          <MenuIcon />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-64 overflow-y-auto p-4">
        <SheetTitle className="mb-4 font-heading text-lg">Admin</SheetTitle>
        <AdminNav />
      </SheetContent>
    </Sheet>
  );
}
