import type { components } from "@/lib/api/schema";
import { Progress } from "@/components/ui/progress";
import { requestApi, requireUser } from "@/lib/auth/session";

type Usage = components["schemas"]["UsageDTO"];

const dateFormat = new Intl.DateTimeFormat("id-ID", { dateStyle: "long", timeZone: "Asia/Jakarta" });
const timeFormat = new Intl.DateTimeFormat("id-ID", { timeStyle: "short", timeZone: "Asia/Jakarta" });

export async function ProfileDetails() {
  const user = await requireUser("/profile");
  const { data: quota } = await (await requestApi()).GET("/v1/me/quota");

  return (
    <div className="mt-8 space-y-8">
      <dl className="grid gap-x-8 gap-y-4 rounded-xl border bg-card p-5 sm:grid-cols-2">
        <Item label="Nama" value={user.name || "Belum diisi"} />
        <Item label="Email" value={user.email} />
        <Item label="Peran" value={user.role === "ADMIN" ? "Admin" : "Pemain"} />
        <Item label="Bergabung" value={dateFormat.format(new Date(user.createdAt))} />
      </dl>

      <section>
        <h2 className="text-xl font-semibold">Kuota hari ini</h2>
        {quota ? (
          <>
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <QuotaCard title="Portrait AI" usage={quota.portrait} />
              <QuotaCard title="Lore AI" usage={quota.lore} />
            </div>
            <p className="mt-3 text-sm text-muted-foreground">
              Kuota kembali penuh pukul {timeFormat.format(new Date(quota.portrait.resetsAt))} WIB. Permintaan yang gagal
              tidak mengurangi kuota.
            </p>
          </>
        ) : (
          <p className="mt-3 text-sm text-muted-foreground">Kuota belum bisa dimuat. Muat ulang halaman ini.</p>
        )}
      </section>
    </div>
  );
}

function Item({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 truncate">{value}</dd>
    </div>
  );
}

function QuotaCard({ title, usage }: { title: string; usage: Usage }) {
  const percent = usage.limit > 0 ? Math.round((usage.used / usage.limit) * 100) : 100;
  return (
    <div className="rounded-xl border bg-card p-5">
      <h3 className="font-sans text-base font-medium">{title}</h3>
      <p className="mt-2 text-3xl font-semibold tabular-nums">
        {usage.remaining}
        <span className="text-base font-normal text-muted-foreground"> / {usage.limit} tersisa</span>
      </p>
      <Progress
        value={percent}
        className="mt-3 [&>[data-slot=progress-indicator]]:bg-arcane"
        aria-label={`${usage.used} dari ${usage.limit} terpakai`}
      />
    </div>
  );
}
