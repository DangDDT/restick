"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useApi } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { APIKey, APIKeyListResponse, CreatedAPIKey } from "@/lib/types";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { useI18n } from "@/lib/i18n";
import {
  Button,
  EmptyState,
  ErrorBanner,
  Field,
  inputClass,
  PageHeader,
  Spinner,
} from "@/components/ui";

function OneTimeKeyDialog({ rawKey, onClose }: { rawKey: string; onClose: () => void }) {
  const { t } = useI18n();
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl">
        <h2 className="text-lg font-bold text-zinc-900">{t("keyCreatedTitle")}</h2>
        <p className="mt-1 text-sm text-zinc-500">
          {t("keyCreatedBody")}
        </p>
        <div className="mt-4 rounded-lg bg-zinc-900 p-4">
          <code className="break-all text-sm text-emerald-300">{rawKey}</code>
        </div>
        <div className="mt-6 flex justify-end">
          <Button onClick={onClose}>{t("done")}</Button>
        </div>
      </div>
    </div>
  );
}

function CreateDialog({
  onConfirm,
  onClose,
}: {
  onConfirm: (name: string, days: number | null) => void;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [name, setName] = useState(t("keyNameDefault"));
  const [days, setDays] = useState("");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <h2 className="text-lg font-bold text-zinc-900">{t("createKeyTitle")}</h2>
        <div className="mt-4 space-y-4">
          <Field label={t("keyName")}>
            <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
          </Field>
          <Field label={t("expiresDays")} hint={t("expiresHint")}>
            <input
              value={days}
              onChange={(e) => setDays(e.target.value)}
              className={inputClass}
              placeholder="30, 90"
            />
          </Field>
        </div>
        <div className="mt-6 flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            {t("cancel")}
          </Button>
          <Button
            onClick={() => {
              const parsed = days.trim() ? Number(days.trim()) : null;
              onConfirm(name.trim() || "API Key", parsed && parsed > 0 ? parsed : null);
            }}
          >
            {t("create")}
          </Button>
        </div>
      </div>
    </div>
  );
}

