"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { z } from "zod";

import { type ActionResult, toResult } from "@/lib/api/errors";
import { requestApi } from "@/lib/auth/session";
import { clearSessionCookies, REFRESH_COOKIE, safeNext, setSessionCookies } from "@/lib/auth/tokens";

import { type LoginInput, loginSchema, type RegisterInput, registerSchema } from "./schemas";

function invalid(error: z.ZodError): Extract<ActionResult, { ok: false }> {
  const fieldErrors: Record<string, string> = {};
  for (const issue of error.issues) {
    const field = issue.path.join(".");
    fieldErrors[field] ??= issue.message;
  }
  return { ok: false, status: 422, message: "Periksa lagi isian yang ditandai.", fieldErrors };
}

export async function login(input: LoginInput, next?: string): Promise<ActionResult> {
  const parsed = loginSchema.safeParse(input);
  if (!parsed.success) return invalid(parsed.error);

  const result = toResult(await (await requestApi()).POST("/v1/auth/login", { body: parsed.data }));
  if (!result.ok) return result;

  setSessionCookies(await cookies(), result.data);
  redirect(safeNext(next, "/characters"));
}

export async function register(input: RegisterInput, next?: string): Promise<ActionResult> {
  const parsed = registerSchema.safeParse(input);
  if (!parsed.success) return invalid(parsed.error);

  const { name, ...rest } = parsed.data;
  const body = name ? { ...rest, name } : rest;
  const result = toResult(await (await requestApi()).POST("/v1/auth/register", { body }));
  if (!result.ok) return result;

  setSessionCookies(await cookies(), result.data);
  redirect(safeNext(next, "/builder"));
}

export async function logout(): Promise<void> {
  const jar = await cookies();
  const refreshToken = jar.get(REFRESH_COOKIE)?.value;
  if (refreshToken) {
    // Revoking is best effort: the cookies go away either way.
    try {
      await (await requestApi()).POST("/v1/auth/logout", { body: { refreshToken } });
    } catch {}
  }
  clearSessionCookies(jar);
  redirect("/");
}
