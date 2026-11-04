"use client";

import { CheckCircle2Icon, CopyIcon, TriangleAlertIcon } from "lucide-react";
import { useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import type { components } from "@/lib/api/schema";

import { ConfirmDelete, TextAreaField, TextField, useAdminAction } from "../content/fields";
import { formatDate } from "../page-header";
import { activateTemplate, createTemplate, deleteTemplate } from "./actions";
import { insertPlaceholder, unknownPlaceholders, usedPlaceholders } from "./placeholders";

type Template = components["schemas"]["TemplateDTO"];
export type TemplateKind = components["schemas"]["TemplateRequest"]["kind"];

const kinds: { kind: TemplateKind; title: string; use: string }[] = [
  { kind: "portrait", title: "Portrait", use: "Prompt gambar untuk AI Portrait." },
  { kind: "lore", title: "Lore", use: "Prompt teks untuk Lore karakter." },
];

interface Draft {
  name: string;
  template: string;
  negativePrompt: string;
}

function draftFrom(t?: Template): Draft {
  return { name: t?.name ?? "", template: t?.template ?? "", negativePrompt: t?.negativePrompt ?? "" };
}

function TemplateForm({ kind, placeholders, draft, setDraft }: { kind: TemplateKind; placeholders: string[]; draft: Draft; setDraft: (d: Draft) => void }) {
  const area = useRef<HTMLTextAreaElement>(null);
  const { pending, errors, run } = useAdminAction("Versi baru disimpan. Aktifkan bila ingin langsung dipakai.");
  const unknown = unknownPlaceholders(draft.template, placeholders);
  const used = usedPlaceholders(draft.template);
  const id = `tpl-${kind}`;

  const insert = (name: string) => {
    const el = area.current;
    const start = el?.selectionStart ?? draft.template.length;
    const end = el?.selectionEnd ?? start;
    const next = insertPlaceholder(draft.template, start, end, name);
    setDraft({ ...draft, template: next.text });
    requestAnimationFrame(() => {
      el?.focus();
      el?.setSelectionRange(next.caret, next.caret);
    });
  };

  const submit = () =>
    run(() =>
      createTemplate({
        kind,
        name: draft.name.trim(),
        template: draft.template,
        negativePrompt: draft.negativePrompt.trim(),
      }),
    );

  return (
    <FieldGroup>
      <TextField id={`${id}-name`} label="Nama versi" required maxLength={60} value={draft.name} error={errors.name} onChange={(name) => setDraft({ ...draft, name })} description="Contoh: Portrait standar, lebih sinematik." />
      <Field data-invalid={!!errors.template || unknown.length > 0}>
        <FieldLabel htmlFor={`${id}-body`}>
          Template <span className="text-muted-foreground">(wajib)</span>
        </FieldLabel>
        <div className="flex flex-wrap gap-1.5" role="group" aria-label="Sisipkan placeholder">
          {placeholders.map((p) => (
            <Button key={p} type="button" size="xs" variant={used.has(p) ? "secondary" : "outline"} className="font-mono" onClick={() => insert(p)}>
              {`{${p}}`}
            </Button>
          ))}
        </div>
        <Textarea
          ref={area}
          id={`${id}-body`}
          rows={12}
          maxLength={4000}
          value={draft.template}
          onChange={(e) => setDraft({ ...draft, template: e.target.value })}
          aria-invalid={!!errors.template || unknown.length > 0}
          className="font-mono text-sm"
        />
        <FieldDescription>
          Klik placeholder untuk menyisipkannya di posisi kursor. Baris yang semua placeholder-nya kosong dibuang otomatis, jadi taruh bagian opsional di baris sendiri. {draft.template.length}/4000
        </FieldDescription>
        {unknown.length > 0 && <FieldError>Placeholder tidak dikenal untuk {kind}: {unknown.join(", ")}</FieldError>}
        <FieldError>{errors.template}</FieldError>
      </Field>
      <TextAreaField
        id={`${id}-neg`}
        label="Hal yang dihindari"
        rows={2}
        maxLength={500}
        value={draft.negativePrompt}
        error={errors.negativePrompt}
        onChange={(negativePrompt) => setDraft({ ...draft, negativePrompt })}
        description="Opsional. Ditambahkan ke prompt sebagai “Avoid: …”."
      />
      <div>
        <Button onClick={submit} disabled={pending || unknown.length > 0 || !draft.template.trim()}>
          {pending ? "Menyimpan…" : "Simpan sebagai versi baru"}
        </Button>
      </div>
    </FieldGroup>
  );
}

function VersionItem({ template, onCopy }: { template: Template; onCopy: () => void }) {
  const activate = useAdminAction(`Versi ${template.version} sekarang aktif.`);
  const del = useAdminAction("Versi dihapus.");
  return (
    <li className="rounded-xl border data-[active=true]:border-primary/60" data-active={template.isActive}>
      <details open={template.isActive} className="group">
        <summary className="flex cursor-pointer list-none flex-wrap items-center gap-x-3 gap-y-1 p-4 [&::-webkit-details-marker]:hidden">
          <span className="font-mono text-sm text-muted-foreground tabular-nums">v{template.version}</span>
          <span className="font-medium">{template.name}</span>
          {template.isActive && (
            <Badge>
              <CheckCircle2Icon aria-hidden="true" />
              Aktif
            </Badge>
          )}
          <span className="ml-auto text-sm text-muted-foreground">{formatDate(template.createdAt)}</span>
        </summary>
        <div className="space-y-4 border-t p-4">
          <pre className="max-h-80 overflow-auto rounded-lg bg-muted p-3 font-mono text-sm whitespace-pre-wrap">{template.template}</pre>
          {template.negativePrompt && (
            <p className="text-sm">
              <span className="text-muted-foreground">Dihindari: </span>
              {template.negativePrompt}
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            {!template.isActive && (
              <Button size="sm" onClick={() => activate.run(() => activateTemplate(template.id))} disabled={activate.pending}>
                {activate.pending ? "Mengaktifkan…" : "Aktifkan"}
              </Button>
            )}
            <Button size="sm" variant="outline" onClick={onCopy}>
              <CopyIcon />
              Pakai sebagai dasar
            </Button>
            {!template.isActive && (
              <ConfirmDelete
                title={`Hapus v${template.version} ${template.name}?`}
                description="Portrait dan lore lama tetap menyimpan prompt finalnya; hanya versi template ini yang hilang."
                disabled={del.pending}
                onConfirm={() => del.run(() => deleteTemplate(template.id))}
              />
            )}
          </div>
        </div>
      </details>
    </li>
  );
}

function KindPanel({ kind, use, templates, placeholders }: { kind: TemplateKind; use: string; templates: Template[]; placeholders: string[] }) {
  const versions = templates.filter((t) => t.kind === kind).sort((a, b) => b.version - a.version);
  const active = versions.find((t) => t.isActive);
  const [draft, setDraft] = useState(() => draftFrom(active));
  const form = useRef<HTMLElement>(null);

  return (
    <div className="grid gap-10 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <section aria-labelledby={`${kind}-versions`}>
        <h2 id={`${kind}-versions`} className="text-xl font-semibold">
          Versi
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">{use} Versi aktif dipakai untuk permintaan baru.</p>
        {!active && (
          <Alert className="mt-4">
            <TriangleAlertIcon />
            <AlertTitle>Belum ada versi aktif</AlertTitle>
            <AlertDescription>User tidak bisa membuat {kind} sampai satu versi diaktifkan.</AlertDescription>
          </Alert>
        )}
        {versions.length === 0 ? (
          <p className="mt-4 text-muted-foreground">Belum ada versi.</p>
        ) : (
          <ul className="mt-4 space-y-3">
            {versions.map((t) => (
              <VersionItem
                key={t.id}
                template={t}
                onCopy={() => {
                  setDraft(draftFrom(t));
                  form.current?.scrollIntoView({ behavior: "smooth", block: "start" });
                }}
              />
            ))}
          </ul>
        )}
      </section>
      <section ref={form} aria-labelledby={`${kind}-new`} className="scroll-mt-20">
        <h2 id={`${kind}-new`} className="mb-1 text-xl font-semibold">
          Versi baru
        </h2>
        <p className="mb-4 text-sm text-muted-foreground">Versi lama tidak diubah; simpanan ini menjadi versi berikutnya.</p>
        <TemplateForm kind={kind} placeholders={placeholders} draft={draft} setDraft={setDraft} />
      </section>
    </div>
  );
}

export function TemplateManager({ templates, placeholders }: { templates: Template[]; placeholders: Record<TemplateKind, string[]> }) {
  return (
    <Tabs defaultValue="portrait">
      <TabsList>
        {kinds.map((k) => (
          <TabsTrigger key={k.kind} value={k.kind}>
            {k.title}
          </TabsTrigger>
        ))}
      </TabsList>
      {kinds.map((k) => (
        <TabsContent key={k.kind} value={k.kind} className="pt-6">
          <KindPanel kind={k.kind} use={k.use} templates={templates} placeholders={placeholders[k.kind]} />
        </TabsContent>
      ))}
    </Tabs>
  );
}
