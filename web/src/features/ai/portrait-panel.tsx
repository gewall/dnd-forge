"use client";

import { SparklesIcon } from "lucide-react";
import { useState, useTransition } from "react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Textarea } from "@/components/ui/textarea";
import type { components } from "@/lib/api/schema";
import { cn } from "@/lib/utils";

import { requestPortrait } from "./actions";
import { optionLabel } from "./labels";
import { upsertById } from "./poll";
import { ChoiceChips, QuotaLine, SafeImage } from "./ui";
import { useJobPoll } from "./use-job-poll";

type Portrait = components["schemas"]["PortraitDTO"];
type Style = components["schemas"]["StyleDTO"];
type Usage = components["schemas"]["UsageDTO"];

const ratios = [
  { id: "3:4", label: "Potret 3:4" },
  { id: "1:1", label: "Persegi 1:1" },
  { id: "4:3", label: "Lanskap 4:3" },
] as const;
type Ratio = (typeof ratios)[number]["id"];

const aspectClass: Record<string, string> = { "1:1": "aspect-square", "3:4": "aspect-[3/4]", "4:3": "aspect-[4/3]" };
const PROMPT_MAX = 300;

export function PortraitPanel({
  characterId,
  characterName,
  styles,
  initial,
  usage,
}: {
  characterId: string;
  characterName: string;
  styles: Style[];
  initial: Portrait[];
  usage: Usage | null;
}) {
  const [portraits, setPortraits] = useState(initial);
  const [style, setStyle] = useState(styles[0]?.id ?? "");
  const [ratio, setRatio] = useState<Ratio>("3:4");
  const [prompt, setPrompt] = useState("");
  const [pendingId, setPendingId] = useState<string | null>(() => initial.find((p) => p.status === "PENDING")?.id ?? null);
  const [message, setMessage] = useState<string | null>(null);
  const [unavailable, setUnavailable] = useState<string | null>(null);
  const [remaining, setRemaining] = useState(usage?.remaining ?? null);
  const [submitting, startTransition] = useTransition();

  useJobPoll<Portrait>(pendingId ? `/api/portraits/${pendingId}` : null, {
    onUpdate: (p) => {
      setPortraits((list) => upsertById(list, p));
      if (p.status === "PENDING") return;
      setPendingId(null);
      if (p.status === "FAILED") {
        // Failed jobs do not use quota.
        setRemaining((r) => (r === null ? r : r + 1));
        setMessage(p.errorMessage || "Portrait gagal dibuat. Kuota tidak terpakai; coba lagi.");
      }
    },
    onTimeout: () => {
      setPendingId(null);
      setMessage("Masih diproses. Buka lagi halaman ini beberapa menit lagi.");
    },
    onError: (m) => {
      setPendingId(null);
      setMessage(m);
    },
  });

  const submit = () => {
    setMessage(null);
    startTransition(async () => {
      const result = await requestPortrait(characterId, { style, aspectRatio: ratio, prompt });
      if (result.ok) {
        setPortraits((list) => upsertById(list, result.data));
        setPendingId(result.data.id);
        setRemaining((r) => (r === null ? r : Math.max(0, r - 1)));
        return;
      }
      if (result.status === 503) setUnavailable(result.message);
      else setMessage(result.message);
    });
  };

  const latest = portraits.find((p) => p.status === "SUCCEEDED");
  const pending = portraits.find((p) => p.id === pendingId);
  const busy = submitting || pendingId !== null;
  const outOfQuota = remaining === 0;

  return (
    <section aria-labelledby="portrait-title" className="rounded-xl border bg-card p-5">
      <div className="flex items-center gap-2">
        <SparklesIcon className="size-4 text-arcane" aria-hidden="true" />
        <h2 id="portrait-title" className="text-xl font-semibold">
          Portrait AI
        </h2>
      </div>

      <div className="mt-4">
        {pending ? (
          <div
            className={cn(
              "grid w-full place-items-center rounded-lg border border-dashed border-arcane/50 bg-arcane/5",
              aspectClass[pending.aspectRatio] ?? "aspect-[3/4]",
            )}
            role="status"
          >
            <span className="flex flex-col items-center gap-2 text-sm text-muted-foreground">
              <SparklesIcon className="size-6 animate-pulse text-arcane motion-reduce:animate-none" aria-hidden="true" />
              Sedang melukis {characterName}…
            </span>
          </div>
        ) : latest ? (
          <a href={latest.imageUrl} target="_blank" rel="noopener noreferrer" className="block">
            <SafeImage
              src={latest.imageUrl}
              alt={`Portrait ${characterName}`}
              className={cn("w-full rounded-lg border", aspectClass[latest.aspectRatio] ?? "aspect-[3/4]")}
            />
          </a>
        ) : (
          <p className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
            Belum ada portrait. Pilih gaya lalu buat yang pertama.
          </p>
        )}
      </div>

      {unavailable ? (
        <Alert className="mt-5">
          <AlertDescription>{unavailable}</AlertDescription>
        </Alert>
      ) : (
        <div className="mt-5 space-y-5">
          {styles.length > 0 && (
            <ChoiceChips
              label="Gaya"
              value={style}
              onChange={setStyle}
              disabled={busy}
              options={styles.map((s) => ({ id: s.id, label: optionLabel("style", s.id, s.description), hint: s.description }))}
            />
          )}
          <ChoiceChips
            label="Bentuk"
            value={ratio}
            onChange={(v) => setRatio(v as Ratio)}
            disabled={busy}
            options={ratios.map((r) => ({ id: r.id, label: r.label }))}
          />
          <Field>
            <FieldLabel htmlFor="portrait-prompt">Detail tambahan</FieldLabel>
            <Textarea
              id="portrait-prompt"
              rows={3}
              maxLength={PROMPT_MAX}
              value={prompt}
              disabled={busy}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Mis. memegang tongkat kristal, latar hutan berkabut"
            />
            <FieldDescription>
              {prompt.length}/{PROMPT_MAX}. Ras, kelas, dan penampilan sudah otomatis disertakan.
            </FieldDescription>
          </Field>
          {message && (
            <Alert variant="destructive">
              <AlertDescription>{message}</AlertDescription>
            </Alert>
          )}
          <div className="flex flex-wrap items-center justify-between gap-3">
            {usage && remaining !== null ? <QuotaLine remaining={remaining} limit={usage.limit} /> : <span />}
            <Button variant="arcane" onClick={submit} disabled={busy || outOfQuota || !style}>
              <SparklesIcon />
              {busy ? "Memproses…" : latest ? "Buat portrait baru" : "Buat portrait"}
            </Button>
          </div>
        </div>
      )}

      {portraits.filter((p) => p.status === "SUCCEEDED").length > 1 && (
        <div className="mt-6 border-t pt-4">
          <h3 className="text-sm font-medium">Portrait sebelumnya</h3>
          <ul className="mt-3 grid grid-cols-4 gap-2">
            {portraits
              .filter((p) => p.status === "SUCCEEDED" && p.id !== latest?.id)
              .slice(0, 8)
              .map((p) => (
                <li key={p.id}>
                  <a href={p.imageUrl} target="_blank" rel="noopener noreferrer" title={optionLabel("style", p.style)}>
                    <SafeImage src={p.thumbUrl} alt={`Portrait ${characterName}, ${optionLabel("style", p.style)}`} className="aspect-square w-full rounded-md border" />
                  </a>
                </li>
              ))}
          </ul>
        </div>
      )}
    </section>
  );
}
