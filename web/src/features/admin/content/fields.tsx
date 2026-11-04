"use client";

import { ImageUpIcon, PlusIcon, Trash2Icon, XIcon } from "lucide-react";
import { useRef, useState, useTransition } from "react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { ActionResult } from "@/lib/api/errors";

import { MAX_UPLOAD_BYTES, UPLOAD_TYPES } from "./upload-limits";

/** Runs a save/delete action with a toast; field errors come back for the form. */
export function useAdminAction(successMessage: string) {
  const [pending, startTransition] = useTransition();
  const [errors, setErrors] = useState<Record<string, string>>({});
  const run = (fn: () => Promise<ActionResult>) => {
    setErrors({});
    startTransition(async () => {
      // A create redirects inside the action, so nothing comes back on success.
      const result = await fn();
      if (result?.ok === false) {
        setErrors(result.fieldErrors);
        toast.error(result.message);
      } else {
        toast.success(successMessage);
      }
    });
  };
  return { pending, errors, run };
}

type Common = { id: string; label: string; error?: string; description?: string };

export function TextField({
  id,
  label,
  error,
  description,
  value,
  onChange,
  maxLength,
  required,
  mono,
}: Common & { value: string; onChange: (v: string) => void; maxLength?: number; required?: boolean; mono?: boolean }) {
  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>
        {label}
        {required && <span className="text-muted-foreground"> (wajib)</span>}
      </FieldLabel>
      <Input
        id={id}
        value={value}
        maxLength={maxLength}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={!!error}
        className={mono ? "font-mono text-sm" : undefined}
      />
      {description && <FieldDescription>{description}</FieldDescription>}
      <FieldError>{error}</FieldError>
    </Field>
  );
}

export function TextAreaField({
  id,
  label,
  error,
  description,
  value,
  onChange,
  maxLength,
  rows = 4,
}: Common & { value: string; onChange: (v: string) => void; maxLength?: number; rows?: number }) {
  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Textarea id={id} rows={rows} value={value} maxLength={maxLength} onChange={(e) => onChange(e.target.value)} aria-invalid={!!error} />
      <FieldDescription>
        {description ? `${description} ` : ""}
        {maxLength ? `${value.length}/${maxLength}` : ""}
      </FieldDescription>
      <FieldError>{error}</FieldError>
    </Field>
  );
}

export function NumberField({ id, label, error, description, value, onChange }: Common & { value: number; onChange: (v: number) => void }) {
  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input id={id} type="number" value={Number.isFinite(value) ? value : ""} onChange={(e) => onChange(e.target.valueAsNumber)} className="w-32" />
      {description && <FieldDescription>{description}</FieldDescription>}
      <FieldError>{error}</FieldError>
    </Field>
  );
}

export function SwitchField({ id, label, checked, onChange }: { id: string; label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <Field orientation="horizontal">
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
      <FieldLabel htmlFor={id} className="font-normal">
        {label}
      </FieldLabel>
    </Field>
  );
}

