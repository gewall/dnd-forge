"use client";

import { Bar, BarChart, CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import type { components } from "@/lib/api/schema";

type Day = components["schemas"]["DayDTO"];

// Palette validated with the dataviz checker: brass + violet pass lightness,
// chroma, CVD and normal-vision separation on both card surfaces.
const outcomeConfig = {
  succeeded: { label: "Sukses", color: "var(--chart-1)" },
  failed: { label: "Gagal", color: "var(--chart-2)" },
} satisfies ChartConfig;

const growthConfig = {
  users: { label: "User baru", color: "var(--chart-1)" },
  characters: { label: "Karakter baru", color: "var(--chart-2)" },
} satisfies ChartConfig;

const dayLabel = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", timeZone: "UTC" });
const fmtDay = (iso: string) => dayLabel.format(new Date(`${iso}T00:00:00Z`));

function OutcomeChart({ title, rows }: { title: string; rows: { date: string; succeeded: number; failed: number }[] }) {
  return (
    <figure className="rounded-xl border bg-card p-4">
      <figcaption className="mb-2 font-medium">{title}</figcaption>
      <ChartContainer config={outcomeConfig} className="aspect-auto h-56 w-full">
        <BarChart data={rows} margin={{ left: -16, right: 4, top: 8 }}>
          <CartesianGrid vertical={false} strokeOpacity={0.4} />
          <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} minTickGap={24} tickFormatter={fmtDay} />
          <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={40} />
          <ChartTooltip cursor={{ fillOpacity: 0.15 }} content={<ChartTooltipContent labelFormatter={(v) => fmtDay(String(v))} />} />
          <ChartLegend content={<ChartLegendContent />} />
          {/* 2px surface gap between stacked segments; rounded end only on the top segment. */}
          <Bar dataKey="succeeded" stackId="a" fill="var(--color-succeeded)" stroke="var(--card)" strokeWidth={2} />
          <Bar dataKey="failed" stackId="a" fill="var(--color-failed)" stroke="var(--card)" strokeWidth={2} radius={[4, 4, 0, 0]} />
        </BarChart>
      </ChartContainer>
    </figure>
  );
}

export function StatsCharts({ days }: { days: Day[] }) {
  const portraits = days.map((d) => ({ date: d.date, succeeded: d.portraitsSucceeded, failed: d.portraitsFailed }));
  const lores = days.map((d) => ({ date: d.date, succeeded: d.loresSucceeded, failed: d.loresFailed }));
  const growth = days.map((d) => ({ date: d.date, users: d.newUsers, characters: d.newCharacters }));

  return (
    <div className="space-y-4">
      <div className="grid gap-4 lg:grid-cols-2">
        <OutcomeChart title="Portrait per hari" rows={portraits} />
        <OutcomeChart title="Lore per hari" rows={lores} />
      </div>
      <figure className="rounded-xl border bg-card p-4">
        <figcaption className="mb-2 font-medium">User dan karakter baru per hari</figcaption>
        <ChartContainer config={growthConfig} className="aspect-auto h-56 w-full">
          <LineChart data={growth} margin={{ left: -16, right: 8, top: 8 }}>
            <CartesianGrid vertical={false} strokeOpacity={0.4} />
            <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={8} minTickGap={24} tickFormatter={fmtDay} />
            <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={40} />
            <ChartTooltip content={<ChartTooltipContent labelFormatter={(v) => fmtDay(String(v))} />} />
            <ChartLegend content={<ChartLegendContent />} />
            <Line dataKey="users" type="monotone" stroke="var(--color-users)" strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
            <Line dataKey="characters" type="monotone" stroke="var(--color-characters)" strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
          </LineChart>
        </ChartContainer>
      </figure>

      {/* Table view of the same rows for screen readers and exact values. */}
      <details className="rounded-xl border bg-card p-4">
        <summary className="cursor-pointer text-sm font-medium">Lihat sebagai tabel</summary>
        <div className="mt-3 max-h-80 overflow-auto">
          <table className="w-full text-sm tabular-nums">
            <thead className="sticky top-0 bg-card text-left text-muted-foreground">
              <tr>
                <th className="py-1.5 pr-3 font-medium">Tanggal</th>
                <th className="py-1.5 pr-3 text-right font-medium">Portrait sukses</th>
                <th className="py-1.5 pr-3 text-right font-medium">Portrait gagal</th>
                <th className="py-1.5 pr-3 text-right font-medium">Lore sukses</th>
                <th className="py-1.5 pr-3 text-right font-medium">Lore gagal</th>
                <th className="py-1.5 pr-3 text-right font-medium">User baru</th>
                <th className="py-1.5 text-right font-medium">Karakter baru</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {[...days].reverse().map((d) => (
                <tr key={d.date}>
                  <td className="py-1.5 pr-3">{fmtDay(d.date)}</td>
                  <td className="py-1.5 pr-3 text-right">{d.portraitsSucceeded}</td>
                  <td className="py-1.5 pr-3 text-right">{d.portraitsFailed}</td>
                  <td className="py-1.5 pr-3 text-right">{d.loresSucceeded}</td>
                  <td className="py-1.5 pr-3 text-right">{d.loresFailed}</td>
                  <td className="py-1.5 pr-3 text-right">{d.newUsers}</td>
                  <td className="py-1.5 text-right">{d.newCharacters}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </div>
  );
}
