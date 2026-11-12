"use client";

// Design sheets: the images the operator generated in an external AI tool and
// uploaded back. A pack is cut from exactly one of them, so the picker and the
// upload grid share the same route and badges.

import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AI_PROVIDERS, PROVIDER_OPTIONS, providerName } from "@/lib/ai-providers";
import { api, errorText } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type { Sheet, SheetList } from "@/lib/types";
import { Icon } from "./icons";
import {
  ArtFrame,
  Button,
  ConfirmDialog,
  ErrorBanner,
  Field,
  Skeleton,
  Stamp,
  SuccessBanner,
  cx,
  inputClass,
  selectClass,
} from "./ui";

/** The list route omits image_url; the admin image route is keyed by both ids. */
export function sheetImageURL(designId: string, sheetId: string): string {
  return `/api/v1/admin/designs/${encodeURIComponent(designId)}/sheets/${encodeURIComponent(sheetId)}/image`;
}

export function useDesignSheets(designId: string | null) {
  return useQuery({
    queryKey: ["design-sheets", designId],
    queryFn: () => api<SheetList>(`/api/v1/admin/designs/${encodeURIComponent(designId ?? "")}/sheets`),
    enabled: Boolean(designId),
    refetchOnMount: "always",
  });
}

function ProviderBadge({ provider }: { provider: string }) {
  return <Stamp tone="active">{providerName(provider)}</Stamp>;
}

function SheetThumb({ sheet, selected }: { sheet: Sheet; selected?: boolean }) {
  return (
    <ArtFrame selected={selected} className="aspect-[4/3] w-full">
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={sheetImageURL(sheet.design_id, sheet.id)}
        alt={sheet.filename}
        loading="lazy"
        className="size-full object-contain p-1"
      />
    </ArtFrame>
  );
}

