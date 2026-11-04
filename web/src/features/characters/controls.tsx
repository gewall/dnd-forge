"use client";

import { Trash2Icon } from "lucide-react";
import { useState, useTransition } from "react";
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
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

import { deleteCharacter, setCharacterPublic } from "./actions";

export function PublicToggle({ id, isPublic }: { id: string; isPublic: boolean }) {
  const [checked, setChecked] = useState(isPublic);
  const [pending, startTransition] = useTransition();

  const change = (next: boolean) => {
    setChecked(next);
    startTransition(async () => {
      const result = await setCharacterPublic(id, next);
      if (!result.ok) {
        setChecked(!next);
        toast.error(result.message);
        return;
      }
      toast.success(next ? "Karakter tampil di galeri publik." : "Karakter disembunyikan dari galeri.");
    });
  };

  return (
    <div className="flex items-center gap-2">
      <Switch id="public-toggle" checked={checked} onCheckedChange={change} disabled={pending} />
      <Label htmlFor="public-toggle" className="font-normal">
        Tampil di galeri publik
      </Label>
    </div>
  );
}

export function DeleteCharacterButton({ id, name }: { id: string; name: string }) {
  const [pending, startTransition] = useTransition();

  const remove = () =>
    startTransition(async () => {
      // Redirects to /characters on success.
      const result = await deleteCharacter(id);
      if (!result.ok) toast.error(result.message);
    });

  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="destructive">
          <Trash2Icon />
          Hapus
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Hapus {name}?</AlertDialogTitle>
          <AlertDialogDescription>
            Karakter, portrait, dan draf lore-nya ikut hilang dan tidak bisa dikembalikan. Kuota yang sudah terpakai
            hari ini tidak kembali.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>Batal</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={remove} disabled={pending}>
            {pending ? "Menghapus…" : "Hapus karakter"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
