"use client";

import { XIcon } from "lucide-react";
import { useState, useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

import { addBannedWords, removeBannedWord } from "./actions";
import { matchBanned, splitWords } from "./banned";

function AddWords({ existing }: { existing: Set<string> }) {
  const [raw, setRaw] = useState("");
  const [error, setError] = useState<string>();
  const [pending, start] = useTransition();
  const words = splitWords(raw).filter((w) => !existing.has(w));

  const submit = () =>
    start(async () => {
      setError(undefined);
      const r = await addBannedWords(words);
      if (r.ok) {
        toast.success(`${r.data} kata ditambahkan.`);
        setRaw("");
      } else {
        setError(r.fieldErrors.word ?? r.message);
        toast.error(r.message);
      }
    });

  return (
    <FieldGroup>
      <Field data-invalid={!!error}>
        <FieldLabel htmlFor="bw-add">Tambah kata atau frasa</FieldLabel>
        <Textarea id="bw-add" rows={3} value={raw} onChange={(e) => setRaw(e.target.value)} aria-invalid={!!error} placeholder="Pisahkan dengan koma atau baris baru" />
        <FieldDescription>
          Masing-masing 2–60 karakter. Huruf besar dan kecil dianggap sama.
          {raw.trim() && ` ${words.length} kata baru.`}
        </FieldDescription>
        <FieldError>{error}</FieldError>
      </Field>
      <div>
        <Button onClick={submit} disabled={pending || words.length === 0}>
          {pending ? "Menambahkan…" : "Tambah"}
        </Button>
      </div>
    </FieldGroup>
  );
}

function WordChip({ word, highlighted }: { word: string; highlighted: boolean }) {
  const [pending, start] = useTransition();
  const remove = () =>
    start(async () => {
      const r = await removeBannedWord(word);
      if (r.ok) toast.success(`“${word}” dihapus dari daftar.`);
      else toast.error(r.message);
    });
  return (
    <li
      data-match={highlighted}
      className="inline-flex items-center gap-1 rounded-full border py-0.5 pr-0.5 pl-3 text-sm data-[match=true]:border-destructive data-[match=true]:bg-destructive/10"
    >
      {word}
      <Button variant="ghost" size="icon-xs" className="rounded-full" aria-label={`Hapus ${word}`} onClick={remove} disabled={pending}>
        <XIcon />
      </Button>
    </li>
  );
}

export function BannedWordsManager({ words }: { words: string[] }) {
  const sorted = [...words].sort((a, b) => a.localeCompare(b, "id"));
  const [filter, setFilter] = useState("");
  const [sample, setSample] = useState("");
  const hits = new Set(matchBanned(sample, sorted));
  const q = filter.trim().toLowerCase();
  const shown = q ? sorted.filter((w) => w.includes(q)) : sorted;

  return (
    <div className="grid gap-10 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <section aria-labelledby="bw-list">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <h2 id="bw-list" className="text-xl font-semibold">
            Daftar <span className="font-normal text-muted-foreground tabular-nums">({sorted.length})</span>
          </h2>
          {sorted.length > 12 && (
            <Input aria-label="Saring daftar" placeholder="Saring…" value={filter} onChange={(e) => setFilter(e.target.value)} className="w-48" />
          )}
        </div>
        {sorted.length === 0 ? (
          <p className="mt-4 text-muted-foreground">Belum ada kata terlarang. Prompt user hanya dibersihkan dari karakter kontrol.</p>
        ) : shown.length === 0 ? (
          <p className="mt-4 text-muted-foreground">Tidak ada yang cocok dengan “{filter}”.</p>
        ) : (
          <ul className="mt-4 flex flex-wrap gap-2">
            {shown.map((w) => (
              <WordChip key={w} word={w} highlighted={hits.has(w)} />
            ))}
          </ul>
        )}
      </section>

      <div className="space-y-10">
        <section aria-labelledby="bw-new">
          <h2 id="bw-new" className="sr-only">
            Tambah
          </h2>
          <AddWords existing={new Set(sorted)} />
        </section>
        <section aria-labelledby="bw-test" className="border-t pt-6">
          <h2 id="bw-test" className="mb-3 font-semibold">
            Coba prompt
          </h2>
          <Field>
            <FieldLabel htmlFor="bw-sample" className="sr-only">
              Contoh prompt user
            </FieldLabel>
            <Textarea id="bw-sample" rows={3} value={sample} onChange={(e) => setSample(e.target.value)} placeholder="Tulis contoh prompt user" />
            <FieldDescription aria-live="polite">
              {!sample.trim()
                ? "Cocok hanya per kata utuh: “ass” tidak memblokir “class”."
                : hits.size === 0
                  ? "Lolos: prompt ini tidak mengandung kata terlarang."
                  : `Ditolak karena: ${[...hits].join(", ")}.`}
            </FieldDescription>
          </Field>
        </section>
      </div>
    </div>
  );
}
