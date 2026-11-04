import { describe, expect, it } from "vitest";

import { fieldFromLocation, toFailure, toResult } from "./errors";

describe("fieldFromLocation", () => {
  it.each([
    ["body.name", "name"],
    ["body.items[2].tags", "items.2.tags"],
    ["path.id", "path.id"],
  ])("%s → %s", (location, field) => {
    expect(fieldFromLocation(location)).toBe(field);
  });
});

describe("toFailure", () => {
  it("keeps the API's Indonesian detail", () => {
    const f = toFailure(429, { status: 429, detail: "kuota generate hari ini sudah habis; coba lagi besok" });
    expect(f.message).toBe("kuota generate hari ini sudah habis; coba lagi besok");
    expect(f.status).toBe(429);
  });

  it("maps 422 field errors and replaces the generic detail", () => {
    const f = toFailure(422, {
      status: 422,
      detail: "validation failed",
      errors: [
        { location: "body.name", message: "expected length >= 1" },
        { location: "body.name", message: "second message is ignored" },
        { location: "body.email", message: "expected string to be RFC 5322 email" },
      ],
    });
    expect(f.message).toBe("Periksa lagi isian yang ditandai.");
    expect(f.fieldErrors).toEqual({
      name: "expected length >= 1",
      email: "expected string to be RFC 5322 email",
    });
  });

  it("falls back per status when the body has no detail", () => {
    expect(toFailure(503).message).toBe("Layanan sedang tidak tersedia. Hubungi admin.");
    expect(toFailure(500, {}).message).toBe("Terjadi kesalahan di server. Coba lagi.");
  });
});

describe("toResult", () => {
  it("returns data on success", () => {
    const r = toResult({ data: { id: "1" }, response: new Response(null, { status: 200 }) });
    expect(r).toEqual({ ok: true, data: { id: "1" } });
  });

  it("returns a failure on error status", () => {
    const r = toResult({
      error: { detail: "email sudah terdaftar" },
      response: new Response(null, { status: 409 }),
    });
    expect(r).toMatchObject({ ok: false, status: 409, message: "email sudah terdaftar" });
  });
});
