import { RaceForm } from "@/features/admin/content/game-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Tambah ras" };

export default function Page() {
  return (
    <>
      <AdminHeader title="Tambah ras" crumbs={[{ label: "Ras", href: "/admin/races" }, { label: "Baru" }]} />
      <RaceForm />
    </>
  );
}
