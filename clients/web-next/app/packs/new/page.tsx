"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  ErrorBanner,
  Field,
  PageHeader,
  Spinner,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { t } from "@/lib/i18n";
import type { AdminPack, Design, DesignPage, PackCreate } from "@/lib/types";

async function fetchAllDesigns(): Promise<Design[]> {
  const items: Design[] = [];
  for (let page = 1; ; page++) {
    const res = await api<DesignPage>(`/api/v1/admin/designs?page=${page}&page_size=100`);
    items.push(...res.items);
    if (res.items.length === 0 || items.length >= res.total) return items;
  }
}

function NewPack() {
  const router = useRouter();
  const designs = useQuery({ queryKey: ["designs-all"], queryFn: fetchAllDesigns });
  const [designId, setDesignId] = useState("");
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [author, setAuthor] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    if (!designId) return setError(t("designRequired"));
    setBusy(true);
    try {
      const body: PackCreate = {
        slug: slug.trim(),
        name: name.trim(),
        author: author.trim(),
        description: description.trim(),
        tags: tags
          .split(",")
          .map((tag) => tag.trim())
          .filter((tag) => tag.length > 0),
        design_id: designId,
      };
      const created = await api<AdminPack>("/api/v1/admin/packs", { method: "POST", json: body });
      router.replace(`/packs/detail?id=${encodeURIComponent(created.id)}`);
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeader
        title={t("newPack")}
        subtitle={t("packsSubtitle")}
        actions={
          <Link
            href="/packs"
            className="rounded-lg border border-zinc-300 bg-white px-4 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-50"
          >
            {t("back")}
          </Link>
        }
      />

      {error && (
        <div className="mb-4">
          <ErrorBanner message={error} />
        </div>
      )}

      {designs.isError && (
        <div className="mb-4">
          <ErrorBanner message={errorText(designs.error)} />
        </div>
      )}

      {designs.isLoading ? (
        <Spinner />
      ) : (
        <form
          onSubmit={submit}
          className="max-w-2xl space-y-4 rounded-xl border border-zinc-200 bg-white p-6"
        >
          <Field
            label={t("designSelect")}
            hint={
              designs.data && designs.data.length === 0 ? t("noDesignsHint") : t("designPickHint")
            }
          >
            <select
              value={designId}
              onChange={(event) => setDesignId(event.target.value)}
              className={inputClass}
            >
              <option value="">{t("chooseDesign")}</option>
              {designs.data?.map((design) => (
                <option key={design.id} value={design.id}>
                  {design.name} ({design.rows}×{design.columns})
                </option>
              ))}
            </select>
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label={t("packName")}>
              <input
                value={name}
                onChange={(event) => setName(event.target.value)}
                className={inputClass}
                required
              />
            </Field>
            <Field label={t("packSlug")} hint={t("slugHint")}>
              <input
                value={slug}
                onChange={(event) => setSlug(event.target.value)}
                className={inputClass}
                pattern="[a-z0-9]+(-[a-z0-9]+)*"
                required
              />
            </Field>
            <Field label={t("packAuthor")}>
              <input
                value={author}
                onChange={(event) => setAuthor(event.target.value)}
                className={inputClass}
              />
            </Field>
            <Field label={t("packTags")}>
              <input
                value={tags}
                onChange={(event) => setTags(event.target.value)}
                className={inputClass}
              />
            </Field>
          </div>
          <Field label={t("packDescription")}>
            <textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              rows={3}
              className={inputClass}
            />
          </Field>
          <div className="flex justify-end">
            <Button type="submit" disabled={busy}>
              {busy ? t("creatingPack") : t("createPack")}
            </Button>
          </div>
        </form>
      )}
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
