"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState, useTransition } from "react";
import { useForm } from "react-hook-form";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

import { login } from "./actions";
import { applyFailure } from "./form-helpers";
import { type LoginInput, loginSchema } from "./schemas";

const googleErrors: Record<string, string> = {
  google_denied: "Login Google dibatalkan.",
  google_state: "Sesi login Google kedaluwarsa. Coba lagi.",
  google_unverified: "Email Google itu belum terverifikasi.",
  google_unavailable: "Login Google belum tersedia. Pakai email dan password dulu.",
  google_blocked: "Akun ini diblokir.",
  google_failed: "Login Google gagal. Coba lagi.",
};

export function LoginForm() {
  const params = useSearchParams();
  const next = params.get("next") ?? undefined;
  const [message, setMessage] = useState<string | null>(() => googleErrors[params.get("error") ?? ""] ?? null);
  const [pending, startTransition] = useTransition();

  const form = useForm<LoginInput>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });
  const { errors } = form.formState;

  const onSubmit = form.handleSubmit((values) => {
    setMessage(null);
    startTransition(async () => {
      const result = await login(values, next);
      // On success the action redirects, so only failures come back.
      if (!result.ok) setMessage(applyFailure(result, ["email", "password"], form.setError) || null);
    });
  });

  const registerHref = next ? `/register?next=${encodeURIComponent(next)}` : "/register";

  return (
    <form onSubmit={onSubmit} noValidate>
      <FieldGroup>
        {message && (
          <Alert variant="destructive">
            <AlertDescription>{message}</AlertDescription>
          </Alert>
        )}
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
            autoComplete="current-password"
            aria-invalid={!!errors.password}
            {...form.register("password")}
          />
          <FieldError errors={[errors.password]} />
        </Field>
        <Button type="submit" size="lg" className="h-10 text-base" disabled={pending}>
          {pending ? "Sedang masuk…" : "Masuk"}
        </Button>
        <p className="text-sm text-muted-foreground">
          Belum punya akun?{" "}
          <Link href={registerHref} className="text-foreground underline underline-offset-2">
            Daftar
          </Link>
        </p>
      </FieldGroup>
    </form>
  );
}
