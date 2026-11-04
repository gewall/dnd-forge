"use client";

import { CheckCircle2Icon, ListIcon, PlugZapIcon } from "lucide-react";
import { useState, useTransition } from "react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { components } from "@/lib/api/schema";

import { ConfirmDelete, TextField, useAdminAction } from "../content/fields";
import { formatDate } from "../page-header";
import { activateProvider, createProvider, deleteProvider, providerModels, testProvider, updateProvider } from "./actions";

type Provider = components["schemas"]["ProviderDTO"];
type ProviderType = components["schemas"]["TypeDTO"];
type Capability = components["schemas"]["ProviderRequest"]["capability"];

export const capabilityLabels: Record<string, { title: string; use: string }> = {
  image: { title: "Gambar", use: "Dipakai untuk AI Portrait." },
  text: { title: "Teks", use: "Dipakai untuk Lore karakter." },
};

const typeLabels: Record<string, string> = {
  openai: "OpenAI",
  deepseek: "DeepSeek",
  google: "Google",
  stability: "Stability AI",
  replicate: "Replicate",
  fake: "Fake (uji)",
};

const typeName = (t: string) => typeLabels[t] ?? t;

/** Password field for an API key; the value never comes back from the API. */
function KeyField({ id, value, onChange, error, keyHint }: { id: string; value: string; onChange: (v: string) => void; error?: string; keyHint?: string }) {
  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>
        API key{!keyHint && <span className="text-muted-foreground"> (wajib)</span>}
      </FieldLabel>
      <Input id={id} type="password" autoComplete="off" spellCheck={false} value={value} maxLength={300} onChange={(e) => onChange(e.target.value)} aria-invalid={!!error} className="font-mono text-sm" />
      <FieldDescription>
        {keyHint ? `Tersimpan: ${keyHint}. Kosongkan untuk tetap memakai key ini.` : "Disimpan terenkripsi; setelah disimpan hanya 4 karakter terakhir yang terlihat."}
      </FieldDescription>
      <FieldError>{error}</FieldError>
    </Field>
  );
}

