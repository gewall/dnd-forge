"use client";

import { useState, useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

import { optionalInt } from "../form-utils";
import { updateAccount, updateQuotaDefaults } from "./actions";

interface AccountState {
  role: "USER" | "ADMIN";
  isBanned: boolean;
  portraitQuota: number | null;
  loreQuota: number | null;
}

export function AccountForm({ id, initial, defaults }: { id: string; initial: AccountState; defaults: { portrait: number; lore: number } | null }) {
  const [role, setRole] = useState(initial.role);
  const [banned, setBanned] = useState(initial.isBanned);
  const [portrait, setPortrait] = useState(initial.portraitQuota?.toString() ?? "");
  const [lore, setLore] = useState(initial.loreQuota?.toString() ?? "");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [pending, startTransition] = useTransition();

  const save = () => {
    setErrors({});
    startTransition(async () => {
      const result = await updateAccount(id, {
        role,
        isBanned: banned,
        portraitQuota: optionalInt(portrait),
        loreQuota: optionalInt(lore),
      });
      if (result.ok) {
        toast.success("Perubahan akun disimpan.");
        return;
      }
      setErrors(result.fieldErrors);
      toast.error(result.message);
    });
  };

  return (
    <FieldGroup className="max-w-md">
      <Field>
        <FieldLabel htmlFor="role">Peran</FieldLabel>
        <Select value={role} onValueChange={(v) => setRole(v as AccountState["role"])}>
          <SelectTrigger id="role" className="w-48">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="USER">Pemain</SelectItem>
            <SelectItem value="ADMIN">Admin</SelectItem>
          </SelectContent>
        </Select>
      </Field>
      <Field orientation="horizontal">
        <Switch id="banned" checked={banned} onCheckedChange={setBanned} />
        <FieldLabel htmlFor="banned" className="font-normal">
          Blokir akun (langsung keluar dari semua sesi)
        </FieldLabel>
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field data-invalid={!!errors.portraitQuota}>
          <FieldLabel htmlFor="pq">Kuota portrait/hari</FieldLabel>
          <Input id="pq" inputMode="numeric" value={portrait} onChange={(e) => setPortrait(e.target.value)} placeholder={defaults ? `Default (${defaults.portrait})` : "Default"} />
          <FieldError>{errors.portraitQuota}</FieldError>
        </Field>
        <Field data-invalid={!!errors.loreQuota}>
          <FieldLabel htmlFor="lq">Kuota lore/hari</FieldLabel>
          <Input id="lq" inputMode="numeric" value={lore} onChange={(e) => setLore(e.target.value)} placeholder={defaults ? `Default (${defaults.lore})` : "Default"} />
          <FieldError>{errors.loreQuota}</FieldError>
        </Field>
      </div>
      <FieldDescription>Kosongkan kuota untuk memakai nilai default.</FieldDescription>
      <div>
        <Button onClick={save} disabled={pending}>
          {pending ? "Menyimpan…" : "Simpan perubahan"}
        </Button>
      </div>
    </FieldGroup>
  );
}

export function QuotaDefaultsForm({ initial }: { initial: { portrait: number; lore: number } }) {
  const [portrait, setPortrait] = useState(String(initial.portrait));
  const [lore, setLore] = useState(String(initial.lore));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [pending, startTransition] = useTransition();

  const save = () => {
    setErrors({});
    startTransition(async () => {
      const result = await updateQuotaDefaults({ portrait: Number(portrait), lore: Number(lore) });
      if (result.ok) toast.success("Kuota default disimpan.");
      else {
        setErrors(result.fieldErrors);
        toast.error(result.message);
      }
    });
  };

  return (
    <FieldGroup className="max-w-md">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field data-invalid={!!errors.portrait}>
          <FieldLabel htmlFor="dp">Portrait per hari</FieldLabel>
          <Input id="dp" inputMode="numeric" value={portrait} onChange={(e) => setPortrait(e.target.value)} />
          <FieldError>{errors.portrait}</FieldError>
        </Field>
        <Field data-invalid={!!errors.lore}>
          <FieldLabel htmlFor="dl">Lore per hari</FieldLabel>
          <Input id="dl" inputMode="numeric" value={lore} onChange={(e) => setLore(e.target.value)} />
          <FieldError>{errors.lore}</FieldError>
        </Field>
      </div>
      <FieldDescription>Berlaku untuk user yang tidak punya kuota sendiri. Reset tiap tengah malam WIB.</FieldDescription>
      <div>
        <Button onClick={save} disabled={pending}>
          {pending ? "Menyimpan…" : "Simpan kuota default"}
        </Button>
      </div>
    </FieldGroup>
  );
}
