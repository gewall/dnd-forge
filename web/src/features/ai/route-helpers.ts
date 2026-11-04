import "server-only";

import { NextResponse } from "next/server";
import { z } from "zod";

const uuid = z.uuid();

/** JSON reply for a BFF route: never cached, problem+json passed through as-is. */
export function jsonReply(body: unknown, status: number): NextResponse {
  return NextResponse.json(body ?? null, { status, headers: { "Cache-Control": "no-store" } });
}

export function isUuid(value: string): boolean {
  return uuid.safeParse(value).success;
}
