"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { components } from "@/lib/api/schema";

import { slugify } from "../form-utils";
import {
  deleteBackground,
  deleteClass,
  deleteRace,
  deleteSubrace,
  saveBackground,
  saveClass,
  saveRace,
  saveSubrace,
} from "./actions";
import {
  cleanTraits,
  ConfirmDelete,
  errorFor,
  ImageField,
  NumberField,
  SwitchField,
  TextAreaField,
  TextField,
  type TraitRow,
  TraitsField,
  useAdminAction,
} from "./fields";

type S = components["schemas"];

const toRows = (traits?: S["TraitDTO"][] | null): TraitRow[] =>
  (traits ?? []).map((t) => ({ name: t.name, description: t.description ?? "" }));

/** Name, slug (follows the name until edited), description, visual prompt, order, active. */
function useBaseFields(initial?: { name: string; slug: string; description: string; visualPrompt: string; sortOrder?: number; isActive: boolean }) {
  const [name, setName] = useState(initial?.name ?? "");
  const [slug, setSlug] = useState(initial?.slug ?? "");
  const [slugTouched, setSlugTouched] = useState(!!initial);
  const [description, setDescription] = useState(initial?.description ?? "");
  const [visualPrompt, setVisualPrompt] = useState(initial?.visualPrompt ?? "");
  const [sortOrder, setSortOrder] = useState(initial?.sortOrder ?? 0);
  const [isActive, setIsActive] = useState(initial?.isActive ?? true);
  return {
    values: { name: name.trim(), slug, description: description.trim(), visualPrompt: visualPrompt.trim(), sortOrder, isActive },
    render: (prefix: string, errors: Record<string, string>, opts: { order?: boolean } = { order: true }) => (
      <>
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField
            id={`${prefix}-name`}
            label="Nama"
            required
            maxLength={80}
            value={name}
            error={errors.name}
            onChange={(v) => {
              setName(v);
              if (!slugTouched) setSlug(slugify(v));
            }}
          />
          <TextField
            id={`${prefix}-slug`}
            label="Slug"
            required
            mono
            maxLength={64}
            value={slug}
            error={errors.slug}
            onChange={(v) => {
              setSlugTouched(true);
              setSlug(v);
            }}
          />
        </div>
        <TextAreaField id={`${prefix}-desc`} label="Deskripsi" rows={4} maxLength={8000} value={description} error={errors.description} onChange={setDescription} />
        <TextAreaField
          id={`${prefix}-visual`}
          label="Visual prompt (bahasa Inggris)"
          rows={2}
          maxLength={500}
          value={visualPrompt}
          error={errors.visualPrompt}
          onChange={setVisualPrompt}
          description="Dimasukkan ke prompt portrait AI."
        />
        <div className="flex flex-wrap items-end gap-8">
          {opts.order && <NumberField id={`${prefix}-order`} label="Urutan" value={sortOrder} error={errors.sortOrder} onChange={setSortOrder} />}
          <SwitchField id={`${prefix}-active`} label="Aktif (tampil dan bisa dipilih)" checked={isActive} onChange={setIsActive} />
        </div>
      </>
    ),
  };
}

function SaveRow({ isNew, pending, onSave, deleteTitle, onDelete }: { isNew: boolean; pending: boolean; onSave: () => void; deleteTitle?: string; onDelete?: () => void }) {
  return (
    <div className="flex flex-wrap gap-2 border-t pt-4">
      <Button onClick={onSave} disabled={pending}>
        {pending ? "Menyimpan…" : isNew ? "Buat" : "Simpan"}
      </Button>
      {!isNew && onDelete && deleteTitle && (
        <ConfirmDelete
          title={deleteTitle}
          description="Ditolak bila masih dipakai karakter. Nonaktifkan saja kalau ingin menyembunyikannya."
          onConfirm={onDelete}
          disabled={pending}
        />
      )}
    </div>
  );
}

