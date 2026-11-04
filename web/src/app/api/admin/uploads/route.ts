import { cookies, headers } from "next/headers";

import { MAX_UPLOAD_BYTES, UPLOAD_TYPES } from "@/features/admin/content/upload-limits";
import { jsonReply } from "@/features/ai/route-helpers";
import { clientIpFrom } from "@/lib/api/request";
import { ACCESS_COOKIE } from "@/lib/auth/tokens";
import { serverEnv } from "@/lib/env";

const allowed = new Set(UPLOAD_TYPES);

/**
 * Forwards an admin image upload to POST /v1/admin/uploads. Multipart is
 * passed through as-is; the API converts it to WebP and checks the role.
 */
export async function POST(request: Request) {
  const form = await request.formData().catch(() => null);
  const file = form?.get("file");
  if (!(file instanceof File)) return jsonReply({ status: 422, detail: "pilih file gambar" }, 422);
  if (!allowed.has(file.type)) return jsonReply({ status: 422, detail: "format harus JPEG, PNG, atau WebP" }, 422);
  if (file.size > MAX_UPLOAD_BYTES) return jsonReply({ status: 413, detail: "ukuran gambar maksimal 4 MB" }, 413);

  const [jar, h] = await Promise.all([cookies(), headers()]);
  const token = jar.get(ACCESS_COOKIE)?.value;
  if (!token) return jsonReply({ status: 401, detail: "butuh login" }, 401);

  const body = new FormData();
  body.set("file", file, file.name);
  const outgoing: Record<string, string> = {
    "X-Internal-Key": serverEnv("INTERNAL_API_KEY"),
    Authorization: `Bearer ${token}`,
  };
  const ip = clientIpFrom(h);
  if (ip) outgoing["X-Client-IP"] = ip;

  // openapi-fetch has no multipart helper worth the indirection here; this is
  // the one bare fetch, and it sets the same headers as api().
  const res = await fetch(`${serverEnv("API_BASE_URL")}/v1/admin/uploads`, { method: "POST", headers: outgoing, body });
  return jsonReply(await res.json().catch(() => null), res.status);
}
