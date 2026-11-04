"use client";

import { EyeIcon, EyeOffIcon, Trash2Icon } from "lucide-react";
import { useTransition } from "react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

import { deleteImageFiles, setImageHidden } from "./actions";

export function HideToggle({ id, isHidden, hasFiles, size = "sm" }: { id: string; isHidden: boolean; hasFiles: boolean; size?: "sm" | "default" }) {
  const [pending, start] = useTransition();
  // Without files the portrait stays hidden for good.
  if (!hasFiles) return null;
  const toggle = () =>
    start(async () => {
      const r = await setImageHidden(id, !isHidden);
      if (r.ok) toast.success(isHidden ? "Portrait ditampilkan lagi." : "Portrait disembunyikan dari galeri dan link media.");
      else toast.error(r.message);
    });
  return (
    <Button size={size} variant={isHidden ? "outline" : "secondary"} onClick={toggle} disabled={pending}>
      {isHidden ? <EyeIcon /> : <EyeOffIcon />}
      {pending ? "Menyimpan…" : isHidden ? "Tampilkan" : "Sembunyikan"}
    </Button>
  );
}

export function DeleteFiles({ id, disabled }: { id: string; disabled?: boolean }) {
  const [pending, start] = useTransition();
  const run = () =>
    start(async () => {
      const r = await deleteImageFiles(id);
      if (r.ok) toast.success("File gambar dihapus.");
      else toast.error(r.message);
    });
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="destructive" disabled={disabled || pending}>
          <Trash2Icon />
          {pending ? "Menghapus…" : "Hapus file"}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Hapus file gambar ini?</AlertDialogTitle>
          <AlertDialogDescription>
            File asli dan thumbnail dihapus dari storage dan tidak bisa dikembalikan. Catatan portrait tetap ada (prompt, model) dan
            tersembunyi permanen. Pakai Sembunyikan kalau mungkin perlu ditampilkan lagi.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Batal</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={run}>
            Hapus file
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