/** Upload form + thumbnail grid for one saved design. */
export function DesignSheets({ designId }: { designId: string }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const sheets = useDesignSheets(designId);
  const [file, setFile] = useState<File | null>(null);
  const [provider, setProvider] = useState<string>(AI_PROVIDERS[0].id);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<Sheet | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  async function upload(event: React.FormEvent) {
    event.preventDefault();
    if (!file) return;
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const form = new FormData();
      form.append("file", file);
      form.append("provider", provider);
      form.append("note", note.trim());
      await api<Sheet>(`/api/v1/admin/designs/${encodeURIComponent(designId)}/sheets`, { form });
      if (fileRef.current) fileRef.current.value = "";
      setFile(null);
      setNote("");
      setNotice(t("sheetUploaded"));
      qc.invalidateQueries({ queryKey: ["design-sheets", designId] });
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!deleting) return;
    setDeleteBusy(true);
    setDeleteError(null);
    try {
      await api<void>(
        `/api/v1/admin/designs/${encodeURIComponent(designId)}/sheets/${encodeURIComponent(deleting.id)}`,
        { method: "DELETE" },
      );
      setDeleting(null);
      setNotice(t("sheetDeleted"));
      qc.invalidateQueries({ queryKey: ["design-sheets", designId] });
    } catch (err) {
      setDeleteError(errorText(err));
    } finally {
      setDeleteBusy(false);
    }
  }

  const items = sheets.data?.items ?? [];

  return (
    <div data-qa="design-sheets" className="space-y-4">
      {notice && <SuccessBanner message={notice} onDismiss={() => setNotice(null)} />}
      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      <form onSubmit={upload} className="grid gap-3 sm:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-end">
        <Field label={t("sheetFile")} hint={t("uploadSheetHint")}>
          <input
            ref={fileRef}
            data-qa="sheet-file"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            onChange={(event) => setFile(event.target.files?.[0] ?? null)}
            className="block w-full text-sm text-ink-2 file:mr-3 file:min-h-9 file:cursor-pointer file:rounded-ctl file:border file:border-line-strong file:bg-panel file:px-3 file:text-sm file:font-medium file:text-ink hover:file:bg-sunk max-md:file:min-h-11"
          />
        </Field>
        <Field label={t("sheetProvider")} hint={t("sheetProviderHint")}>
          <select
            data-qa="sheet-provider"
            value={provider}
            onChange={(event) => setProvider(event.target.value)}
            className={selectClass}
          >
            {PROVIDER_OPTIONS.map((option) => (
              <option key={option.id} value={option.id}>
                {option.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("sheetNote")}>
          <input
            data-qa="sheet-note"
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder={t("sheetNotePlaceholder")}
            className={inputClass}
          />
        </Field>
        <Button type="submit" disabled={!file || busy} loading={busy} icon={busy ? undefined : "upload"}>
          {busy ? t("uploading") : t("sheetUpload")}
        </Button>
      </form>

      {sheets.isLoading ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          {[0, 1, 2].map((index) => (
            <Skeleton key={index} className="aspect-[4/3] w-full" />
          ))}
        </div>
      ) : sheets.isError ? (
        <div className="flex flex-wrap items-center gap-3">
          <ErrorBanner message={errorText(sheets.error)} />
          <Button variant="secondary" size="sm" icon="refresh" onClick={() => void sheets.refetch()}>
            {t("retry")}
          </Button>
        </div>
      ) : items.length === 0 ? (
        <div data-qa="sheet-empty" className="crop rounded-panel px-4 py-6 text-sm text-ink-2">
          <p className="font-medium text-ink">{t("noSheets")}</p>
          <p className="mt-1">{t("noSheetsHint")}</p>
        </div>
      ) : (
        <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          {items.map((sheet) => (
            <li
              key={sheet.id}
              data-qa="sheet-tile"
              data-sheet-id={sheet.id}
              data-provider={sheet.provider}
              className="flex min-w-0 flex-col rounded-panel border border-line p-2"
            >
              <SheetThumb sheet={sheet} />
              <div className="mt-2 flex items-center justify-between gap-2">
                <ProviderBadge provider={sheet.provider} />
                <span className="num shrink-0 text-[11px] text-ink-3">{formatDate(sheet.created_at)}</span>
              </div>
              <p className="mono mt-1 truncate text-xs text-ink" title={sheet.filename}>
                {sheet.filename}
              </p>
              <p className="num text-[11px] text-ink-3">
                {sheet.width}×{sheet.height} · {sheet.byte_size} B
              </p>
              {sheet.note && <p className="mt-1 break-words text-xs text-ink-2">{sheet.note}</p>}
              {sheet.prompt_snapshot && (
                <details data-qa="sheet-prompt" className="mt-1.5">
                  <summary className="cursor-pointer text-xs text-ink-2">{t("sheetPromptSnapshot")}</summary>
                  <pre
                    tabIndex={0}
                    aria-label={t("sheetPromptSnapshot")}
                    className="mono mt-1.5 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-ctl bg-sunk p-2 text-[11px] leading-4 text-ink-2"
                  >
                    {sheet.prompt_snapshot}
                  </pre>
                </details>
              )}
              <div className="mt-2 flex justify-end">
                <Button
                  variant="danger"
                  size="sm"
                  icon="trash"
                  aria-label={`${t("delete")} ${sheet.filename}`}
                  onClick={() => {
                    setDeleteError(null);
                    setDeleting(sheet);
                  }}
                >
                  {t("delete")}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {deleting && (
        <ConfirmDialog
          title={t("delete")}
          body={
            <>
              <p>{t("sheetDeleteConfirm")}</p>
              <p className="mono mt-2 text-xs text-ink-3">{deleting.filename}</p>
            </>
          }
          confirmLabel={t("delete")}
          onConfirm={() => void remove()}
          onClose={() => {
            if (deleteBusy) return;
            setDeleting(null);
            setDeleteError(null);
          }}
          busy={deleteBusy}
          error={deleteError}
        />
      )}
    </div>
  );
}

/** Selectable sheet grid reused by the new-pack form and the pack detail. */
export function SheetPicker({
  designId,
  value,
  onChange,
}: {
  designId: string;
  value: string;
  onChange: (sheetId: string) => void;
}) {
  const { t } = useI18n();
  const sheets = useDesignSheets(designId);
  const items = sheets.data?.items ?? [];

  if (sheets.isLoading) return <Skeleton className="h-36 w-full" />;
  if (sheets.isError) {
    return (
      <div className="flex flex-wrap items-center gap-3">
        <ErrorBanner message={errorText(sheets.error)} />
        <Button variant="secondary" size="sm" icon="refresh" onClick={() => void sheets.refetch()}>
          {t("retry")}
        </Button>
      </div>
    );
  }
  if (items.length === 0) {
    return (
      <div data-qa="sheet-empty" className="crop rounded-panel px-4 py-6 text-sm text-ink-2">
        <p className="font-medium text-ink">{t("noSheets")}</p>
        <p className="mt-1">{t("noSheetsHint")}</p>
        <Link
          href="/factory"
          className="mt-2 inline-flex min-h-8 items-center text-sm font-medium text-cut-ink hover:underline max-md:min-h-11"
        >
          {t("openDesignSheets")}
        </Link>
      </div>
    );
  }

  return (
    <div data-qa="sheet-picker">
      <p className="mb-3 text-xs text-ink-2">{t("sheetPickHint")}</p>
      <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {items.map((sheet) => {
          const selected = sheet.id === value;
          return (
            <li key={sheet.id}>
              <button
                type="button"
                aria-pressed={selected}
                data-qa="sheet-option"
                data-sheet-id={sheet.id}
                data-provider={sheet.provider}
                onClick={() => onChange(sheet.id)}
                className={cx(
                  "block w-full rounded-panel border p-2 text-left transition-colors",
                  selected ? "cutline border-cut/60 bg-cut-soft/40" : "border-line hover:border-line-strong hover:bg-sunk",
                )}
              >
                <span className="relative block">
                  <SheetThumb sheet={sheet} selected={selected} />
                  {selected && (
                    <span className="absolute right-1.5 top-1.5 inline-flex size-6 items-center justify-center rounded-full bg-cut text-on-cut">
                      <Icon name="check" size={14} />
                    </span>
                  )}
                </span>
                <span className="mt-2 flex flex-wrap items-center gap-1.5">
                  <ProviderBadge provider={sheet.provider} />
                  <span className="min-w-0 truncate text-xs text-ink-2">{sheet.filename}</span>
                </span>
                {sheet.note && <span className="mt-1 block truncate text-xs text-ink-3">{sheet.note}</span>}
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
