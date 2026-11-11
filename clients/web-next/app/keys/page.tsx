"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  CopyButton,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  Spinner,
  StatusPill,
  SuccessBanner,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type { CreatedKey, Key, KeyList } from "@/lib/types";

function Keys() {
  const { t } = useI18n();
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [expiry, setExpiry] = useState("");
  const [raw, setRaw] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const keys = useQuery({
    queryKey: ["admin-keys"],
    queryFn: () => api<KeyList>("/api/v1/admin/api-keys"),
  });

  function refresh() {
    qc.invalidateQueries({ queryKey: ["admin-keys"] });
  }

  async function createKey(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setMessage(null);
    if (!name.trim()) return setError(t("keyNameRequired"));

    let expiresAt: string | null = null;
    if (expiry) {
      const when = new Date(expiry);
      if (Number.isNaN(when.getTime()) || when.getTime() <= Date.now()) {
        return setError(t("keyExpiryPast"));
      }
      expiresAt = `${when.toISOString().slice(0, 19)}Z`;
    }

    setBusy(true);
    try {
      const created = await api<CreatedKey>("/api/v1/admin/api-keys", {
        method: "POST",
        json: { name: name.trim(), expires_at: expiresAt },
      });
      setRaw(created.raw);
      setMessage(t("keyCreatedDone"));
      setName("");
      setExpiry("");
      refresh();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  async function rotate(key: Key) {
    if (!window.confirm(t("rotateConfirm"))) return;
    setError(null);
    setMessage(null);
    try {
      const created = await api<CreatedKey>(`/api/v1/admin/api-keys/${key.id}/rotate`, {
        method: "POST",
      });
      setRaw(created.raw);
      setMessage(t("keyRotatedDone"));
      refresh();
    } catch (err) {
      setError(errorText(err));
    }
  }

  async function disable(key: Key) {
    if (!window.confirm(t("disableConfirm"))) return;
    setError(null);
    setMessage(null);
    try {
      await api<void>(`/api/v1/admin/api-keys/${key.id}`, { method: "DELETE" });
      setMessage(t("keyDisabledDone"));
      refresh();
    } catch (err) {
      setError(errorText(err));
    }
  }

  return (
    <>
      <PageHeader title={t("keysTitle")} subtitle={t("keysSubtitle")} />

      {message && (
        <div className="mb-4">
          <SuccessBanner message={message} />
        </div>
      )}
      {error && (
        <div className="mb-4">
          <ErrorBanner message={error} />
        </div>
      )}

      {raw && (
        <div className="mb-5 rounded-xl border border-emerald-200 bg-emerald-50 p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-emerald-900">{t("keyRawTitle")}</p>
              <p className="text-xs text-emerald-800">{t("keyRawHint")}</p>
            </div>
            <div className="flex gap-2">
              <CopyButton text={raw} />
              <Button variant="secondary" onClick={() => setRaw(null)}>
                {t("cancel")}
              </Button>
            </div>
          </div>
          <code className="mt-3 block select-all break-all rounded-lg bg-zinc-900 p-3 text-sm text-emerald-300">
            {raw}
          </code>
        </div>
      )}

      <form
        onSubmit={createKey}
        className="mb-6 grid gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-[1fr_1fr_auto]"
      >
        <Field label={t("keyName")}>
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            className={inputClass}
          />
        </Field>
        <Field label={t("keyExpiry")} hint={t("keyExpiryHint")}>
          <input
            type="datetime-local"
            value={expiry}
            onChange={(event) => setExpiry(event.target.value)}
            className={inputClass}
          />
        </Field>
        <div className="flex items-end">
          <Button type="submit" disabled={busy}>
            {busy ? t("saving") : t("newKey")}
          </Button>
        </div>
      </form>

      {keys.isLoading ? (
        <Spinner />
      ) : keys.isError ? (
        <ErrorBanner message={errorText(keys.error)} />
      ) : keys.data && keys.data.items.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-zinc-200 bg-white">
          <table className="w-full min-w-[46rem] text-left text-sm">
            <thead className="border-b border-zinc-200 bg-zinc-50 text-xs uppercase tracking-wider text-zinc-500">
              <tr>
                <th className="px-4 py-3 font-semibold">{t("keyName")}</th>
                <th className="px-4 py-3 font-semibold">{t("keyPrefix")}</th>
                <th className="px-4 py-3 font-semibold">{t("keyStatus")}</th>
                <th className="px-4 py-3 font-semibold">{t("keyCreated")}</th>
                <th className="px-4 py-3 font-semibold">{t("keyExpires")}</th>
                <th className="px-4 py-3 font-semibold">{t("disabledAtLabel")}</th>
                <th className="px-4 py-3 font-semibold">{t("actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {keys.data.items.map((key) => (
                <tr key={key.id} className="align-top">
                  <td className="px-4 py-3 font-medium text-zinc-900">{key.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-zinc-600">{key.prefix}</td>
                  <td className="px-4 py-3">
                    <StatusPill status={key.active ? "active" : "inactive"} />
                  </td>
                  <td className="px-4 py-3 text-xs text-zinc-500">{formatDate(key.created_at)}</td>
                  <td className="px-4 py-3 text-xs text-zinc-500">{formatDate(key.expires_at)}</td>
                  <td className="px-4 py-3 text-xs text-zinc-500">{formatDate(key.disabled_at)}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-2">
                      <button
                        onClick={() => void rotate(key)}
                        className="text-xs font-medium text-indigo-600 hover:underline"
                      >
                        {t("rotate")}
                      </button>
                      {key.active && (
                        <button
                          onClick={() => void disable(key)}
                          className="text-xs font-medium text-red-600 hover:underline"
                        >
                          {t("disableKey")}
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState title={t("noKeys")} />
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <Keys />
      </AppShell>
    </RequireAuth>
  );
}
