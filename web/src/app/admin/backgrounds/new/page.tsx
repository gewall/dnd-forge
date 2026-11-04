import { BackgroundForm } from "@/features/admin/content/game-forms";
import { AdminHeader } from "@/features/admin/page-header";

export const metadata = { title: "Tambah latar belakang" };

export default function Page() {
  return (
    <>
      <AdminHeader title="Tambah latar belakang" crumbs={[{ label: "Latar belakang", href: "/admin/backgrounds" }, { label: "Baru" }]} />
      <BackgroundForm />
    </>
  );
}
