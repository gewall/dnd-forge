# dnd-forge/web

Frontend Next.js 16 (App Router, Cache Components) untuk DnD Guide & Character Forge, di-deploy ke Vercel
(Root Directory `web`, region `sin1`). Next.js berperan sebagai BFF: browser hanya bicara ke situs ini, dan server
Next.js memanggil Go API dengan header `X-Internal-Key`. Kontrak API ada di `../api/openapi.json`.

## Menjalankan lokal

```sh
cp .env.example .env.local   # isi INTERNAL_API_KEY
npm ci
npm run dev
```

Di Windows jalankan dari path dengan casing yang benar (`C:\Algi\dnd\web`). Kalau casing-nya beda, Turbopack
memuat React dua kali dan build gagal dengan "Cannot read properties of null (reading 'useContext')".

## Perintah

| Perintah | Fungsi |
| --- | --- |
| `npm run gen:api` | regenerasi `src/lib/api/schema.d.ts` dari `../api/openapi.json` |
| `npm run lint` | ESLint |
| `npm run typecheck` | tipe route Next.js + `tsc --noEmit` |
| `npm test` | unit test Vitest |
| `npm run build` | build produksi |

CI (`.github/workflows/web.yml`) menjalankan semua perintah di atas, dan menolak perubahan bila `schema.d.ts`
tertinggal dari `openapi.json`.

## Aturan

- Semua panggilan ke API lewat `api()` di `src/lib/api/client.ts` (server-only). Jangan memanggil API dengan
  `fetch` mentah, karena header internal key dan IP pengunjung bisa terlewat.
- Error API (problem+json) diubah ke `ActionResult` lewat `toResult()` di `src/lib/api/errors.ts`. Pesan `detail`
  berbahasa Indonesia dan aman ditampilkan langsung ke pengguna.
- Warna `primary` (kuningan) khusus untuk aksi utama. Warna `arcane` (ungu) khusus untuk fitur buatan AI.
