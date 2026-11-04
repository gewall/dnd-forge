"use client";

import { useState } from "react";

import { Markdown } from "@/components/markdown/markdown";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import type { components } from "@/lib/api/schema";

import { slugify } from "../form-utils";
import { deleteArticle, deleteCategory, saveArticle, saveCategory } from "./actions";
import { ConfirmDelete, ImageField, NumberField, TextAreaField, TextField, useAdminAction } from "./fields";

type Category = components["schemas"]["CategoryDTO"];
type Article = components["schemas"]["ArticleDTO"];

function CategoryForm({ category, onDone }: { category?: Category; onDone?: () => void }) {
  const [title, setTitle] = useState(category?.title ?? "");
  const [slug, setSlug] = useState(category?.slug ?? "");
  const [slugTouched, setSlugTouched] = useState(!!category);
  const [description, setDescription] = useState(category?.description ?? "");
  const [sortOrder, setSortOrder] = useState(category?.sortOrder ?? 0);
  const { pending, errors, run } = useAdminAction(category ? "Kategori disimpan." : "Kategori ditambahkan.");

  const submit = () =>
    run(async () => {
      const r = await saveCategory(category?.id ?? null, { title: title.trim(), slug, description: description.trim(), sortOrder });
      if (r.ok) {
        if (!category) {
          setTitle("");
          setSlug("");
          setSlugTouched(false);
          setDescription("");
        }
        onDone?.();
      }
      return r;
    });

  return (
    <FieldGroup className="max-w-2xl">
      <div className="grid gap-4 sm:grid-cols-2">
        <TextField
          id={`ct-${category?.id ?? "new"}`}
          label="Judul"
          required
          maxLength={120}
          value={title}
          error={errors.title}
          onChange={(v) => {
            setTitle(v);
            if (!slugTouched) setSlug(slugify(v));
          }}
        />
        <TextField
          id={`cs-${category?.id ?? "new"}`}
          label="Slug"
          required
          mono
          maxLength={64}
          value={slug}
          error={errors.slug}
          description="Bagian URL: /guide/slug"
          onChange={(v) => {
            setSlugTouched(true);
            setSlug(v);
          }}
        />
      </div>
      <TextAreaField id={`cd-${category?.id ?? "new"}`} label="Deskripsi" rows={2} maxLength={1000} value={description} error={errors.description} onChange={setDescription} />
      <NumberField id={`co-${category?.id ?? "new"}`} label="Urutan" value={sortOrder} error={errors.sortOrder} onChange={setSortOrder} description="Kecil tampil lebih dulu." />
      <div className="flex gap-2">
        <Button onClick={submit} disabled={pending}>
          {pending ? "Menyimpan…" : category ? "Simpan" : "Tambah kategori"}
        </Button>
        {onDone && category && (
          <Button variant="ghost" onClick={onDone}>
            Batal
          </Button>
        )}
      </div>
    </FieldGroup>
  );
}

export function CategoryManager({ categories, counts }: { categories: Category[]; counts: Record<string, number> }) {
  const [editing, setEditing] = useState<string | null>(null);
  const del = useAdminAction("Kategori dihapus.");

  return (
    <div className="space-y-10">
      <div className="rounded-xl border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Judul</TableHead>
              <TableHead>Slug</TableHead>
              <TableHead className="text-right">Artikel</TableHead>
              <TableHead className="text-right">Urutan</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {categories.map((c) =>
              editing === c.id ? (
                <TableRow key={c.id}>
                  <TableCell colSpan={5} className="py-4">
                    <CategoryForm category={c} onDone={() => setEditing(null)} />
                  </TableCell>
                </TableRow>
              ) : (
                <TableRow key={c.id}>
                  <TableCell className="font-medium">{c.title}</TableCell>
                  <TableCell className="font-mono text-xs">{c.slug}</TableCell>
                  <TableCell className="text-right tabular-nums">{counts[c.id] ?? 0}</TableCell>
                  <TableCell className="text-right tabular-nums">{c.sortOrder}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button size="sm" variant="outline" onClick={() => setEditing(c.id)}>
                        Ubah
                      </Button>
                      <ConfirmDelete
                        title={`Hapus kategori ${c.title}?`}
                        description="Hanya kategori tanpa artikel yang bisa dihapus."
                        disabled={del.pending}
                        onConfirm={() => del.run(() => deleteCategory(c.id))}
                      />
                    </div>
                  </TableCell>
                </TableRow>
              ),
            )}
          </TableBody>
        </Table>
      </div>
      <section>
        <h2 className="mb-4 text-xl font-semibold">Kategori baru</h2>
        <CategoryForm />
      </section>
    </div>
  );
}

