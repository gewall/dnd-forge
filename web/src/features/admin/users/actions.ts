"use server";

import { refresh } from "next/cache";
import { z } from "zod";

import { type ActionResult, toResult } from "@/lib/api/errors";
import { requestApi } from "@/lib/auth/session";

const quota = z.number().int().min(0, "Minimal 0.").max(1000, "Maksimal 1000.").nullable();

const accountInput = z.object({
  role: z.enum(["USER", "ADMIN"]),
  isBanned: z.boolean(),
  portraitQuota: quota,
  loreQuota: quota,
});

const defaultsInput = z.object({
  portrait: z.number().int().min(0, "Minimal 0.").max(1000, "Maksimal 1000."),
  lore: z.number().int().min(0, "Minimal 0.").max(1000, "Maksimal 1000."),
});

function invalid(error: z.ZodError): Extract<ActionResult, { ok: false }> {
  const fieldErrors: Record<string, string> = {};
  for (const issue of error.issues) fieldErrors[issue.path.join(".")] ??= issue.message;
  return { ok: false, status: 422, message: "Periksa lagi isian yang ditandai.", fieldErrors };
}

/** Full account state: role, block, and per-user quotas (null = default). Blocking ends all sessions. */
export async function updateAccount(id: string, input: z.input<typeof accountInput>): Promise<ActionResult> {
  if (!z.uuid().safeParse(id).success) return { ok: false, status: 404, message: "User tidak ditemukan.", fieldErrors: {} };
  const parsed = accountInput.safeParse(input);
  if (!parsed.success) return invalid(parsed.error);
  const result = toResult(
    await (await requestApi()).PUT("/v1/admin/users/{id}", { params: { path: { id } }, body: parsed.data }),
  );
  if (!result.ok) return result;
  refresh();
  return { ok: true, data: undefined };
}

export async function updateQuotaDefaults(input: z.input<typeof defaultsInput>): Promise<ActionResult> {
  const parsed = defaultsInput.safeParse(input);
  if (!parsed.success) return invalid(parsed.error);
  const result = toResult(await (await requestApi()).PUT("/v1/admin/settings/quota", { body: parsed.data }));
  if (!result.ok) return result;
  refresh();
  return { ok: true, data: undefined };
}