/** Model name, with an optional list fetched with the stored key. */
function ModelField({ id, providerId, value, onChange, error }: { id: string; providerId?: string; value: string; onChange: (v: string) => void; error?: string }) {
  const [models, setModels] = useState<string[] | null>(null);
  const [loading, startLoading] = useTransition();
  const load = () =>
    startLoading(async () => {
      if (!providerId) return;
      const r = await providerModels(providerId);
      if (!r.ok) return void toast.error(r.message);
      setModels(r.data);
      if (r.data.length === 0) toast.info("Provider tidak mengembalikan daftar model.");
    });

  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>
        Model default <span className="text-muted-foreground">(wajib)</span>
      </FieldLabel>
      <div className="flex gap-2">
        <Input id={id} value={value} maxLength={80} onChange={(e) => onChange(e.target.value)} aria-invalid={!!error} className="font-mono text-sm" />
        {providerId && (
          <Button type="button" variant="outline" onClick={load} disabled={loading}>
            <ListIcon />
            {loading ? "Memuat…" : "Daftar model"}
          </Button>
        )}
      </div>
      {models && models.length > 0 && (
        <Select value={models.includes(value) ? value : ""} onValueChange={onChange}>
          <SelectTrigger aria-label="Pilih dari daftar model" className="w-full">
            <SelectValue placeholder={`Pilih dari ${models.length} model`} />
          </SelectTrigger>
          <SelectContent>
            {models.map((m) => (
              <SelectItem key={m} value={m} className="font-mono text-sm">
                {m}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      <FieldDescription>
        {providerId ? "Ketik nama model, atau ambil daftarnya memakai key tersimpan." : "Daftar model bisa diambil setelah provider disimpan."}
      </FieldDescription>
      <FieldError>{error}</FieldError>
    </Field>
  );
}

function EditProvider({ provider, onDone }: { provider: Provider; onDone: () => void }) {
  const [label, setLabel] = useState(provider.label);
  const [model, setModel] = useState(provider.defaultModel);
  const [apiKey, setApiKey] = useState("");
  const { pending, errors, run } = useAdminAction("Provider disimpan.");
  const submit = () =>
    run(async () => {
      const r = await updateProvider(provider.id, { label: label.trim(), defaultModel: model.trim(), apiKey });
      if (r.ok) onDone();
      return r;
    });

  return (
    <FieldGroup className="max-w-2xl">
      <TextField id={`pl-${provider.id}`} label="Nama" required maxLength={60} value={label} error={errors.label} onChange={setLabel} />
      <ModelField id={`pm-${provider.id}`} providerId={provider.id} value={model} error={errors.defaultModel} onChange={setModel} />
      <KeyField id={`pk-${provider.id}`} keyHint={provider.keyHint} value={apiKey} error={errors.apiKey} onChange={setApiKey} />
      <div className="flex gap-2">
        <Button onClick={submit} disabled={pending}>
          {pending ? "Menyimpan…" : "Simpan"}
        </Button>
        <Button variant="ghost" onClick={onDone}>
          Batal
        </Button>
      </div>
    </FieldGroup>
  );
}

function ProviderRow({ provider }: { provider: Provider }) {
  const [editing, setEditing] = useState(false);
  const [testing, startTest] = useTransition();
  const activate = useAdminAction(`${provider.label} sekarang aktif.`);
  const del = useAdminAction("Provider dihapus.");

  const test = () =>
    startTest(async () => {
      const r = await testProvider(provider.id);
      if (r.ok) toast.success(`Koneksi ${provider.label} berhasil.`);
      else toast.error(r.message);
    });

  return (
    <li className="rounded-xl border p-4 data-[active=true]:border-primary/60" data-active={provider.isActive}>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="flex flex-wrap items-center gap-2 font-medium">
            {provider.label}
            {provider.isActive && (
              <Badge>
                <CheckCircle2Icon aria-hidden="true" />
                Aktif
              </Badge>
            )}
          </p>
          <dl className="mt-1 grid grid-cols-[auto_1fr] gap-x-3 text-sm text-muted-foreground">
            <dt>Jenis</dt>
            <dd>{typeName(provider.type)}</dd>
            <dt>Model</dt>
            <dd className="font-mono text-xs leading-5 break-all">{provider.defaultModel}</dd>
            <dt>Key</dt>
            <dd className="font-mono text-xs leading-5">{provider.keyHint || "-"}</dd>
            <dt>Diubah</dt>
            <dd>{formatDate(provider.updatedAt)}</dd>
          </dl>
        </div>
        {!editing && (
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" onClick={test} disabled={testing}>
              <PlugZapIcon />
              {testing ? "Menguji…" : "Uji koneksi"}
            </Button>
            {!provider.isActive && (
              <Button size="sm" onClick={() => activate.run(() => activateProvider(provider.id))} disabled={activate.pending}>
                {activate.pending ? "Mengaktifkan…" : "Aktifkan"}
              </Button>
            )}
            <Button size="sm" variant="outline" onClick={() => setEditing(true)}>
              Ubah
            </Button>
            {!provider.isActive && (
              <ConfirmDelete
                title={`Hapus ${provider.label}?`}
                description="Key ikut terhapus. Provider yang sudah pernah dipakai tidak bisa dihapus; biarkan saja nonaktif."
                disabled={del.pending}
                onConfirm={() => del.run(() => deleteProvider(provider.id))}
              />
            )}
          </div>
        )}
      </div>
      {editing && (
        <div className="mt-4 border-t pt-4">
          <EditProvider provider={provider} onDone={() => setEditing(false)} />
        </div>
      )}
    </li>
  );
}

function NewProvider({ types }: { types: ProviderType[] }) {
  const [type, setType] = useState(types[0]?.type ?? "");
  const capsOf = (t: string) => (types.find((x) => x.type === t)?.capabilities ?? []) as Capability[];
  const [capability, setCapability] = useState<Capability | "">(capsOf(type)[0] ?? "");
  const [label, setLabel] = useState("");
  const [model, setModel] = useState("");
  const [apiKey, setApiKey] = useState("");
  const { pending, errors, run } = useAdminAction("Provider ditambahkan. Aktifkan bila ingin langsung dipakai.");

  const pickType = (t: string) => {
    setType(t);
    const caps = capsOf(t);
    if (!caps.includes(capability as Capability)) setCapability(caps[0] ?? "");
  };

  const submit = () =>
    run(async () => {
      if (!type || !capability) return { ok: false, status: 422, message: "Pilih jenis dan kemampuan provider.", fieldErrors: {} };
      const r = await createProvider({
        type: type as components["schemas"]["ProviderRequest"]["type"],
        capability,
        label: label.trim(),
        defaultModel: model.trim(),
        apiKey: apiKey.trim(),
      });
      if (r.ok) {
        setLabel("");
        setModel("");
        setApiKey("");
      }
      return r;
    });

  if (types.length === 0) return <p className="text-muted-foreground">Build API ini tidak mendukung provider apa pun.</p>;

  return (
    <FieldGroup className="max-w-2xl">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field data-invalid={!!errors.type}>
          <FieldLabel htmlFor="np-type">Jenis</FieldLabel>
          <Select value={type} onValueChange={pickType}>
            <SelectTrigger id="np-type" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {types.map((t) => (
                <SelectItem key={t.type} value={t.type}>
                  {typeName(t.type)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldError>{errors.type}</FieldError>
        </Field>
        <Field data-invalid={!!errors.capability}>
          <FieldLabel htmlFor="np-cap">Kemampuan</FieldLabel>
          <Select value={capability} onValueChange={(v) => setCapability(v as Capability)}>
            <SelectTrigger id="np-cap" className="w-full">
              <SelectValue placeholder="Pilih kemampuan" />
            </SelectTrigger>
            <SelectContent>
              {capsOf(type).map((c) => (
                <SelectItem key={c} value={c}>
                  {capabilityLabels[c]?.title ?? c}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldDescription>{capability ? capabilityLabels[capability]?.use : null}</FieldDescription>
          <FieldError>{errors.capability}</FieldError>
        </Field>
      </div>
      <TextField id="np-label" label="Nama" required maxLength={60} value={label} error={errors.label} onChange={setLabel} description="Contoh: OpenAI utama." />
      <ModelField id="np-model" value={model} error={errors.defaultModel} onChange={setModel} />
      <KeyField id="np-key" value={apiKey} error={errors.apiKey} onChange={setApiKey} />
      <div>
        <Button onClick={submit} disabled={pending}>
          {pending ? "Menyimpan…" : "Tambah provider"}
        </Button>
      </div>
    </FieldGroup>
  );
}

export function ProviderManager({ providers, types }: { providers: Provider[]; types: ProviderType[] }) {
  const caps = ["image", "text"];
  return (
    <div className="space-y-12">
      {caps.map((cap) => {
        const list = providers.filter((p) => p.capability === cap);
        const active = list.find((p) => p.isActive);
        return (
          <section key={cap} aria-labelledby={`cap-${cap}`}>
            <h2 id={`cap-${cap}`} className="text-xl font-semibold">
              {capabilityLabels[cap].title}
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {capabilityLabels[cap].use}{" "}
              {active ? `Permintaan baru memakai ${active.label}.` : "Belum ada provider aktif, jadi fitur ini nonaktif untuk user."}
            </p>
            {list.length === 0 ? (
              <p className="mt-4 text-muted-foreground">Belum ada provider.</p>
            ) : (
              <ul className="mt-4 space-y-3">
                {list.map((p) => (
                  <ProviderRow key={p.id} provider={p} />
                ))}
              </ul>
            )}
          </section>
        );
      })}
      <section aria-labelledby="new-provider" className="border-t pt-8">
        <h2 id="new-provider" className="mb-4 text-xl font-semibold">
          Provider baru
        </h2>
        <NewProvider types={types} />
      </section>
    </div>
  );
}
