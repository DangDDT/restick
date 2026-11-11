"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { KeyPackAccess } from "@/components/key-pack-access";
import {
  Button,
  ConfirmDialog,
  CopyButton,
  Dialog,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  SkeletonRows,
  StatusPill,
  SuccessBanner,
  WarnBanner,
  cx,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type { CreatedKey, Key, KeyList } from "@/lib/types";

type Pending = { kind: "rotate" | "disable"; key: Key };

function Keys() {
  const { t } = useI18n();
  const qc = useQueryClient();
  const nameInput = useRef<HTMLInputElement>(null);
  const rawSelect = useRef<HTMLTextAreaElement>(null);
  const [name, setName] = useState("");
  const [expiry, setExpiry] = useState("");
  const [nameError, setNameError] = useState<string | null>(null);
  const [expiryError, setExpiryError] = useState<string | null>(null);
  // The raw secret lives only in this component's state: never in the URL or storage.
  const [created, setCreated] = useState<CreatedKey | null>(null);
  const [accessKey, setAccessKey] = useState<Key | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<Pending | null>(null);
  const [pendingBusy, setPendingBusy] = useState(false);
  const [pendingError, setPendingError] = useState<string | null>(null);

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
    setNameError(null);
    setExpiryError(null);
    if (!name.trim()) return setNameError(t("keyNameRequired"));

    let expiresAt: string | null = null;
    if (expiry) {
      const when = new Date(expiry);
      if (Number.isNaN(when.getTime()) || when.getTime() <= Date.now()) {
        return setExpiryError(t("keyExpiryPast"));
      }
      expiresAt = `${when.toISOString().slice(0, 19)}Z`;
    }

    setBusy(true);
    try {
      const created = await api<CreatedKey>("/api/v1/admin/api-keys", {
        method: "POST",
        json: { name: name.trim(), expires_at: expiresAt },
      });
      setCreated(created);
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

  async function confirmPending() {
    if (!pending) return;
    setPendingBusy(true);
    setPendingError(null);
    try {
      if (pending.kind === "rotate") {
        // Close the confirmation before the secret dialog opens so the two never stack.
        const created = await api<CreatedKey>(`/api/v1/admin/api-keys/${pending.key.id}/rotate`, { method: "POST" });
        setPending(null);
        setCreated(created);
        setMessage(t("keyRotatedDone"));
      } else {
        await api<void>(`/api/v1/admin/api-keys/${pending.key.id}`, { method: "DELETE" });
        setPending(null);
        setMessage(t("keyDisabledDone"));
      }
      setError(null);
      refresh();
    } catch (err) {
      setPendingError(errorText(err));
    } finally {
      setPendingBusy(false);
    }
  }

  function closeSecret() {
    setCreated(null);
  }

  const list = keys.data?.items ?? [];

  return (
    <>
      <PageHeader title={t("keysTitle")} subtitle={t("keysSubtitle")} />

      {message && (
        <div className="mb-5">
          <SuccessBanner message={message} onDismiss={() => setMessage(null)} />
        </div>
      )}
      {error && (
        <div className="mb-5">
          <ErrorBanner message={error} onDismiss={() => setError(null)} />
        </div>
      )}

      <form
        onSubmit={createKey}
        className="mb-6 grid gap-3 border-y border-line py-4 sm:grid-cols-[1fr_1fr_auto] sm:items-end"
        aria-busy={busy || undefined}
      >
        <Field label={t("keyName")} error={nameError ?? undefined}>
          <input
            ref={nameInput}
            value={name}
            onChange={(event) => {
              setName(event.target.value);
              setNameError(null);
            }}
            className={inputClass}
            aria-invalid={Boolean(nameError) || undefined}
          />
        </Field>
        <Field label={t("keyExpiry")} hint={expiryError ? undefined : t("keyExpiryHint")} error={expiryError ?? undefined}>
          <input
            type="datetime-local"
            value={expiry}
            onChange={(event) => {
              setExpiry(event.target.value);
              setExpiryError(null);
            }}
            className={inputClass}
            aria-invalid={Boolean(expiryError) || undefined}
          />
        </Field>
        <Button type="submit" disabled={busy} loading={busy} icon={busy ? undefined : "key"}>
          {busy ? t("saving") : t("newKey")}
        </Button>
      </form>

      {keys.isLoading ? (
        <SkeletonRows rows={4} />
      ) : keys.isError ? (
        <div className="space-y-3">
          <ErrorBanner message={errorText(keys.error)} />
          <Button variant="secondary" icon="refresh" onClick={() => void keys.refetch()}>
            {t("retry")}
          </Button>
        </div>
      ) : list.length > 0 ? (
        <>
          {/* Desktop: a ruled table. */}
          <div className="relative hidden overflow-x-auto border-y border-line md:block">
            <table className="w-full min-w-[46rem] text-left text-sm">
              <thead className="border-b border-line text-xs font-medium text-ink-3">
                <tr>
                  <th scope="col" className="px-3 py-2.5 font-medium">{t("keyName")}</th>
                  <th scope="col" className="px-3 py-2.5 font-medium">{t("keyPrefix")}</th>
                  <th scope="col" className="px-3 py-2.5 font-medium">{t("keyStatus")}</th>
                  <th scope="col" className="px-3 py-2.5 font-medium">{t("keyCreated")}</th>
                  <th scope="col" className="px-3 py-2.5 font-medium">{t("keyExpires")}</th>
                  <th scope="col" className="px-3 py-2.5 font-medium">{t("disabledAtLabel")}</th>
                  <th scope="col" className="px-3 py-2.5 font-medium">{t("packAccess")}</th>
                  <th scope="col" className="px-3 py-2.5 font-medium">
                    <span className="sr-only">{t("actions")}</span>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {list.map((key) => (
                  <tr key={key.id} className="align-middle">
                    <td className="px-3 py-3 font-medium text-ink">{key.name}</td>
                    <td className="mono px-3 py-3 text-xs text-ink-2">{key.prefix}</td>
                    <td className="px-3 py-3">
                      <StatusPill status={key.active ? "active" : "inactive"} />
                    </td>
                    <td className="num px-3 py-3 text-xs text-ink-3">{formatDate(key.created_at)}</td>
                    <td className="num px-3 py-3 text-xs text-ink-3">{formatDate(key.expires_at)}</td>
                    <td className="num px-3 py-3 text-xs text-ink-3">{formatDate(key.disabled_at)}</td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-2">
                        <span data-qa="key-pack-count" className="num text-xs text-ink-3">
                          {key.pack_count}
                        </span>
                        <Button variant="secondary" size="sm" icon="key" onClick={() => setAccessKey(key)}>
                          {t("packAccess")}
                        </Button>
                      </div>
                    </td>
                    <td className="px-3 py-3">
                      <KeyActions keyItem={key} onAction={setPending} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Phone: stacked rows, same data and actions. */}
          <ul className="divide-y divide-line border-y border-line md:hidden">
            {list.map((key) => (
              <li key={key.id} className="space-y-2.5 py-3.5">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="min-w-0 flex-1 truncate font-medium text-ink">{key.name}</p>
                  <StatusPill status={key.active ? "active" : "inactive"} />
                </div>
                <p className="mono text-xs text-ink-2">{key.prefix}</p>
                <dl className="num grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs text-ink-3">
                  <dt>{t("keyCreated")}</dt>
                  <dd>{formatDate(key.created_at)}</dd>
                  <dt>{t("keyExpires")}</dt>
                  <dd>{formatDate(key.expires_at)}</dd>
                  <dt>{t("disabledAtLabel")}</dt>
                  <dd>{formatDate(key.disabled_at)}</dd>
                  <dt>{t("packAccess")}</dt>
                  <dd data-qa="key-pack-count">{key.pack_count}</dd>
                </dl>
                <div className="flex flex-wrap gap-2">
                  <Button variant="secondary" size="sm" icon="key" onClick={() => setAccessKey(key)}>
                    {t("packAccess")}
                  </Button>
                  <KeyActions keyItem={key} onAction={setPending} />
                </div>
              </li>
            ))}
          </ul>
        </>
      ) : (
        <EmptyState title={t("noKeys")} icon="key">
          <Button variant="secondary" icon="plus" onClick={() => nameInput.current?.focus()}>
            {t("newKey")}
          </Button>
        </EmptyState>
      )}

      {created && (
        <Dialog
          title={t("keyRawTitle")}
          onClose={closeSecret}
          footer={
            <Button variant="secondary" onClick={closeSecret}>
              {t("close")}
            </Button>
          }
        >
          <div className="space-y-4">
            <WarnBanner message={t("keyRawHint")} />
            <textarea
              ref={rawSelect}
              readOnly
              rows={2}
              value={created.raw}
              onFocus={(event) => event.currentTarget.select()}
              className={cx(inputClass, "mono resize-none break-all text-[13px] leading-5")}
              aria-label={t("keyRawTitle")}
            />
            <CopyButton text={created.raw} selectRef={rawSelect} variant="primary" />
            <div className="border-t border-line pt-4">
              <h3 className="mb-2 text-sm font-semibold text-ink">{t("packAccessTitle")}</h3>
              <KeyPackAccess keyId={created.id} />
            </div>
          </div>
        </Dialog>
      )}

      {accessKey && (
        <Dialog
          title={`${accessKey.name} · ${t("packAccessTitle")}`}
          onClose={() => setAccessKey(null)}
          footer={
            <Button variant="secondary" onClick={() => setAccessKey(null)}>
              {t("close")}
            </Button>
          }
        >
          <KeyPackAccess keyId={accessKey.id} />
        </Dialog>
      )}

      {pending && (
        <ConfirmDialog
          title={t(pending.kind === "rotate" ? "keyRotateTitle" : "keyDisableTitle")}
          body={
            <>
              <p>{t(pending.kind === "rotate" ? "rotateConfirm" : "disableConfirm")}</p>
              <p className="mono mt-2 text-xs text-ink-3">
                {pending.key.name} · {pending.key.prefix}
              </p>
            </>
          }
          confirmLabel={t(pending.kind === "rotate" ? "rotate" : "disableKey")}
          onConfirm={() => void confirmPending()}
          onClose={() => {
            if (pendingBusy) return;
            setPending(null);
            setPendingError(null);
          }}
          busy={pendingBusy}
          error={pendingError}
        />
      )}
    </>
  );
}

function KeyActions({ keyItem, onAction }: { keyItem: Key; onAction: (pending: Pending) => void }) {
  const { t } = useI18n();
  return (
    <div className="flex flex-wrap gap-2">
      <Button variant="secondary" size="sm" icon="refresh" onClick={() => onAction({ kind: "rotate", key: keyItem })}>
        {t("rotate")}
      </Button>
      {keyItem.active && (
        <Button variant="danger" size="sm" icon="lock" onClick={() => onAction({ kind: "disable", key: keyItem })}>
          {t("disableKey")}
        </Button>
      )}
    </div>
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
