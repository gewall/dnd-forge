"use client";

import { ArrowLeftIcon, ArrowRightIcon } from "lucide-react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState, useTransition } from "react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { createCharacter, updateCharacter } from "@/features/characters/actions";
import { TraitList } from "@/features/gamedata/components";
import { cn } from "@/lib/utils";

import { ChoiceCards } from "./choice-cards";
import {
  activeSubraces,
  applyPreselect,
  canOpenStep,
  type Draft,
  firstIncompleteStep,
  type GameData,
  limits,
  stepComplete,
  steps,
  toRequest,
  withRace,
} from "./draft";
import { type DraftState, DraftProvider, useDraft } from "./store";

export type WizardMode = { kind: "create" } | { kind: "edit"; id: string; initial: Draft };

export function Wizard({ data, mode }: { data: GameData; mode: WizardMode }) {
  return (
    <DraftProvider persistDraft={mode.kind === "create"} initial={mode.kind === "edit" ? mode.initial : undefined}>
      {(hydrated) => (hydrated ? <WizardBody data={data} mode={mode} /> : <WizardSkeleton />)}
    </DraftProvider>
  );
}

function pickDraft(s: DraftState): Draft {
  return {
    raceId: s.raceId,
    subraceId: s.subraceId,
    classId: s.classId,
    backgroundId: s.backgroundId,
    name: s.name,
    gender: s.gender,
    appearance: s.appearance,
    isPublic: s.isPublic,
  };
}

function WizardBody({ data, mode }: { data: GameData; mode: WizardMode }) {
  const state = useDraft((s) => s);
  const draft = pickDraft(state);
  const router = useRouter();
  const params = useSearchParams();
  const preselected = useRef(false);

  // Encyclopedia links arrive as /builder?race=elf; apply once, then clean the URL.
  useEffect(() => {
    if (mode.kind !== "create" || preselected.current) return;
    preselected.current = true;
    const slugs = { race: params.get("race"), class: params.get("class"), background: params.get("background") };
    if (!slugs.race && !slugs.class && !slugs.background) return;
    const next = applyPreselect(pickDraft(state), data, slugs);
    state.replace(next, firstIncompleteStep(next, data));
    router.replace("/builder", { scroll: false });
  }, [mode.kind, params, state, data, router]);

  const step = canOpenStep(state.step, draft, data) ? state.step : firstIncompleteStep(draft, data);
  const current = steps[step];

  const topRef = useRef<HTMLDivElement>(null);
  const goTo = (i: number) => {
    state.setStep(i);
    topRef.current?.scrollIntoView({ block: "start" });
  };

  return (
    <div ref={topRef} className="scroll-mt-20">
      <StepIndicator current={step} draft={draft} data={data} onSelect={goTo} />
      <div className="mt-8 grid gap-10 lg:grid-cols-[minmax(0,1fr)_17rem]">
        <section aria-labelledby="step-title" className="min-w-0">
          <h2 id="step-title" className="text-2xl font-semibold">
            {stepTitles[current.key]}
          </h2>
          <p className="mt-1 text-muted-foreground">{stepHints[current.key]}</p>
          <div className="mt-6">
            {current.key === "race" && <RaceStep data={data} draft={draft} state={state} />}
            {current.key === "class" && <ClassStep data={data} draft={draft} state={state} />}
            {current.key === "background" && <BackgroundStep data={data} draft={draft} state={state} />}
            {current.key === "details" && <DetailsStep draft={draft} state={state} />}
            {current.key === "summary" && <SummaryStep data={data} draft={draft} mode={mode} onEdit={goTo} />}
          </div>
          {current.key !== "summary" && (
            <div className="mt-10 flex items-center justify-between gap-3 border-t pt-6">
              <Button variant="ghost" onClick={() => goTo(step - 1)} disabled={step === 0}>
                <ArrowLeftIcon />
                Kembali
              </Button>
              <Button
                size="lg"
                className="h-10 px-4"
                onClick={() => goTo(step + 1)}
                disabled={!stepComplete(current.key, draft, data)}
              >
                Lanjut ke {steps[step + 1].label.toLowerCase()}
                <ArrowRightIcon />
              </Button>
            </div>
          )}
        </section>
        <DraftSummary data={data} draft={draft} />
      </div>
    </div>
  );
}