function KeyTable({
  keys,
  admin,
  onRotate,
  onDelete,
}: {
  keys: APIKey[];
  admin: boolean;
  onRotate: (id: string) => void;
  onDelete: (key: { id: string; admin: boolean }) => void;
}) {
  const { t } = useI18n();
  if (keys.length === 0) {
    return <EmptyState title={t("noKeys")} hint={t("noKeysHint")} />;
  }
  return (
    <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white">
      <table className="w-full text-left text-sm">
        <thead className="border-b border-zinc-200 bg-zinc-50 text-xs uppercase tracking-wider text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">{t("keyName")}</th>
            <th className="px-4 py-3 font-semibold">{t("prefix")}</th>
            {admin && <th className="px-4 py-3 font-semibold">{t("owner")}</th>}
            <th className="px-4 py-3 font-semibold">{t("status")}</th>
            <th className="px-4 py-3 font-semibold">{t("created")}</th>
            <th className="px-4 py-3 font-semibold">{t("actions")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-100">
          {keys.map((k) => (
            <tr key={k.id} className="hover:bg-zinc-50">
              <td className="px-4 py-3 font-medium text-zinc-900">{k.name}</td>
              <td className="px-4 py-3 font-mono text-xs text-zinc-600">{k.key_prefix}</td>
              {admin && <td className="px-4 py-3 text-zinc-600">{k.user_email ?? "—"}</td>}
              <td className="px-4 py-3">
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-semibold ${
                    k.is_active
                      ? "bg-emerald-100 text-emerald-700"
                      : "bg-zinc-100 text-zinc-500"
                  }`}
                >
                  {k.is_active ? t("active") : t("inactive")}
                </span>
              </td>
              <td className="px-4 py-3 text-xs text-zinc-500">
                {new Date(k.created_at).toLocaleDateString()}
              </td>
              <td className="px-4 py-3">
                <div className="flex gap-2">
                  {!admin && (
                    <button
                      onClick={() => onRotate(k.id)}
                      className="text-xs font-medium text-indigo-600 hover:underline"
                    >
                      {t("rotate")}
                    </button>
                  )}
                  <button
                    onClick={() => {
                      if (window.confirm(t("deleteKeyConfirm"))) {
                        onDelete({ id: k.id, admin });
                      }
                    }}
                    className="text-xs font-medium text-red-600 hover:underline"
                  >
                    {t("delete")}
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ApiKeys() {
  const { request } = useApi();
  const { t } = useI18n();
  const { isAdmin } = useAuth();
  const qc = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [rawKey, setRawKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const { data, isLoading, isError, error: qError } = useQuery({
    queryKey: ["api-keys"],
    queryFn: async () => {
      const mine = await request<APIKeyListResponse>("/api/v1/me/api-keys");
      let all: APIKey[] = [];
      if (isAdmin) {
        const res = await request<APIKeyListResponse>("/api/v1/admin/api-keys");
        all = res.items;
      }
      return { mine: mine.items, all };
    },
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["api-keys"] });

  const createKey = useMutation({
    mutationFn: (body: { name: string; expires_in_days?: number }) =>
      request<CreatedAPIKey>("/api/v1/me/api-keys", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    onSuccess: (created) => {
      invalidate();
      setShowCreate(false);
      if (created.raw_key) setRawKey(created.raw_key);
    },
    onError: (err) => setError(err instanceof Error ? err.message : t("keyCreateFailed")),
  });

  const rotateKey = useMutation({
    mutationFn: (id: string) =>
      request<CreatedAPIKey>(`/api/v1/me/api-keys/${id}/rotate`, { method: "POST" }),
    onSuccess: (rotated) => {
      invalidate();
      if (rotated.raw_key) setRawKey(rotated.raw_key);
    },
    onError: (err) => setError(err instanceof Error ? err.message : t("keyRotateFailed")),
  });

  const deleteKey = useMutation({
    mutationFn: (key: { id: string; admin: boolean }) =>
      request<void>(
        key.admin ? `/api/v1/admin/api-keys/${key.id}` : `/api/v1/me/api-keys/${key.id}`,
        { method: "DELETE" },
      ),
    onSuccess: invalidate,
    onError: (err) => setError(err instanceof Error ? err.message : t("keyDeleteFailed")),
  });

  return (
    <>
      <PageHeader
        title={t("keysTitle")}
        subtitle={t("keysSubtitle")}
        actions={
          <Button onClick={() => setShowCreate(true)}>{t("createKey")}</Button>
        }
      />

      {error && <div className="mb-4"><ErrorBanner message={error} /></div>}
      {isError && (
        <div className="mb-4">
          <ErrorBanner message={qError instanceof Error ? qError.message : t("loadKeysFailed")} />
        </div>
      )}

      {isLoading ? (
        <Spinner />
      ) : (
        <div className="space-y-8">
          <section>
            <h2 className="mb-3 text-sm font-semibold text-zinc-700">{t("personalKeys")}</h2>
            <KeyTable
              keys={data?.mine ?? []}
              admin={false}
              onRotate={(id) => rotateKey.mutate(id)}
              onDelete={(key) => deleteKey.mutate(key)}
            />
          </section>
          {isAdmin && (
            <section>
              <h2 className="mb-3 text-sm font-semibold text-zinc-700">{t("systemKeys")}</h2>
              <KeyTable
                keys={data?.all ?? []}
                admin
                onRotate={() => {}}
                onDelete={(key) => deleteKey.mutate(key)}
              />
            </section>
          )}
        </div>
      )}

      {showCreate && (
        <CreateDialog
          onClose={() => setShowCreate(false)}
          onConfirm={(name, days) =>
            createKey.mutate(
              days ? { name, expires_in_days: days } : { name },
            )
          }
        />
      )}
      {rawKey && <OneTimeKeyDialog rawKey={rawKey} onClose={() => setRawKey(null)} />}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <ApiKeys />
      </AppShell>
    </RequireAuth>
  );
}
