"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState, useTransition } from "react";
import { useForm } from "react-hook-form";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

import { register as registerAccount } from "./actions";
import { applyFailure } from "./form-helpers";
import { type RegisterInput, registerSchema } from "./schemas";

export function RegisterForm() {
  const params = useSearchParams();
  const next = params.get("next") ?? undefined;
  const [message, setMessage] = useState<string | null>(null);
  const [pending, startTransition] = useTransition();

  const form = useForm<RegisterInput>({
    resolver: zodResolver(registerSchema),
    defaultValues: { name: "", email: "", password: "" },
  });
  const { errors } = form.formState;

  const onSubmit = form.handleSubmit((values) => {
    setMessage(null);
    startTransition(async () => {
      const result = await registerAccount(values, next);
      if (!result.ok) setMessage(applyFailure(result, ["name", "email", "password"], form.setError) || null);
    });
  });

  const loginHref = next ? `/login?next=${encodeURIComponent(next)}` : "/login";

  return (
    <form onSubmit={onSubmit} noValidate>
      <FieldGroup>
        {message && (
          <Alert variant="destructive">
            <AlertDescription>{message}</AlertDescription>
          </Alert>
        )}
        <Field data-invalid={!!errors.name}>
          <FieldLabel htmlFor="name">Nama</FieldLabel>
          <Input id="name" autoComplete="nickname" aria-invalid={!!errors.name} {...form.register("name")} />
          <FieldDescription>Boleh dikosongkan. Tampil di menu akun.</FieldDescription>
          <FieldError errors={[errors.name]} />
        </Field>
        <Field data-invalid={!!errors.email}>
          <FieldLabel htmlFor="email">Email</FieldLabel>
          <Input id="email" type="email" autoComplete="email" aria-invalid={!!errors.email} {...form.register("email")} />
          <FieldError errors={[errors.email]} />
        </Field>
        <Field data-invalid={!!errors.password}>
          <FieldLabel htmlFor="password">Password</FieldLabel>
          <Input
            id="password"
            type="password"
            autoComplete="new-password"
            aria-invalid={!!errors.password}
            {...form.register("password")}
          />
          <FieldDescription>Minimal 8 karakter.</FieldDescription>
          <FieldError errors={[errors.password]} />
        </Field>
        <Button type="submit" size="lg" className="h-10 text-base" disabled={pending}>
          {pending ? "Membuat akun…" : "Buat akun"}
        </Button>
        <p className="text-sm text-muted-foreground">
          Sudah punya akun?{" "}
          <Link href={loginHref} className="text-foreground underline underline-offset-2">
            Masuk
          </Link>
        </p>
      </FieldGroup>
    </form>
  );
}
