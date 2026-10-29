# dnd-forge/api

Backend Go untuk DnD Guide & Character Forge: chi + Huma v2 (OpenAPI 3.1), pgx v5 + sqlc, goose, River.

## Menjalankan lokal

```sh
cp .env.example .env              # isi secret dari: go run ./cmd/tools gen-keys
ssh -N -L 5432:127.0.0.1:5432 vps # tunnel ke DB dev di VPS
set -a; . ./.env; set +a
go run ./cmd/api                  # migrasi jalan otomatis (MIGRATE_ON_START)
go run ./cmd/tools seed                       # data SRD 5.2 + kategori panduan (aman diulang)
go run ./cmd/tools seed-admin -email you@example.com
```

Dengan `APP_ENV=development`, dokumentasi interaktif terbuka di `http://localhost:8080/docs`.
Semua request lain wajib membawa header `X-Internal-Key`.

## Perintah

| Perintah | Fungsi |
| --- | --- |
| `go test ./...` | unit test |
| `TEST_DATABASE_URL=... go test -p 1 -tags integration ./...` | test store ke PostgreSQL sungguhan |
| `sqlc generate` | regenerasi `internal/db/sqlc` dari `internal/db/queries/*.sql` |
| `go run ./cmd/tools openapi` | tulis `openapi.json` (dipakai frontend untuk generate tipe TS) |
| `go run ./cmd/tools migrate` | jalankan migrasi tanpa start server |
| `go run ./cmd/tools seed` | isi ras/kelas/latar belakang SRD 5.2 dan kategori panduan yang belum ada |

CI menolak perubahan bila `sqlc generate` atau `openapi.json` belum diperbarui.

## Struktur

Satu package per fitur, dependensi satu arah `http.go → service.go → interface`:

- `service.go` mendefinisikan interface kecil yang ia butuhkan (mis. `auth.UserStore`, `auth.TokenStore`).
- `store.go` memenuhi interface itu dengan sqlc; tipe sqlc tidak keluar dari file ini.
- `http.go` hanya memetakan DTO ↔ domain dan error lewat `httpapi.Error`.
- Error domain dibuat dengan `apperr.New(kind, pesan)`; `httpapi` menerjemahkan kind → status HTTP.
- Semua adapter konkret dirangkai di `internal/app` (composition root), tanpa global state.

Aturan akses operasi: `httpapi.Authenticated`, `httpapi.AdminOnly`, `httpapi.OptionalAuth`,
`httpapi.RateLimited(op, "auth")`.

## Gambar

Upload admin (`POST /v1/admin/uploads`) dikonversi ke WebP maks 1600px oleh encoder pure Go
(`internal/imaging`, tanpa cgo) lalu disimpan privat di Garage. Response dan DTO hanya memuat URL
stabil `PUBLIC_BASE_URL/v1/media/content/<id>.webp`, yang menjawab 302 ke URL presigned
`s3.gewall.my.id` (berlaku `PRESIGN_TTL`). Endpoint media tidak butuh `X-Internal-Key`.

## Lisensi konten

Data ras, kelas, dan latar belakang di `internal/gamedata/seed` berasal dari SRD 5.2 (CC BY 4.0);
deskripsinya ringkasan sendiri dalam bahasa Indonesia. Situs wajib menampilkan atribusi berikut:

> This work includes material from the System Reference Document 5.2 ("SRD 5.2") by Wizards of
> the Coast LLC, available at https://www.dndbeyond.com/srd. The SRD 5.2 is licensed under the
> Creative Commons Attribution 4.0 International License, available at
> https://creativecommons.org/licenses/by/4.0/legalcode.
