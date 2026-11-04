// Session helpers for proxy.ts. No "server-only" import: proxy does not run in
// the React server environment, so env is read straight from process.env.
import { importSPKI, jwtVerify } from "jose";

import { type ApiClient, createApi } from "@/lib/api/base";

import type { SessionDTO } from "./tokens";

const ISSUER = "dnd-forge-api";
const AUDIENCE = "dnd-forge-web";

export interface AccessClaims {
  userId: string;
  role: "USER" | "ADMIN";
}

function proxyApi(clientIp?: string): ApiClient | null {
  const baseUrl = process.env.API_BASE_URL;
  const internalKey = process.env.INTERNAL_API_KEY;
  if (!baseUrl || !internalKey) return null;
  return createApi({ baseUrl, internalKey, clientIp });
}

let publicKey: Promise<CryptoKey> | undefined;

/**
 * Ed25519 key that signs access tokens: JWT_PUBLIC_KEY_PEM when set, else
 * fetched once from GET /v1/auth/public-key. A failed fetch is not cached.
 */
function accessTokenKey(): Promise<CryptoKey> {
  publicKey ??= (async () => {
    let pem = process.env.JWT_PUBLIC_KEY_PEM?.replace(/\\n/g, "\n");
    if (!pem) {
      const client = proxyApi();
      if (!client) throw new Error("API_BASE_URL/INTERNAL_API_KEY belum diisi");
      const { data } = await client.GET("/v1/auth/public-key");
      if (!data?.publicKeyPem) throw new Error("public key tidak tersedia");
      pem = data.publicKeyPem;
    }
    return importSPKI(pem, "EdDSA");
  })().catch((err: unknown) => {
    publicKey = undefined;
    throw err;
  });
  return publicKey;
}

/** Verified claims of an access token, or null if it is invalid or expired. */
export async function verifyAccessToken(token: string): Promise<AccessClaims | null> {
  try {
    const { payload } = await jwtVerify(token, await accessTokenKey(), {
      issuer: ISSUER,
      audience: AUDIENCE,
      algorithms: ["EdDSA"],
    });
    if (typeof payload.sub !== "string") return null;
    return { userId: payload.sub, role: payload.role === "ADMIN" ? "ADMIN" : "USER" };
  } catch {
    return null;
  }
}

export type RefreshOutcome = { ok: true; session: SessionDTO } | { ok: false; status: number };

/** Trades a refresh token for a new session (the API rotates the refresh token). */
export async function refreshSession(refreshToken: string, clientIp?: string): Promise<RefreshOutcome> {
  const client = proxyApi(clientIp);
  if (!client) return { ok: false, status: 503 };
  try {
    const { data, response } = await client.POST("/v1/auth/refresh", { body: { refreshToken } });
    if (response.ok && data) return { ok: true, session: data };
    return { ok: false, status: response.status };
  } catch {
    return { ok: false, status: 503 };
  }
}
