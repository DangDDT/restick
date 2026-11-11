"use client";

import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { DesignPanel } from "@/components/design-panel";
import { SheetPicker } from "@/components/design-sheets";
import { RequireAuth } from "@/components/guards";
import { Button, ErrorBanner, Field, PageHeader, Section, SkeletonRows, inputClass, selectClass } from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
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
  const { t } = useI18n();
  const router = useRouter();
  const designs = useQuery({ queryKey: ["designs-all"], queryFn: fetchAllDesigns });
  const [designId, setDesignId] = useState("");
  const [sheetId, setSheetId] = useState("");
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [author, setAuthor] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState("");
  const [busy, setBusy] = useState(false);
  const [designError, setDesignError] = useState<string | null>(null);
  const [sheetError, setSheetError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const design = designs.data?.find((item) => item.id === designId);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setDesignError(null);
    setSheetError(null);
    if (!designId) return setDesignError(t("designRequired"));
    if (!sheetId) return setSheetError(t("sheetRequired"));
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
        sheet_id: sheetId,
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
      <PageHeader title={t("newPack")} subtitle={t("packsSubtitle")} back={{ href: "/packs", label: t("back") }} />

      {error && (
        <div className="mb-5">
          <ErrorBanner message={error} />
        </div>
      )}

      {designs.isError && (
        <div className="mb-5 space-y-3">
          <ErrorBanner message={errorText(designs.error)} />
          <Button variant="secondary" icon="refresh" onClick={() => void designs.refetch()}>
            {t("retry")}
          </Button>
        </div>
      )}

      {designs.isLoading ? (
        <SkeletonRows rows={4} />
      ) : (
        <form onSubmit={submit} className="max-w-3xl" aria-busy={busy || undefined}>
          <Section title={t("designLabel")}>
            <Field
              label={t("designSelect")}
              hint={designs.data && designs.data.length === 0 ? t("noDesignsHint") : t("designPickHint")}
              error={designError ?? undefined}
            >
              <select
                value={designId}
                onChange={(event) => {
                  setDesignId(event.target.value);
                  // A sheet belongs to one design; switching designs drops the old choice.
                  setSheetId("");
                  setDesignError(null);
                  setSheetError(null);
                }}
                className={selectClass}
                aria-invalid={Boolean(designError) || undefined}
              >
                <option value="">{t("chooseDesign")}</option>
                {designs.data?.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} ({item.rows}×{item.columns})
                  </option>
                ))}
              </select>
            </Field>

            {design && (
              <div className="mt-5 space-y-5">
                <DesignPanel design={design} />
                <div>
                  <p className="mb-1.5 text-[13px] font-medium text-ink">{t("sheetLabel")}</p>
                  <SheetPicker
                    designId={design.id}
                    value={sheetId}
                    onChange={(next) => {
                      setSheetId(next);
                      setSheetError(null);
                    }}
                  />
                  {sheetError && (
                    <p role="alert" className="mt-1.5 text-xs text-danger">
                      {sheetError}
                    </p>
                  )}
                </div>
              </div>
            )}
          </Section>

          <Section title={t("pkCatalogSection")}>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t("packName")}>
                <input value={name} onChange={(event) => setName(event.target.value)} className={inputClass} required />
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
                <input value={author} onChange={(event) => setAuthor(event.target.value)} className={inputClass} />
              </Field>
              <Field label={t("packTags")}>
                <input value={tags} onChange={(event) => setTags(event.target.value)} className={inputClass} />
              </Field>
            </div>
            <div className="mt-4">
              <Field label={t("packDescription")}>
                <textarea
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                  rows={3}
                  className={inputClass}
                />
              </Field>
            </div>
          </Section>

          <div className="flex justify-end pt-2">
            <Button type="submit" disabled={busy} loading={busy} icon={busy ? undefined : "plus"}>
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
