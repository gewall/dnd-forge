"use client";

import { BookOpenTextIcon, CheckIcon } from "lucide-react";
import { useState, useTransition } from "react";
import { toast } from "sonner";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Textarea } from "@/components/ui/textarea";
import type { components } from "@/lib/api/schema";

import { applyLore, requestLore } from "./actions";
import { optionLabel } from "./labels";
import { upsertById } from "./poll";
import { ChoiceChips, QuotaLine } from "./ui";
import { useJobPoll } from "./use-job-poll";

type Lore = components["schemas"]["LoreDTO"];
type Options = components["schemas"]["OptionsDTO"];
type Usage = components["schemas"]["UsageDTO"];

const PROMPT_MAX = 500;

export function LorePanel({
  characterId,
  options,
  initial,
  usage,
  backstory,
}: {
  characterId: string;
  options: Options | null;
  initial: Lore[];
  usage: Usage | null;
  backstory: string;
}) {
  const tones = options?.tones ?? [];
  const lengths = options?.lengths ?? [];
  const languages = options?.languages ?? [];

  const [lores, setLores] = useState(initial);
  const [tone, setTone] = useState(tones[0]?.id ?? "");
  const [length, setLength] = useState(lengths.some((l) => l.id === "medium") ? "medium" : (lengths[0]?.id ?? ""));
  const [language, setLanguage] = useState(languages.some((l) => l.id === "id") ? "id" : (languages[0]?.id ?? ""));
  const [prompt, setPrompt] = useState("");
  const [pendingId, setPendingId] = useState<string | null>(() => initial.find((l) => l.status === "PENDING")?.id ?? null);
  const [message, setMessage] = useState<string | null>(null);
  const [unavailable, setUnavailable] = useState<string | null>(null);
  const [remaining, setRemaining] = useState(usage?.remaining ?? null);
  const [submitting, startSubmit] = useTransition();
  const [applying, startApply] = useTransition();

  useJobPoll<Lore>(pendingId ? `/api/lore/${pendingId}` : null, {
    onUpdate: (l) => {
      setLores((list) => upsertById(list, l));
      if (l.status === "PENDING") return;
      setPendingId(null);
      if (l.status === "FAILED") {
        setRemaining((r) => (r === null ? r : r + 1));
        setMessage(l.errorMessage || "Draf gagal ditulis. Kuota tidak terpakai; coba lagi.");
      }
    },
    onTimeout: () => {
      setPendingId(null);
      setMessage("Masih ditulis. Buka lagi halaman ini sebentar lagi.");
    },
    onError: (m) => {
      setPendingId(null);
      setMessage(m);
    },
  });

  const submit = () => {
    setMessage(null);
    startSubmit(async () => {
      const result = await requestLore(characterId, { tone, length, language, prompt });
      if (result.ok) {
        setLores((list) => upsertById(list, result.data));
        setPendingId(result.data.id);
        setRemaining((r) => (r === null ? r : Math.max(0, r - 1)));
        return;
      }
      if (result.status === 503) setUnavailable(result.message);
      else setMessage(result.message);
    });
  };

  const apply = (id: string) =>
    startApply(async () => {
      const result = await applyLore(id);
      if (result.ok) toast.success("Draf dipakai sebagai kisah latar.");
      else toast.error(result.message);
    });

  const busy = submitting || pendingId !== null;
  const drafts = lores.filter((l) => l.status !== "FAILED");

  return (
    <section aria-labelledby="lore-title" className="rounded-xl border bg-card p-5">
      <div className="flex items-center gap-2">
        <BookOpenTextIcon className="size-4 text-arcane" aria-hidden="true" />
        <h2 id="lore-title" className="text-xl font-semibold">
          Lore AI
        </h2>
      </div>
      <p className="mt-1 text-sm text-muted-foreground">
        Minta draf kisah latar, lalu pilih yang paling cocok untuk dijadikan kisah latar karakter.
      </p>

      {unavailable ? (
        <Alert className="mt-5">
          <AlertDescription>{unavailable}</AlertDescription>
        </Alert>
      ) : (
        <div className="mt-5 space-y-5">
          {tones.length > 0 && (
            <ChoiceChips
              label="Nada cerita"
              value={tone}
              onChange={setTone}
              disabled={busy}
              options={tones.map((t) => ({ id: t.id, label: optionLabel("tone", t.id, t.description), hint: t.description }))}
            />
          )}
          <div className="grid gap-5 sm:grid-cols-2">
            {lengths.length > 0 && (
              <ChoiceChips
                label="Panjang"
                value={length}
                onChange={setLength}
                disabled={busy}
                options={lengths.map((l) => ({ id: l.id, label: optionLabel("length", l.id, l.description), hint: l.description }))}
              />
            )}
            {languages.length > 0 && (
              <ChoiceChips
                label="Bahasa"
                value={language}
                onChange={setLanguage}
                disabled={busy}
                options={languages.map((l) => ({ id: l.id, label: optionLabel("language", l.id, l.description) }))}
              />
            )}
          </div>
          <Field>
            <FieldLabel htmlFor="lore-prompt">Keinginanmu tentang ceritanya</FieldLabel>
            <Textarea
              id="lore-prompt"
              rows={3}
              maxLength={PROMPT_MAX}
              value={prompt}
              disabled={busy}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Mis. dibesarkan pustakawan, mencari kakak yang hilang"
            />
            <FieldDescription>
              {prompt.length}/{PROMPT_MAX}. Boleh dikosongkan.
            </FieldDescription>
          </Field>
          {message && (
            <Alert variant="destructive">
              <AlertDescription>{message}</AlertDescription>
            </Alert>
          )}
          <div className="flex flex-wrap items-center justify-between gap-3">
            {usage && remaining !== null ? <QuotaLine remaining={remaining} limit={usage.limit} /> : <span />}
            <Button variant="arcane" onClick={submit} disabled={busy || remaining === 0 || !tone}>
              <BookOpenTextIcon />
              {busy ? "Menulis…" : "Tulis draf"}
            </Button>
          </div>
        </div>
      )}

      {drafts.length > 0 && (
        <div className="mt-6 border-t pt-4">
          <h3 className="text-sm font-medium">Draf</h3>
          <ul className="mt-3 space-y-4">
            {drafts.slice(0, 10).map((l) => (
              <LoreDraft key={l.id} lore={l} inUse={!!l.content && l.content.trim() === backstory.trim()} onApply={apply} applying={applying} />
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

function LoreDraft({
  lore,
  inUse,
  onApply,
  applying,
}: {
  lore: Lore;
  inUse: boolean;
  onApply: (id: string) => void;
  applying: boolean;
}) {
  const [open, setOpen] = useState(false);
  const meta = `${optionLabel("tone", lore.tone)}, ${optionLabel("length", lore.length).toLowerCase()}`;

  if (lore.status === "PENDING") {
    return (
      <li className="rounded-lg border border-dashed border-arcane/50 bg-arcane/5 p-4 text-sm text-muted-foreground" role="status">
        Menulis draf {meta.toLowerCase()}…
      </li>
    );
  }

  return (
    <li className="rounded-lg border p-4">
      <p className="text-sm text-muted-foreground">{meta}</p>
      <p className={open ? "mt-2 font-serif leading-relaxed whitespace-pre-line" : "mt-2 line-clamp-4 font-serif leading-relaxed whitespace-pre-line"}>
        {lore.content}
      </p>
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <Button variant="ghost" size="sm" onClick={() => setOpen((v) => !v)} aria-expanded={open}>
          {open ? "Ringkas" : "Baca semua"}
        </Button>
        {inUse ? (
          <span className="flex items-center gap-1 text-sm text-arcane">
            <CheckIcon className="size-4" aria-hidden="true" />
            Sedang dipakai
          </span>
        ) : (
          <Button variant="outline" size="sm" onClick={() => onApply(lore.id)} disabled={applying}>
            Pakai sebagai kisah latar
          </Button>
        )}
      </div>
    </li>
  );
}