export function RaceForm({ race }: { race?: S["RaceDTO"] }) {
  const base = useBaseFields(race);
  const [traits, setTraits] = useState<TraitRow[]>(() => toRows(race?.traits));
  const [image, setImage] = useState({ key: race?.imageKey ?? "", url: race?.imageUrl });
  const save = useAdminAction(race ? "Ras disimpan." : "Ras dibuat.");
  const del = useAdminAction("Ras dihapus.");

  return (
    <FieldGroup className="max-w-3xl">
      {base.render("r", save.errors)}
      <ImageField id="r-image" label="Gambar" imageKey={image.key} imageUrl={image.url} onChange={(key, url) => setImage({ key, url })} />
      <TraitsField label="Trait" value={traits} onChange={setTraits} error={errorFor(save.errors, "traits")} />
      <SaveRow
        isNew={!race}
        pending={save.pending || del.pending}
        onSave={() => save.run(() => saveRace(race?.id ?? null, { ...base.values, traits: cleanTraits(traits), imageKey: image.key }))}
        deleteTitle={race ? `Hapus ras ${race.name}?` : undefined}
        onDelete={race ? () => del.run(() => deleteRace(race.id)) : undefined}
      />
    </FieldGroup>
  );
}

function SubraceForm({ raceId, subrace, onDone }: { raceId: string; subrace?: S["SubraceDTO"]; onDone?: () => void }) {
  const base = useBaseFields(subrace ? { ...subrace, sortOrder: 0 } : undefined);
  const [traits, setTraits] = useState<TraitRow[]>(() => toRows(subrace?.traits));
  const save = useAdminAction(subrace ? "Subras disimpan." : "Subras ditambahkan.");
  const del = useAdminAction("Subras dihapus.");
  const prefix = `s-${subrace?.id ?? "new"}`;

  const { sortOrder: _ignored, ...values } = base.values;
  return (
    <FieldGroup>
      {base.render(prefix, save.errors, { order: false })}
      <TraitsField label="Trait subras" value={traits} onChange={setTraits} error={errorFor(save.errors, "traits")} />
      <div className="flex flex-wrap gap-2">
        <Button
          onClick={() =>
            save.run(async () => {
              const r = await saveSubrace(raceId, subrace?.id ?? null, { ...values, traits: cleanTraits(traits) });
              if (r.ok) onDone?.();
              return r;
            })
          }
          disabled={save.pending}
        >
          {save.pending ? "Menyimpan…" : subrace ? "Simpan subras" : "Tambah subras"}
        </Button>
        {subrace && (
          <ConfirmDelete
            title={`Hapus subras ${subrace.name}?`}
            description="Ditolak bila masih dipakai karakter."
            disabled={del.pending}
            onConfirm={() => del.run(() => deleteSubrace(subrace.id))}
          />
        )}
        {onDone && (
          <Button variant="ghost" onClick={onDone}>
            Tutup
          </Button>
        )}
      </div>
    </FieldGroup>
  );
}

export function SubraceManager({ raceId, subraces }: { raceId: string; subraces: S["SubraceDTO"][] }) {
  const [open, setOpen] = useState<string | null>(null);
  return (
    <div className="max-w-3xl space-y-3">
      {subraces.length === 0 && <p className="text-muted-foreground">Ras ini belum punya subras.</p>}
      {subraces.map((s) => (
        <div key={s.id} className="rounded-xl border p-4">
          {open === s.id ? (
            <SubraceForm raceId={raceId} subrace={s} onDone={() => setOpen(null)} />
          ) : (
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="font-medium">
                  {s.name} {!s.isActive && <span className="text-sm text-muted-foreground">(nonaktif)</span>}
                </p>
                <p className="font-mono text-xs text-muted-foreground">{s.slug}</p>
              </div>
              <Button size="sm" variant="outline" onClick={() => setOpen(s.id)}>
                Ubah
              </Button>
            </div>
          )}
        </div>
      ))}
      <div className="rounded-xl border border-dashed p-4">
        {open === "new" ? (
          <SubraceForm raceId={raceId} onDone={() => setOpen(null)} />
        ) : (
          <Button variant="outline" onClick={() => setOpen("new")}>
            Tambah subras
          </Button>
        )}
      </div>
    </div>
  );
}

