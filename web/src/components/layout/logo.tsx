import Link from "next/link";

import { cn } from "@/lib/utils";

/** Twenty-sided die seen face-on: a hexagon around a centred triangle. */
export function D20Mark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className={cn("size-6", className)}>
      <g fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round">
        <path d="M12 2.5 20.5 7.25v9.5L12 21.5 3.5 16.75v-9.5Z" />
        <path d="M12 7.5 16.5 15h-9Z" />
        <path d="M12 2.5v5M3.5 7.25 7.5 15M20.5 7.25 16.5 15M7.5 15 3.5 16.75M16.5 15l4 1.75M7.5 15 12 21.5l4.5-6.5" />
      </g>
    </svg>
  );
}

export function Logo() {
  return (
    <Link href="/" className="flex items-center gap-2 rounded-md text-foreground outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
      <D20Mark className="text-primary" />
      <span className="font-heading text-xl font-semibold tracking-tight">DnD Forge</span>
    </Link>
  );
}