export function ArticleForm({ article, categories }: { article?: Article; categories: Category[] }) {
  const [categoryId, setCategoryId] = useState(article?.categoryId ?? categories[0]?.id ?? "");
  const [title, setTitle] = useState(article?.title ?? "");
  const [slug, setSlug] = useState(article?.slug ?? "");
  const [slugTouched, setSlugTouched] = useState(!!article);
  const [excerpt, setExcerpt] = useState(article?.excerpt ?? "");
  const [content, setContent] = useState(article?.content ?? "");
  const [status, setStatus] = useState<"DRAFT" | "PUBLISHED">(article?.status ?? "DRAFT");
  const [sortOrder, setSortOrder] = useState(article?.sortOrder ?? 0);
  const [cover, setCover] = useState({ key: article?.coverKey ?? "", url: article?.coverUrl });
  const save = useAdminAction(article ? "Artikel disimpan." : "Artikel dibuat.");
  const del = useAdminAction("Artikel dihapus.");

  const submit = () =>
    save.run(() =>
      saveArticle(article?.id ?? null, {
        categoryId,
        title: title.trim(),
        slug,
        excerpt: excerpt.trim(),
        content,
        status,
        sortOrder,
        coverKey: cover.key,
      }),
    );
  const e = save.errors;

  return (
    <div className="grid gap-10 xl:grid-cols-[minmax(0,1fr)_20rem]">
      <FieldGroup className="min-w-0">
        <TextField
          id="a-title"
          label="Judul"
          required
          maxLength={160}
          value={title}
          error={e.title}
          onChange={(v) => {
            setTitle(v);
            if (!slugTouched) setSlug(slugify(v));
          }}
        />
        <TextAreaField id="a-excerpt" label="Ringkasan" rows={2} maxLength={500} value={excerpt} error={e.excerpt} onChange={setExcerpt} description="Tampil di daftar artikel dan hasil pencarian." />
        <Field data-invalid={!!e.content}>
          <FieldLabel htmlFor="a-content">Isi (Markdown)</FieldLabel>
          <Tabs defaultValue="write">
            <TabsList>
              <TabsTrigger value="write">Tulis</TabsTrigger>
              <TabsTrigger value="preview">Pratinjau</TabsTrigger>
            </TabsList>
            <TabsContent value="write">
              <Textarea id="a-content" rows={24} value={content} onChange={(ev) => setContent(ev.target.value)} className="font-mono text-sm" />
            </TabsContent>
            <TabsContent value="preview">
              <div className="min-h-96 rounded-lg border p-5">
                {content.trim() ? <Markdown source={content} /> : <p className="text-muted-foreground">Belum ada isi.</p>}
              </div>
            </TabsContent>
          </Tabs>
          <FieldError>{e.content}</FieldError>
        </Field>
      </FieldGroup>

      <FieldGroup className="xl:sticky xl:top-20 xl:self-start">
        <Field data-invalid={!!e.status}>
          <FieldLabel htmlFor="a-status">Status</FieldLabel>
          <Select value={status} onValueChange={(v) => setStatus(v as typeof status)}>
            <SelectTrigger id="a-status" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="DRAFT">Draf (tidak tampil)</SelectItem>
              <SelectItem value="PUBLISHED">Terbit</SelectItem>
            </SelectContent>
          </Select>
        </Field>
        <Field data-invalid={!!e.categoryId}>
          <FieldLabel htmlFor="a-cat">Kategori</FieldLabel>
          <Select value={categoryId} onValueChange={setCategoryId}>
            <SelectTrigger id="a-cat" className="w-full">
              <SelectValue placeholder="Pilih kategori" />
            </SelectTrigger>
            <SelectContent>
              {categories.map((c) => (
                <SelectItem key={c.id} value={c.id}>
                  {c.title}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldError>{e.categoryId}</FieldError>
        </Field>
        <TextField
          id="a-slug"
          label="Slug"
          required
          mono
          maxLength={64}
          value={slug}
          error={e.slug}
          onChange={(v) => {
            setSlugTouched(true);
            setSlug(v);
          }}
        />
        <NumberField id="a-order" label="Urutan" value={sortOrder} error={e.sortOrder} onChange={setSortOrder} />
        <ImageField id="a-cover" label="Gambar sampul" imageKey={cover.key} imageUrl={cover.url} onChange={(key, url) => setCover({ key, url })} />
        <div className="flex flex-wrap gap-2 border-t pt-4">
          <Button onClick={submit} disabled={save.pending || !categoryId}>
            {save.pending ? "Menyimpan…" : article ? "Simpan artikel" : "Buat artikel"}
          </Button>
          {article && (
            <ConfirmDelete
              title={`Hapus ${article.title}?`}
              description="Artikel hilang dari panduan dan tidak bisa dikembalikan."
              disabled={del.pending}
              onConfirm={() => del.run(() => deleteArticle(article.id))}
            />
          )}
        </div>
      </FieldGroup>
    </div>
  );
}
