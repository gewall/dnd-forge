import { type Crumb, PageBreadcrumb } from "@/components/layout/page-breadcrumb";

export function AdminHeader({
  title,
  description,
  crumbs,
  actions,
}: {
  title: string;
  description?: string;
  crumbs?: Crumb[];
  actions?: React.ReactNode;
}) {
  return (
    <div className="mb-8">
      {crumbs && <PageBreadcrumb items={crumbs} />}
      <div className={crumbs ? "mt-4 flex flex-wrap items-end justify-between gap-4" : "flex flex-wrap items-end justify-between gap-4"}>
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{title}</h1>
          {description && <p className="mt-1 max-w-2xl text-muted-foreground">{description}</p>}
        </div>
        {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
      </div>
    </div>
  );
}

export const adminDate = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeStyle: "short",
  timeZone: "Asia/Jakarta",
});

export function formatDate(iso: string | null | undefined): string {
  return iso ? adminDate.format(new Date(iso)) : "-";
}
