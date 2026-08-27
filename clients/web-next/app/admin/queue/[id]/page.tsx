"use client";

import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useApi } from "@/lib/api";
import type { PackDetail } from "@/lib/types";
import {
  REJECTION_CODES,
  REJECTION_DEFAULT_TEXT,
  REJECTION_LABELS,
} from "@/lib/constants";
import { AppShell } from "@/components/app-shell";
import { RequireAdmin } from "@/components/guards";
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

function RejectDialog({
  onConfirm,
  onClose,
}: {
  onConfirm: (code: string, text: string) => void;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [code, setCode] = useState<string>(REJECTION_CODES[0]);
  const [text, setText] = useState(REJECTION_DEFAULT_TEXT[REJECTION_CODES[0]]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl">
        <h2 className="text-lg font-bold text-zinc-900">{t("rejectPack")}</h2>
        <div className="mt-4 space-y-4">
          <Field label={t("reasonCode")}>
            <select
              value={code}
              onChange={(e) => {
                setCode(e.target.value);
                setText(REJECTION_DEFAULT_TEXT[e.target.value]);
              }}
              className={inputClass}
            >
              {REJECTION_CODES.map((c) => (
                <option key={c} value={c}>
                  {REJECTION_LABELS[c]}
                </option>
              ))}
            </select>
          </Field>
          <Field label={t("reasonText")}>
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
            {t("cancel")}
          </Button>
          <Button variant="danger" onClick={() => onConfirm(code, text.trim())}>
            {t("btnReject")}
          </Button>
        </div>
      </div>
    </div>
  );
}

function QueueDetail({ id }: { id: string }) {
  const { request } = useApi();
  const { t } = useI18n();
  const qc = useQueryClient();
  const [showReject, setShowReject] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { data, isLoading, isError, error: qError } = useQuery({
    queryKey: ["admin-pack", id],
    queryFn: () => request<PackDetail>(`/api/v1/admin/packs/${id}`),
  });

  const action = useMutation({
    mutationFn: async (op: { path: string; method: string; body?: unknown }) =>
      request<void>(op.path, {
        method: op.method,
        body: op.body ? JSON.stringify(op.body) : undefined,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-queue"] });
      qc.invalidateQueries({ queryKey: ["admin-pack", id] });
      setShowReject(false);
    },
    onError: (err) => setError(err instanceof Error ? err.message : t("actionFailed")),
  });

  if (isLoading) return <Spinner />;
  if (isError || !data) {
    return (
      <ErrorBanner
        message={qError instanceof Error ? qError.message : t("loadPackFailed")}
      />
    );
  }

  return (
    <>
      <PageHeader
        title={data.name}
        subtitle={`${data.slug} · ${data.author} · v${data.version}`}
        actions={<StatusPill status={data.status} />}
      />

      {error && <div className="mb-4"><ErrorBanner message={error} /></div>}
      {data.reject_reason && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          <span className="font-semibold">{t("rejectionReason")}</span> {data.reject_reason}
        </div>
      )}

      <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8">
        {data.stickers.map((s) => (
          <div key={s.file} className="overflow-hidden rounded-lg border border-zinc-200 bg-white">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={s.thumb_url || s.file_url}
              alt={s.file}
              className="aspect-square w-full object-cover"
              loading="lazy"
            />
          </div>
        ))}
      </div>

      <div className="mt-8 flex gap-2 border-t border-zinc-200 pt-6">
        <Button
          variant="danger"
          onClick={() => setShowReject(true)}
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
          {t("approvePublish")}
        </Button>
      </div>

      {showReject && (
        <RejectDialog
          onClose={() => setShowReject(false)}
          onConfirm={(code, text) =>
            action.mutate({
              path: `/api/v1/admin/packs/${id}/reject`,
              method: "POST",
              body: { reason_code: code, rejection_code: code, reason_text: text },
            })
          }
        />
      )}
    </>
  );
}

export default function Page() {
  const { t } = useI18n();
  const params = useParams<{ id: string }>();
  const id = params?.id;

  return (
    <RequireAdmin>
      <AppShell>
        {id ? (
          <QueueDetail id={id} />
        ) : (
          <EmptyState title={t("detailNoSelect")} hint={t("queueNoSelectHint")} />
        )}
      </AppShell>
    </RequireAdmin>
  );
}