"use client";

import { ImageOffIcon } from "lucide-react";
import { RadioGroup as RadioGroupPrimitive } from "radix-ui";
import { useState } from "react";

import { cn } from "@/lib/utils";

/** Portrait image that falls back to a placeholder (hidden by a moderator, or files deleted → 404). */
export function SafeImage({ src, alt, className }: { src?: string; alt: string; className?: string }) {
  const [broken, setBroken] = useState(false);
  if (!src || broken) {
    return (
      <div className={cn("grid place-items-center bg-muted text-muted-foreground", className)} role="img" aria-label={alt}>
        <ImageOffIcon className="size-6" aria-hidden="true" />
      </div>
    );
  }
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={src} alt={alt} loading="lazy" onError={() => setBroken(true)} className={cn("object-cover", className)} />;
}

export interface ChipOption {
  id: string;
  label: string;
  hint?: string;
}

/** Compact single choice (style, tone, length) as a row of chips. */
export function ChoiceChips({
  label,
  options,
  value,
  onChange,
  disabled,
}: {
  label: string;
  options: ChipOption[];
  value: string;
  onChange: (id: string) => void;
  disabled?: boolean;
}) {
  return (
    <fieldset disabled={disabled} className="min-w-0">
      <legend className="mb-2 text-sm font-medium">{label}</legend>
      <RadioGroupPrimitive.Root
        aria-label={label}
        value={value}
        onValueChange={onChange}
        disabled={disabled}
        className="flex flex-wrap gap-2"
      >
        {options.map((o) => (
          <RadioGroupPrimitive.Item
            key={o.id}
            value={o.id}
            title={o.hint}
            className={cn(
              "rounded-full border px-3 py-1 text-sm transition-colors outline-none",
              "hover:border-muted-foreground/50 focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50",
              "data-[state=checked]:border-arcane data-[state=checked]:bg-arcane/15 data-[state=checked]:text-foreground",
            )}
          >
            {o.label}
          </RadioGroupPrimitive.Item>
        ))}
      </RadioGroupPrimitive.Root>
    </fieldset>
  );
}

/** "3 dari 5 tersisa hari ini" with a reset hint. */
export function QuotaLine({ remaining, limit }: { remaining: number; limit: number }) {
  return (
    <p className={cn("text-sm", remaining === 0 ? "text-destructive" : "text-muted-foreground")}>
      {remaining === 0 ? "Kuota hari ini habis. Kembali penuh tengah malam WIB." : `${remaining} dari ${limit} tersisa hari ini.`}
    </p>
  );
}
