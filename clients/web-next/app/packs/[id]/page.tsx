"use client";

import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, useState } from "react";
import { useApi } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { PackDetail, Sticker } from "@/lib/types";
import {
  REJECTION_CODES,
  REJECTION_DEFAULT_TEXT,
  REJECTION_LABELS,
  REVOCATION_CODES,
  REVOCATION_DEFAULT_TEXT,
  REVOCATION_LABELS,
} from "@/lib/constants";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  EmptyState,
  ErrorBanner,
  Field,
  inputClass,
  PageHeader,
  Spinner,
  StatusPill,
} from "@/components/ui";
import { useI18n } from "@/lib/i18n";

function ReasonDialog({
  title,
  codes,
  labels,
  defaults,
  reasonLabel,
  textLabel,
  cancelLabel,
  confirmLabel,
  onConfirm,
  onClose,
}: {
  title: string;
  codes: readonly string[];
  labels: Record<string, string>;
  defaults: Record<string, string>;
  reasonLabel: string;
  textLabel: string;
  cancelLabel: string;
  confirmLabel: string;
  onConfirm: (code: string, text: string) => void;
  onClose: () => void;
}) {
  const [code, setCode] = useState(codes[0]);
  const [text, setText] = useState(defaults[codes[0]]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl">
        <h2 className="text-lg font-bold text-zinc-900">{title}</h2>
        <div className="mt-4 space-y-4">
          <Field label={reasonLabel}>
            <select
              value={code}
              onChange={(e) => {
                setCode(e.target.value);
                setText(defaults[e.target.value]);
              }}
              className={inputClass}
            >
              {codes.map((c) => (
                <option key={c} value={c}>
                  {labels[c]}
                </option>
              ))}
            </select>
          </Field>
          <Field label={textLabel}>
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={4}
              className={inputClass}
            />
          </Field>
        </div>
        <div className="mt-6 flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            {cancelLabel}
          </Button>
          <Button variant="danger" onClick={() => onConfirm(code, text.trim())}>
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}

function StickerEditor({ sticker, onChange }: { sticker: Sticker; onChange: (s: Sticker) => void }) {
  const { t } = useI18n();
  return (
    <div className="rounded-lg border border-zinc-200 bg-white p-3">
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={sticker.thumb_url || sticker.file_url}
        alt={sticker.file}
        className="mb-2 h-24 w-24 rounded object-cover"
        loading="lazy"
      />
      <input
        value={sticker.emoji_hint}
        onChange={(e) => onChange({ ...sticker, emoji_hint: e.target.value })}
        className={`${inputClass} mb-2`}
        placeholder={t("emojiHint")}
      />
      <input
        value={sticker.keywords.join(", ")}
        onChange={(e) =>
          onChange({
            ...sticker,
            keywords: e.target.value
              .split(",")
              .map((k) => k.trim())
              .filter((k) => k.length > 0),
          })
        }
        className={inputClass}
        placeholder={t("keywords")}
      />
    </div>
  );
}

function PackDetailView({ id }: { id: string }) {
  const { request } = useApi();
  const { isAdmin } = useAuth();
  const { t } = useI18n();
  const qc = useQueryClient();

  const [dialog, setDialog] = useState<"reject" | "revoke" | null>(null);
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [tags, setTags] = useState("");
  const [stickers, setStickers] = useState<Sticker[]>([]);
  const [error, setError] = useState<string | null>(null);

  const { data, isLoading, isError, error: qError } = useQuery({
    queryKey: ["pack", id],
    queryFn: () => request<PackDetail>(`/api/v1/packs/${id}`),
  });

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["pack", id] });
    qc.invalidateQueries({ queryKey: ["my-packs"] });
    qc.invalidateQueries({ queryKey: ["admin-queue"] });
    qc.invalidateQueries({ queryKey: ["admin-published"] });
  };

  const action = useMutation({
    mutationFn: async (op: { path: string; method: string; body?: unknown }) =>
      request<void>(op.path, {
        method: op.method,
        body: op.body ? JSON.stringify(op.body) : undefined,
      }),
    onSuccess: () => {
      invalidate();
      setDialog(null);
    },
    onError: (err) => setError(err instanceof Error ? err.message : t("actionFailed")),
  });

  const saveMeta = useMutation({
    mutationFn: () =>
      request<PackDetail>(`/api/v1/packs/${id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: name.trim(),
          slug: slug.trim(),
          tags: tags
            .split(",")
            .map((t) => t.trim())
            .filter((t) => t.length > 0),
        }),
      }),
    onSuccess: () => {
      setEditing(false);
      invalidate();
    },
    onError: (err) => setError(err instanceof Error ? err.message : t("saveFailed")),
  });

  const saveStickers = useMutation({
    mutationFn: () =>
      request<PackDetail>(`/api/v1/packs/${id}/stickers`, {
        method: "PUT",
        body: JSON.stringify({
          stickers: stickers.map((s) => ({
            file: s.file,
            emoji_hint: s.emoji_hint,
            keywords: s.keywords,
          })),
        }),
      }),
    onSuccess: () => invalidate(),
    onError: (err) => setError(err instanceof Error ? err.message : t("saveFailed")),
  });

  if (isLoading) return <Spinner />;
  if (isError || !data) {
    return (
      <ErrorBanner
        message={qError instanceof Error ? qError.message : t("loadPackFailed")}
      />
    );
  }

  const pack = data;
  const canEdit = pack.status === "draft" || pack.status === "rejected";
  const canSubmit = pack.status === "draft" || pack.status === "rejected";
  const canConfirm = pack.status === "awaiting_confirm";
  const canOwnerRevoke = pack.status === "published";
  const canDelete = pack.status === "rejected";

  function startEdit() {
    setName(pack.name);
    setSlug(pack.slug);
    setTags(pack.tags.join(", "));
    setStickers(pack.stickers);
    setEditing(true);
  }

  return (
    <>
      <PageHeader
        title={pack.name}
        subtitle={`${pack.slug} · ${pack.author} · v${pack.version}`}
        actions={
          <div className="flex items-center gap-2">
            <StatusPill status={pack.status} />
            {canEdit && (
              <Button variant="secondary" onClick={startEdit}>
                {t("edit")}
              </Button>
            )}
          </div>
        }
      />

      {error && <div className="mb-4"><ErrorBanner message={error} /></div>}

      {pack.reject_reason && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          <span className="font-semibold">{t("rejectionReason")}</span> {pack.reject_reason}
        </div>
      )}

      {editing ? (
        <div className="space-y-6">
          <div className="max-w-2xl space-y-4 rounded-xl border border-zinc-200 bg-white p-6">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label={t("packName")}>
                <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
              </Field>
              <Field label={t("packSlug")}>
                <input value={slug} onChange={(e) => setSlug(e.target.value)} className={inputClass} />
              </Field>
            </div>
            <Field label={t("packTags")}>
              <input value={tags} onChange={(e) => setTags(e.target.value)} className={inputClass} />
            </Field>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={() => setEditing(false)}>
                {t("cancel")}
              </Button>
              <Button onClick={() => saveMeta.mutate()} disabled={saveMeta.isPending}>
                {saveMeta.isPending ? t("loading") : t("saveMetadata")}
              </Button>
            </div>
          </div>

          <div>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-semibold text-zinc-700">
                {t("stickerMetadata")} ({stickers.length})
              </h2>
              <Button onClick={() => saveStickers.mutate()} disabled={saveStickers.isPending}>
                {saveStickers.isPending ? t("loading") : t("saveStickers")}
              </Button>
            </div>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {stickers.map((s) => (
                <StickerEditor
                  key={s.file}
                  sticker={s}
                  onChange={(next) =>
                    setStickers((prev) => prev.map((p) => (p.file === s.file ? next : p)))
                  }
                />
              ))}
            </div>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8">
          {pack.stickers.map((s) => (
            <div key={s.file} className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={s.thumb_url || s.file_url}
                alt={s.file}
                className="aspect-square w-full object-cover"
                loading="lazy"
              />
              {s.emoji_hint && (
                <div className="truncate px-2 py-1 text-center text-xs text-zinc-500">
                  {s.emoji_hint}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      <div className="mt-8 flex flex-wrap gap-2 border-t border-zinc-200 pt-6">
        {canSubmit && (
          <Button
            onClick={() => action.mutate({ path: `/api/v1/packs/${id}/submit`, method: "POST" })}
            disabled={action.isPending}
          >
            {t("submitProcessing")}
          </Button>
        )}
        {canConfirm && (
          <Button
            onClick={() => action.mutate({ path: `/api/v1/packs/${id}/confirm`, method: "POST" })}
            disabled={action.isPending}
          >
            {t("confirmPublish")}
          </Button>
        )}
        {canOwnerRevoke && (
          <Button
            variant="danger"
            onClick={() => action.mutate({ path: `/api/v1/packs/${id}/revoke`, method: "POST" })}
            disabled={action.isPending}
          >
            {t("withdraw")}
          </Button>
        )}
        {canDelete && (
          <Button
            variant="danger"
            onClick={() => {
              if (window.confirm(t("deleteConfirm"))) {
                action.mutate({ path: `/api/v1/packs/${id}`, method: "DELETE" });
              }
            }}
            disabled={action.isPending}
          >
            {t("deletePack")}
          </Button>
        )}

        {isAdmin && pack.status === "processing" && (
          <>
            <Button
              variant="danger"
              onClick={() => setDialog("reject")}
              disabled={action.isPending}
            >
              {t("btnReject")}
            </Button>
            <Button
              onClick={() =>
                action.mutate({ path: `/api/v1/admin/packs/${id}/approve`, method: "POST" })
              }
              disabled={action.isPending}
            >
              {t("btnApprove")}
            </Button>
          </>
        )}
        {isAdmin && pack.status === "published" && (
          <Button variant="danger" onClick={() => setDialog("revoke")} disabled={action.isPending}>
            {t("btnRevoke")}
          </Button>
        )}
      </div>

      {dialog === "reject" && (
        <ReasonDialog
          title={t("rejectPack")}
          codes={REJECTION_CODES}
          labels={REJECTION_LABELS}
          defaults={REJECTION_DEFAULT_TEXT}
          reasonLabel={t("reasonCode")}
          textLabel={t("reasonText")}
          cancelLabel={t("cancel")}
          confirmLabel={t("btnReject")}
          onClose={() => setDialog(null)}
          onConfirm={(code, text) =>
            action.mutate({
              path: `/api/v1/admin/packs/${id}/reject`,
              method: "POST",
              body: { reason_code: code, rejection_code: code, reason_text: text },
            })
          }
        />
      )}
      {dialog === "revoke" && (
        <ReasonDialog
          title={t("revokePack")}
          codes={REVOCATION_CODES}
          labels={REVOCATION_LABELS}
          defaults={REVOCATION_DEFAULT_TEXT}
          reasonLabel={t("reasonCode")}
          textLabel={t("reasonText")}
          cancelLabel={t("cancel")}
          confirmLabel={t("btnRevoke")}
          onClose={() => setDialog(null)}
          onConfirm={(code, text) =>
            action.mutate({
              path: `/api/v1/admin/packs/${id}/revoke`,
              method: "POST",
              body: { reason_code: code, revocation_code: code, reason_text: text },
            })
          }
        />
      )}
    </>
  );
}

function DetailPage() {
  const { t } = useI18n();
  const params = useParams<{ id: string }>();
  const id = params?.id;

  return (
    <RequireAuth>
      <AppShell>
        {id ? (
          <PackDetailView id={id} />
        ) : (
          <EmptyState title={t("detailNoSelect")} hint={t("detailNoSelectHint")} />
        )}
      </AppShell>
    </RequireAuth>
  );
}

export default function Page() {
  return (
    <Suspense fallback={<Spinner />}>
      <DetailPage />
    </Suspense>
  );
}