const stepTitles: Record<(typeof steps)[number]["key"], string> = {
  race: "Pilih ras",
  class: "Pilih kelas",
  background: "Pilih latar belakang",
  details: "Nama dan penampilan",
  summary: "Periksa karaktermu",
};

const stepHints: Record<(typeof steps)[number]["key"], string> = {
  race: "Ras menentukan asal-usul dan kemampuan bawaan karaktermu.",
  class: "Kelas menentukan cara karaktermu bertarung dan berkembang.",
  background: "Latar belakang adalah hidup karaktermu sebelum berpetualang.",
  details: "Penampilan juga dipakai sebagai bahan portrait AI nanti.",
  summary: "Masih bisa diubah lewat tombol Ubah di tiap bagian.",
};

function StepIndicator({
  current,
  draft,
  data,
  onSelect,
}: {
  current: number;
  draft: Draft;
  data: GameData;
  onSelect: (i: number) => void;
}) {
  return (
    <ol className="flex gap-1 overflow-x-auto border-b pb-px" aria-label="Langkah membuat karakter">
      {steps.map((s, i) => {
        const done = stepComplete(s.key, draft, data) && s.key !== "summary";
        const open = canOpenStep(i, draft, data);
        return (
          <li key={s.key} className="shrink-0">
            <button
              type="button"
              disabled={!open}
              onClick={() => onSelect(i)}
              aria-current={i === current ? "step" : undefined}
              className={cn(
                "-mb-px flex items-center gap-2 border-b-2 px-3 py-2.5 text-[0.95rem] transition-colors disabled:cursor-not-allowed disabled:opacity-50",
                i === current ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
              )}
            >
              <span
                className={cn(
                  "grid size-6 place-items-center rounded-full border text-xs tabular-nums",
                  done && "border-primary bg-primary text-primary-foreground",
                )}
              >
                {i + 1}
              </span>
              {s.label}
            </button>
          </li>
        );
      })}
    </ol>
  );
}

function DetailPanel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mt-8 rounded-xl border bg-card p-5">
      <h3 className="text-lg font-semibold">{title}</h3>
      <div className="mt-4">{children}</div>
    </div>
  );
}

function RaceStep({ data, draft, state }: { data: GameData; draft: Draft; state: DraftState }) {
  const race = data.races.find((r) => r.id === draft.raceId);
  const subs = activeSubraces(race);
  return (
    <>
      <ChoiceCards
        label="Ras"
        value={draft.raceId}
        onChange={(id) => state.replace(withRace(draft, id, data), state.step)}
        choices={data.races.map((r) => {
          const n = activeSubraces(r).length;
          return { id: r.id, name: r.name, description: r.description, facts: n ? [`${n} subras`] : undefined };
        })}
      />
      {race && subs.length > 0 && (
        <div className="mt-8">
          <h3 className="text-lg font-semibold">Subras {race.name}</h3>
          <p className="mt-1 mb-4 text-sm text-muted-foreground">Wajib dipilih untuk ras ini.</p>
          <ChoiceCards
            label={`Subras ${race.name}`}
            columns={2}
            value={draft.subraceId}
            onChange={(id) => state.update({ subraceId: id })}
            choices={subs.map((s) => ({ id: s.id, name: s.name, description: s.description }))}
          />
        </div>
      )}
      {race && (race.traits?.length ?? 0) > 0 && (
        <DetailPanel title={`Trait ${race.name}`}>
          <TraitList traits={race.traits ?? []} />
        </DetailPanel>
      )}
    </>
  );
}

