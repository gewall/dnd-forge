"use client";

import { MenuIcon } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

import { mainNav } from "./nav";

export function MobileNav() {
  const [open, setOpen] = useState(false);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="md:hidden" aria-label="Buka menu">
          <MenuIcon />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-72">
        <SheetHeader>
          <SheetTitle className="font-heading text-lg">Menu</SheetTitle>
        </SheetHeader>
        <nav className="flex flex-col gap-1 px-4">
          {mainNav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              onClick={() => setOpen(false)}
              className="rounded-md px-2 py-2 text-base hover:bg-muted"
            >
              {item.label}
            </Link>
          ))}
          <Button asChild className="mt-4">
            <Link href="/builder" onClick={() => setOpen(false)}>
              Buat karakter
            </Link>
          </Button>
        </nav>
      </SheetContent>
    </Sheet>
  );
}