/** Uploads to /api/admin/uploads and reports the storage key; "" removes the image. */
export function ImageField({
  id,
  label,
  imageKey,
  imageUrl,
  onChange,
}: {
  id: string;
  label: string;
  imageKey: string;
  imageUrl?: string;
  onChange: (key: string, url?: string) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);

  const pick = async (file: File | undefined) => {
    if (!file) return;
    if (!UPLOAD_TYPES.includes(file.type)) return toast.error("Format harus JPEG, PNG, atau WebP.");
    if (file.size > MAX_UPLOAD_BYTES) return toast.error("Ukuran gambar maksimal 4 MB.");
    setUploading(true);
    try {
      const body = new FormData();
      body.set("file", file);
      const res = await fetch("/api/admin/uploads", { method: "POST", body });
      const data = await res.json().catch(() => null);
      if (!res.ok) throw new Error(data?.detail ?? "Upload gagal.");
      onChange(data.key, data.url);
      toast.success("Gambar diunggah. Simpan untuk memakainya.");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Upload gagal.");
    } finally {
      setUploading(false);
      if (input.current) input.current.value = "";
    }
  };

  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <div className="flex items-start gap-4">
        {imageKey && imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={imageUrl} alt="" className="size-24 rounded-lg border object-cover" />
        ) : (
          <div className="grid size-24 place-items-center rounded-lg border border-dashed text-muted-foreground">
            <ImageUpIcon className="size-5" aria-hidden="true" />
          </div>
        )}
        <div className="flex flex-col gap-2">
          <input ref={input} id={id} type="file" accept={UPLOAD_TYPES.join(",")} className="sr-only" onChange={(e) => pick(e.target.files?.[0])} />
          <Button type="button" variant="outline" size="sm" onClick={() => input.current?.click()} disabled={uploading}>
            {uploading ? "Mengunggah…" : imageKey ? "Ganti gambar" : "Pilih gambar"}
          </Button>
          {imageKey && (
            <Button type="button" variant="ghost" size="sm" onClick={() => onChange("", undefined)}>
              <XIcon />
              Hapus gambar
            </Button>
          )}
          <FieldDescription>JPEG, PNG, atau WebP, maks 4 MB. Dikonversi ke WebP 1600px.</FieldDescription>
        </div>
      </div>
    </Field>
  );
}

export interface TraitRow {
  name: string;
  description: string;
}

/** Ordered list of traits/features as name + description pairs. */
export function TraitsField({ label, value, onChange, error }: { label: string; value: TraitRow[]; onChange: (v: TraitRow[]) => void; error?: string }) {
  const set = (i: number, patch: Partial<TraitRow>) => onChange(value.map((t, j) => (j === i ? { ...t, ...patch } : t)));
  return (
    <fieldset className="space-y-3">
      <legend className="text-sm font-medium">{label}</legend>
      {value.length === 0 && <p className="text-sm text-muted-foreground">Belum ada.</p>}
      {value.map((t, i) => (
        <div key={i} className="grid gap-2 rounded-lg border p-3 sm:grid-cols-[14rem_1fr_auto]">
          <Input aria-label={`Nama ${i + 1}`} placeholder="Nama" value={t.name} onChange={(e) => set(i, { name: e.target.value })} />
          <Textarea aria-label={`Deskripsi ${i + 1}`} placeholder="Deskripsi" rows={2} value={t.description} onChange={(e) => set(i, { description: e.target.value })} />
          <Button type="button" variant="ghost" size="icon" aria-label={`Hapus ${t.name || `baris ${i + 1}`}`} onClick={() => onChange(value.filter((_, j) => j !== i))}>
            <Trash2Icon />
          </Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={() => onChange([...value, { name: "", description: "" }])}>
        <PlusIcon />
        Tambah
      </Button>
      {error && <p className="text-sm text-destructive">{error}</p>}
    </fieldset>
  );
}

/** Drops empty rows so a half-added trait does not fail validation. */
export function cleanTraits(rows: TraitRow[]): TraitRow[] {
  return rows.map((r) => ({ name: r.name.trim(), description: r.description.trim() })).filter((r) => r.name || r.description);
}

export function ConfirmDelete({ title, description, onConfirm, disabled }: { title: string; description: string; onConfirm: () => void; disabled?: boolean }) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button type="button" variant="destructive" disabled={disabled}>
          <Trash2Icon />
          Hapus
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Batal</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={onConfirm}>
            Hapus
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/** Error for a field or, for list fields, the first error inside it ("traits.2.name"). */
export function errorFor(errors: Record<string, string>, field: string): string | undefined {
  if (errors[field]) return errors[field];
  const key = Object.keys(errors).find((k) => k.startsWith(`${field}.`));
  return key ? `${key.slice(field.length + 1)}: ${errors[key]}` : undefined;
}