function ClassStep({ data, draft, state }: { data: GameData; draft: Draft; state: DraftState }) {
  const cls = data.classes.find((c) => c.id === draft.classId);
  return (
    <>
      <ChoiceCards
        label="Kelas"
        value={draft.classId}
        onChange={(id) => state.update({ classId: id })}
        choices={data.classes.map((c) => ({
          id: c.id,
          name: c.name,
          description: c.description,
          facts: [`d${c.hitDie}`, c.primaryAbility],
        }))}
      />
      {cls && (cls.features?.length ?? 0) > 0 && (
        <DetailPanel title={`Fitur ${cls.name}`}>
          <TraitList traits={cls.features ?? []} />
        </DetailPanel>
      )}
    </>
  );
}

function BackgroundStep({ data, draft, state }: { data: GameData; draft: Draft; state: DraftState }) {
  return (
    <ChoiceCards
      label="Latar belakang"
      columns={2}
      value={draft.backgroundId}
      onChange={(id) => state.update({ backgroundId: id })}
      choices={data.backgrounds.map((b) => ({
        id: b.id,
        name: b.name,
        description: b.description,
        facts: b.skillProficiencies ?? undefined,
      }))}
    />
  );
}

function DetailsStep({ draft, state }: { draft: Draft; state: DraftState }) {
  const [touched, setTouched] = useState(false);
  const nameError =
    touched && !draft.name.trim()
      ? "Beri nama karaktermu."
      : draft.name.trim().length > limits.name
        ? `Nama maksimal ${limits.name} karakter.`
        : undefined;

  return (
    <FieldGroup className="max-w-xl">
      <Field data-invalid={!!nameError}>
        <FieldLabel htmlFor="char-name">Nama</FieldLabel>
        <Input
          id="char-name"
          value={draft.name}
          maxLength={limits.name}
          onChange={(e) => state.update({ name: e.target.value })}
          onBlur={() => setTouched(true)}
          aria-invalid={!!nameError}
          autoComplete="off"
        />
        <FieldError>{nameError}</FieldError>
      </Field>
      <Field>
        <FieldLabel htmlFor="char-gender">Gender</FieldLabel>
        <Input
          id="char-gender"
          value={draft.gender}
          maxLength={limits.gender}
          onChange={(e) => state.update({ gender: e.target.value })}
          placeholder="Boleh dikosongkan"
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="char-appearance">Penampilan</FieldLabel>
        <Textarea
          id="char-appearance"
          rows={5}
          value={draft.appearance}
          maxLength={limits.appearance}
          onChange={(e) => state.update({ appearance: e.target.value })}
          placeholder="Mis. rambut perak dikepang, bekas luka di pipi kiri, jubah biru tua."
        />
        <FieldDescription>
          {draft.appearance.length}/{limits.appearance} karakter. Bagian ini jadi bahan portrait AI.
        </FieldDescription>
      </Field>
      <Field orientation="horizontal">
        <Switch id="char-public" checked={draft.isPublic} onCheckedChange={(v) => state.update({ isPublic: v })} />
        <FieldLabel htmlFor="char-public" className="font-normal">
          Tampilkan di galeri publik bersama portrait-nya
        </FieldLabel>
      </Field>
    </FieldGroup>
  );
}

