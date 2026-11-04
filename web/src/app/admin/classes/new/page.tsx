import { ClassForm } from "@/features/admin/content/game-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Tambah kelas" };

export default function Page() {
  return (
    <>
      <AdminHeader title="Tambah kelas" crumbs={[{ label: "Kelas", href: "/admin/classes" }, { label: "Baru" }]} />
      <ClassForm />
    </>
  );
}
