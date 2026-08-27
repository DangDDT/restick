"use client";

import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { useApi } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { CreatePackResponse } from "@/lib/types";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { Button, ErrorBanner, Field, inputClass, PageHeader } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

interface PickedFile {
  name: string;
  bytes: ArrayBuffer;
  contentType: string;
}

function contentTypeFor(ext: string): string {
  if (ext === "jpg" || ext === "jpeg") return "image/jpeg";
  if (ext === "webp") return "image/webp";
  return "image/png";
}

function safeName(name: string, index: number): string {
  const cleaned = name.replace(/[^A-Za-z0-9._-]/g, "_");
  return cleaned.length > 0 ? cleaned : `file_${index}.png`;
}

function NewPack() {
  const { request } = useApi();
  const { t } = useI18n();
  const { user } = useAuth();
  const router = useRouter();
  const fileInput = useRef<HTMLInputElement>(null);

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [author, setAuthor] = useState(user?.display_name ?? "");
  const [tags, setTags] = useState("");
  const [sourceKind, setSourceKind] = useState<"sheet" | "files">("sheet");
  const [files, setFiles] = useState<PickedFile[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onPick(e: React.ChangeEvent<HTMLInputElement>) {
    const list = e.target.files;
    if (!list || list.length === 0) return;
    const picked: PickedFile[] = [];
    for (let i = 0; i < list.length; i++) {
      const f = list[i];
      const ext = (f.name.split(".").pop() ?? "png").toLowerCase();
      const bytes = await f.arrayBuffer();
      picked.push({
        name: sourceKind === "sheet" ? `sheet.${ext}` : safeName(f.name, i),
        bytes,
        contentType: contentTypeFor(ext),
      });
    }
    setFiles(picked);
    if (e.target) e.target.value = "";
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (files.length === 0) {
      setError(t("chooseImages"));
      return;
    }
    setBusy(true);
    try {
      const tagList = tags
        .split(",")
        .map((t) => t.trim())
        .filter((t) => t.length > 0);

      const created = await request<CreatePackResponse>("/api/v1/packs", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          slug: slug.trim(),
          author: author.trim(),
          tags: tagList,
          source_kind: sourceKind,
        }),
      });

      for (const f of files) {
        await request<void>(`/v1/objects/${created.upload_prefix}${f.name}`, {
          method: "PUT",
          headers: { "Content-Type": f.contentType },
          body: f.bytes,
        });
      }

      await request<void>(`/api/v1/packs/${created.id}/submit`, { method: "POST" });
      router.replace(`/packs/${created.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("uploadFailed"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeader title={t("uploadPackTitle")} subtitle={t("uploadPackSubtitle")} />
      {error && <div className="mb-4"><ErrorBanner message={error} /></div>}

      <form onSubmit={onSubmit} className="max-w-2xl space-y-5 rounded-xl border border-zinc-200 bg-white p-6">
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <Field label={t("packName")}>
            <input value={name} onChange={(e) => setName(e.target.value)} required className={inputClass} placeholder={t("packNamePlaceholder")} />
          </Field>
          <Field label={t("packSlug")}>
            <input value={slug} onChange={(e) => setSlug(e.target.value)} required className={inputClass} placeholder={t("packSlugPlaceholder")} />
          </Field>
        </div>
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <Field label={t("packAuthor")}>
            <input value={author} onChange={(e) => setAuthor(e.target.value)} className={inputClass} placeholder="Author name" />
          </Field>
          <Field label={t("packTags")}>
            <input value={tags} onChange={(e) => setTags(e.target.value)} className={inputClass} placeholder={t("packTagsPlaceholder")} />
          </Field>
        </div>

        <Field label={t("sourceKind")}>
          <div className="flex gap-2">
            {(["sheet", "files"] as const).map((k) => (
              <button
                key={k}
                type="button"
                onClick={() => setSourceKind(k)}
                className={`rounded-lg border px-4 py-2 text-sm font-medium capitalize ${
                  sourceKind === k
                    ? "border-indigo-600 bg-indigo-50 text-indigo-700"
                    : "border-zinc-300 bg-white text-zinc-600 hover:bg-zinc-50"
                }`}
              >
                {k === "sheet" ? t("sourceSheet") : t("sourceFiles")}
              </button>
            ))}
          </div>
        </Field>

        <Field label={t("images")} hint={sourceKind === "sheet" ? t("sheetHint") : t("filesHint")}>
          <input
            ref={fileInput}
            type="file"
            accept="image/*"
            multiple={sourceKind === "files"}
            onChange={onPick}
            className="block w-full text-sm text-zinc-600 file:mr-4 file:rounded-lg file:border-0 file:bg-indigo-50 file:px-4 file:py-2 file:text-sm file:font-medium file:text-indigo-700 hover:file:bg-indigo-100"
          />
          {files.length > 0 && (
            <ul className="mt-2 space-y-1 text-xs text-zinc-600">
              {files.map((f, i) => (
                <li key={i} className="truncate">
                  {f.name} <span className="text-zinc-400">({f.contentType})</span>
                </li>
              ))}
            </ul>
          )}
        </Field>

        <div className="flex justify-end gap-2 pt-2">
          <Button type="submit" disabled={busy}>
            {busy ? t("uploading") : t("createSubmit")}
          </Button>
        </div>
      </form>
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <NewPack />
      </AppShell>
    </RequireAuth>
  );
}
