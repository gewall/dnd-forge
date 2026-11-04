import Link from "next/link";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { adminApi, adminData, param } from "@/features/admin/api";
import { AdminHeader } from "@/features/admin/page-header";
import { StatsCharts } from "@/features/admin/stats-charts";
import type { components } from "@/lib/api/schema";
import { cn } from "@/lib/utils";

export const metadata = { title: "Ringkasan" };

const periods = [7, 30, 90] as const;
const num = new Intl.NumberFormat("id-ID");

type Props = PageProps<"/admin">;

export default function AdminHome({ searchParams }: Props) {
  return (
    <>
      <AdminHeader title="Ringkasan" description="Total sepanjang waktu, lalu aktivitas harian dalam periode yang dipilih (hari WIB)." />
      <Suspense fallback={<Skeleton className="h-96 w-full" />}>
        <Stats searchParams={searchParams} />
      </Suspense>
    </>
  );
}

async function Stats({ searchParams }: Pick<Props, "searchParams">) {
  const requested = Number(param((await searchParams).days));
  const days = (periods as readonly number[]).includes(requested) ? requested : 30;
  const report = adminData(
    await (await adminApi()).GET("/v1/admin/stats", { params: { query: { days } } }),
    "statistik",
  );
  if (!report) return <p className="text-muted-foreground">Statistik belum bisa dimuat.</p>;
  const t = report.totals;

  return (
    <div className="space-y-10">
      <section aria-label="Total">
        <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <Tile label="User" value={t.users} sub={`${t.admins} admin, ${t.bannedUsers} diblokir`} />
          <Tile label="Karakter" value={t.characters} sub={`${t.publicCharacters} tampil di galeri`} />
          <Tile
            label="Portrait sukses"
            value={t.portraitsSucceeded}
            sub={`${t.portraitsFailed} gagal, ${t.portraitsPending} diproses, ${t.portraitsHidden} disembunyikan`}
          />
          <Tile label="Lore sukses" value={t.loresSucceeded} sub={`${t.loresFailed} gagal, ${t.loresPending} diproses`} />
        </dl>
      </section>

      <section aria-labelledby="daily">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h2 id="daily" className="text-xl font-semibold">
            Aktivitas harian
          </h2>
          <nav aria-label="Periode" className="inline-flex rounded-lg border p-0.5">
            {periods.map((p) => (
              <Link
                key={p}
                href={`/admin?days=${p}`}
                aria-current={p === days ? "page" : undefined}
                className={cn(
                  "rounded-md px-3 py-1 text-sm",
                  p === days ? "bg-secondary font-medium text-secondary-foreground" : "text-muted-foreground hover:text-foreground",
                )}
              >
                {p} hari
              </Link>
            ))}
          </nav>
        </div>
        <StatsCharts days={report.days ?? []} />
      </section>

      <section aria-labelledby="models">
        <h2 id="models" className="text-xl font-semibold">
          Pemakaian AI per model
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">Dalam {days} hari terakhir, tersibuk dulu.</p>
        <ModelTable models={report.models ?? []} />
      </section>
    </div>
  );
}

function Tile({ label, value, sub }: { label: string; value: number; sub: string }) {
  return (
    <div className="rounded-xl border bg-card p-5">
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-3xl font-semibold tabular-nums">{num.format(value)}</dd>
      <dd className="mt-1 text-sm text-muted-foreground">{sub}</dd>
    </div>
  );
}

function ModelTable({ models }: { models: components["schemas"]["ModelUsageDTO"][] }) {
  if (models.length === 0) {
    return <p className="mt-4 text-muted-foreground">Belum ada permintaan AI dalam periode ini.</p>;
  }
  return (
    <div className="mt-4 rounded-xl border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Provider</TableHead>
            <TableHead>Model</TableHead>
            <TableHead>Jenis</TableHead>
            <TableHead className="text-right">Permintaan</TableHead>
            <TableHead className="text-right">Sukses</TableHead>
            <TableHead className="text-right">Gagal</TableHead>
            <TableHead className="text-right">Token prompt</TableHead>
            <TableHead className="text-right">Token jawaban</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody className="tabular-nums">
          {models.map((m) => (
            <TableRow key={`${m.providerId}-${m.model}`}>
              <TableCell>{m.providerLabel}</TableCell>
              <TableCell className="font-mono text-xs">{m.model}</TableCell>
              <TableCell>{m.capability === "image" ? "Gambar" : "Teks"}</TableCell>
              <TableCell className="text-right">{num.format(m.requests)}</TableCell>
              <TableCell className="text-right">{num.format(m.succeeded)}</TableCell>
              <TableCell className="text-right">{num.format(m.failed)}</TableCell>
              <TableCell className="text-right">{num.format(m.promptTokens)}</TableCell>
              <TableCell className="text-right">{num.format(m.completionTokens)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
