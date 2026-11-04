import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

export interface ListRow {
  id: string;
  name: string;
  slug: string;
  isActive: boolean;
  sortOrder: number;
  extra?: string;
}

/** Admin table for races, classes, backgrounds (inactive rows included). */
export function GameDataTable({ rows, base, extraLabel }: { rows: ListRow[]; base: string; extraLabel?: string }) {
  if (rows.length === 0) return <p className="text-muted-foreground">Belum ada data.</p>;
  return (
    <div className="rounded-xl border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Nama</TableHead>
            <TableHead>Slug</TableHead>
            {extraLabel && <TableHead>{extraLabel}</TableHead>}
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Urutan</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell>
                <Link href={`${base}/${r.id}`} className="font-medium hover:text-primary hover:underline">
                  {r.name}
                </Link>
              </TableCell>
              <TableCell className="font-mono text-xs">{r.slug}</TableCell>
              {extraLabel && <TableCell>{r.extra}</TableCell>}
              <TableCell>{r.isActive ? "Aktif" : <span className="text-muted-foreground">Nonaktif</span>}</TableCell>
              <TableCell className="text-right tabular-nums">{r.sortOrder}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

export function NewButton({ href, label }: { href: string; label: string }) {
  return (
    <Button asChild>
      <Link href={href}>{label}</Link>
    </Button>
  );
}
