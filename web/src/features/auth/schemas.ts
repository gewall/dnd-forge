import { z } from "zod";

// Mirrors the API limits (LoginRequest/RegisterRequest) for quick feedback;
// the API validates again.
const email = z.email("Masukkan alamat email yang valid.").max(254, "Email terlalu panjang.");

export const loginSchema = z.object({
  email,
  password: z.string().min(1, "Masukkan password.").max(128, "Password terlalu panjang."),
});

export const registerSchema = z.object({
  name: z.string().trim().max(80, "Nama maksimal 80 karakter."),
  email,
  password: z.string().min(8, "Password minimal 8 karakter.").max(128, "Password maksimal 128 karakter."),
});

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
