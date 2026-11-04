import { CheckCircle2Icon, CircleAlertIcon } from "lucide-react";
import { connection } from "next/server";

import { api } from "@/lib/api/client";

interface Check {
  label: string;
  ok: boolean;
}

async function runChecks(): Promise<Check[]> {
  try {
    const client = api();
    const [health, races] = await Promise.all([client.GET("/healthz"), client.GET("/v1/races")]);
    const raceCount = races.data?.items?.length ?? 0;
    return [
      { label: health.response.ok ? "Server data tersambung" : "Server data tidak menjawab", ok: health.response.ok },
      {
        label: races.response.ok ? `${raceCount} ras siap dipilih` : "Data ras belum bisa dimuat",
        ok: races.response.ok && raceCount > 0,
      },
    ];
  } catch {
    return [{ label: "Server data tidak terjangkau", ok: false }];
  }
}

/** Live check of the Vercel → Go path; rendered per request. */
export async function ServiceStatus() {
  await connection();
  const checks = await runChecks();

  return (
    <ul className="mt-4 space-y-2">
      {checks.map((c) => (
        <li key={c.label} className="flex items-center gap-2 text-sm">
          {c.ok ? (
            <CheckCircle2Icon className="size-4 text-primary" aria-hidden="true" />
          ) : (
            <CircleAlertIcon className="size-4 text-destructive" aria-hidden="true" />
          )}
          {c.label}
        </li>
      ))}
    </ul>
  );
}