const hitDice = [6, 8, 10, 12] as const;

export function ClassForm({ cls }: { cls?: S["ClassDTO"] }) {
  const base = useBaseFields(cls);
  const [hitDie, setHitDie] = useState<number>(cls?.hitDie ?? 8);
  const [primaryAbility, setPrimaryAbility] = useState(cls?.primaryAbility ?? "");
  const [features, setFeatures] = useState<TraitRow[]>(() => toRows(cls?.features));
  const [image, setImage] = useState({ key: cls?.imageKey ?? "", url: cls?.imageUrl });
  const save = useAdminAction(cls ? "Kelas disimpan." : "Kelas dibuat.");
  const del = useAdminAction("Kelas dihapus.");

  return (
    <FieldGroup className="max-w-3xl">
      {base.render("c", save.errors)}
      <div className="grid gap-4 sm:grid-cols-2">
        <Field data-invalid={!!save.errors.hitDie}>
          <FieldLabel htmlFor="c-hd">Hit die</FieldLabel>
          <Select value={String(hitDie)} onValueChange={(v) => setHitDie(Number(v))}>
            <SelectTrigger id="c-hd" className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {hitDice.map((d) => (
                <SelectItem key={d} value={String(d)}>
                  d{d}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldError>{save.errors.hitDie}</FieldError>
        </Field>
        <TextField id="c-pa" label="Ability utama" required maxLength={60} value={primaryAbility} error={save.errors.primaryAbility} onChange={setPrimaryAbility} />
      </div>
      <ImageField id="c-image" label="Gambar" imageKey={image.key} imageUrl={image.url} onChange={(key, url) => setImage({ key, url })} />
      <TraitsField label="Fitur kelas" value={features} onChange={setFeatures} error={errorFor(save.errors, "features")} />
      <SaveRow
        isNew={!cls}
        pending={save.pending || del.pending}
        onSave={() =>
          save.run(() =>
            saveClass(cls?.id ?? null, {
              ...base.values,
              hitDie: hitDie as S["ClassRequest"]["hitDie"],
              primaryAbility: primaryAbility.trim(),
              features: cleanTraits(features),
              imageKey: image.key,
            }),
          )
        }
        deleteTitle={cls ? `Hapus kelas ${cls.name}?` : undefined}
        onDelete={cls ? () => del.run(() => deleteClass(cls.id)) : undefined}
      />
    </FieldGroup>
  );
}

export function BackgroundForm({ bg }: { bg?: S["BackgroundDTO"] }) {
  const base = useBaseFields(bg);
  const [skills, setSkills] = useState((bg?.skillProficiencies ?? []).join(", "));
  const [image, setImage] = useState({ key: bg?.imageKey ?? "", url: bg?.imageUrl });
  const save = useAdminAction(bg ? "Latar belakang disimpan." : "Latar belakang dibuat.");
  const del = useAdminAction("Latar belakang dihapus.");
  const skillList = skills
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

  return (
    <FieldGroup className="max-w-3xl">
      {base.render("b", save.errors)}
      <TextField
        id="b-skills"
        label="Skill yang dikuasai"
        value={skills}
        error={errorFor(save.errors, "skillProficiencies")}
        onChange={setSkills}
        description="Pisahkan dengan koma, mis. Insight, Religion."
      />
      <ImageField id="b-image" label="Gambar" imageKey={image.key} imageUrl={image.url} onChange={(key, url) => setImage({ key, url })} />
      <SaveRow
        isNew={!bg}
        pending={save.pending || del.pending}
        onSave={() => save.run(() => saveBackground(bg?.id ?? null, { ...base.values, skillProficiencies: skillList, imageKey: image.key }))}
        deleteTitle={bg ? `Hapus latar ${bg.name}?` : undefined}
        onDelete={bg ? () => del.run(() => deleteBackground(bg.id)) : undefined}
      />
    </FieldGroup>
  );
}