function SummaryStep({
  data,
  draft,
  mode,
  onEdit,
}: {
  data: GameData;
  draft: Draft;
  mode: WizardMode;
  onEdit: (step: number) => void;
}) {
  const reset = useDraft((s) => s.reset);
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [error, setError] = useState<{ message: string; needsLogin: boolean } | null>(null);

  const race = data.races.find((r) => r.id === draft.raceId);
  const sub = activeSubraces(race).find((s) => s.id === draft.subraceId);
  const cls = data.classes.find((c) => c.id === draft.classId);
  const bg = data.backgrounds.find((b) => b.id === draft.backgroundId);

  const save = () => {
    setError(null);
    startTransition(async () => {
      const body = toRequest(draft);
      const result = mode.kind === "create" ? await createCharacter(body) : await updateCharacter(mode.id, body);
      if (result.ok) {
        // Edit redirects on the server; only creation lands here.
        if (mode.kind === "create" && result.data) {
          reset();
          router.push(`/characters/${result.data.id}`);
        }
        return;
      }
      setError({ message: result.message, needsLogin: result.status === 401 });
    });
  };

  const rows: { label: string; value: string; step: number }[] = [
    { label: "Ras", value: sub ? `${race?.name} (${sub.name})` : (race?.name ?? "-"), step: 0 },
    { label: "Kelas", value: cls ? `${cls.name}, hit die d${cls.hitDie}` : "-", step: 1 },
    { label: "Latar belakang", value: bg?.name ?? "-", step: 2 },
    { label: "Nama", value: draft.name.trim() || "-", step: 3 },
    { label: "Gender", value: draft.gender.trim() || "Tidak diisi", step: 3 },
    { label: "Galeri publik", value: draft.isPublic ? "Ditampilkan" : "Tidak ditampilkan", step: 3 },
  ];

  return (
    <div className="max-w-2xl">
      <dl className="divide-y rounded-xl border bg-card">
        {rows.map((r) => (
          <div key={r.label} className="flex items-center gap-4 px-4 py-3">
            <dt className="w-32 shrink-0 text-muted-foreground">{r.label}</dt>
            <dd className="min-w-0 flex-1 truncate">{r.value}</dd>
            <Button variant="link" size="sm" onClick={() => onEdit(r.step)} aria-label={`Ubah ${r.label.toLowerCase()}`}>
              Ubah
            </Button>
          </div>
        ))}
        {draft.appearance.trim() && (
          <div className="px-4 py-3">
            <dt className="text-muted-foreground">Penampilan</dt>
            <dd className="mt-1 whitespace-pre-line">{draft.appearance.trim()}</dd>
          </div>
        )}
      </dl>

      {error && (
        <Alert variant="destructive" className="mt-6">
          <AlertDescription>
            {error.needsLogin ? (
              <>
                Masuk dulu untuk menyimpan. Pilihanmu tetap tersimpan di perangkat ini.{" "}
                <Link href="/login?next=/builder" className="font-medium underline underline-offset-2">
                  Masuk
                </Link>{" "}
                atau{" "}
                <Link href="/register?next=/builder" className="font-medium underline underline-offset-2">
                  daftar
                </Link>
                .
              </>
            ) : (
              error.message
            )}
          </AlertDescription>
        </Alert>
      )}

      <div className="mt-8 flex items-center justify-between gap-3 border-t pt-6">
        <Button variant="ghost" onClick={() => onEdit(3)}>
          <ArrowLeftIcon />
          Kembali
        </Button>
        <Button size="lg" className="h-10 px-5 text-base" onClick={save} disabled={pending}>
          {pending ? "Menyimpan…" : mode.kind === "create" ? "Simpan karakter" : "Simpan perubahan"}
        </Button>
      </div>
    </div>
  );
}

function DraftSummary({ data, draft }: { data: GameData; draft: Draft }) {
  const race = data.races.find((r) => r.id === draft.raceId);
  const sub = activeSubraces(race).find((s) => s.id === draft.subraceId);
  const cls = data.classes.find((c) => c.id === draft.classId);
  const bg = data.backgrounds.find((b) => b.id === draft.backgroundId);
  const items = [
    { label: "Ras", value: sub?.name ?? race?.name },
    { label: "Kelas", value: cls?.name },
    { label: "Latar belakang", value: bg?.name },
  ];

  return (
    <aside className="hidden lg:block">
      <div className="sticky top-20 rounded-xl border bg-card p-5">
        <p className="font-heading text-xl font-semibold">{draft.name.trim() || "Karakter tanpa nama"}</p>
        <dl className="mt-4 space-y-3 text-sm">
          {items.map((i) => (
            <div key={i.label}>
              <dt className="text-muted-foreground">{i.label}</dt>
              <dd className={cn("mt-0.5", !i.value && "text-muted-foreground/60")}>{i.value ?? "Belum dipilih"}</dd>
            </div>
          ))}
        </dl>
      </div>
    </aside>
  );
}

export function WizardSkeleton() {
  return (
    <div>
      <Skeleton className="h-11 w-full" />
      <div className="mt-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-32" />
        ))}
      </div>
    </div>
  );
}
