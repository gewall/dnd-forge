"use client";

import { CheckIcon } from "lucide-react";
import { RadioGroup as RadioGroupPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

export interface Choice {
  id: string;
  name: string;
  description?: string;
  facts?: string[];
}

/**
 * Single-choice cards built on Radix RadioGroup: arrow keys move between
 * options and screen readers hear a radio group.
 */
export function ChoiceCards({
  label,
  choices,
  value,
  onChange,
  columns = 3,
}: {
  label: string;
  choices: Choice[];
  value?: string;
  onChange: (id: string) => void;
  columns?: 2 | 3;
}) {
  return (
    <RadioGroupPrimitive.Root
      aria-label={label}
      value={value ?? ""}
      onValueChange={onChange}
      className={cn("grid gap-3 sm:grid-cols-2", columns === 3 && "lg:grid-cols-3")}
    >
      {choices.map((c) => (
        <RadioGroupPrimitive.Item
          key={c.id}
          value={c.id}
          className={cn(
            "group relative flex flex-col items-start rounded-xl border bg-card p-4 text-left transition-colors outline-none",
            "hover:border-muted-foreground/40 focus-visible:ring-3 focus-visible:ring-ring/50",
            "data-[state=checked]:border-primary data-[state=checked]:bg-primary/5",
          )}
        >
          <span className="flex w-full items-start justify-between gap-3">
            <span className="font-heading text-lg leading-snug font-semibold">{c.name}</span>
            <span
              aria-hidden="true"
              className="mt-0.5 grid size-5 shrink-0 place-items-center rounded-full border text-primary-foreground group-data-[state=checked]:border-primary group-data-[state=checked]:bg-primary"
            >
              <CheckIcon className="hidden size-3.5 group-data-[state=checked]:block" />
            </span>
          </span>
          {c.description && <span className="mt-1.5 line-clamp-3 text-sm text-muted-foreground">{c.description}</span>}
          {c.facts && c.facts.length > 0 && (
            <span className="mt-3 flex flex-wrap gap-1.5">
              {c.facts.map((f) => (
                <span key={f} className="rounded-md bg-secondary px-1.5 py-0.5 text-xs text-secondary-foreground">
                  {f}
                </span>
              ))}
            </span>
          )}
        </RadioGroupPrimitive.Item>
      ))}
    </RadioGroupPrimitive.Root>
  );
}
