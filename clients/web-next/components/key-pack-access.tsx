"use client";

// Per-key pack grants. A key with no grants reads nothing, so this checklist is
// the only way its client sees packs. Every toggle saves the full set.

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { api, errorText } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import type { AdminPack, AdminPackPage, KeyPacks } from "@/lib/types";
import { Button, ErrorBanner, Skeleton, StatusPill, checkClass, cx, inputClass } from "./ui";

async function fetchAllPacks(): Promise<AdminPack[]> {
  const items: AdminPack[] = [];
  for (let page = 1; ; page++) {
    const res = await api<AdminPackPage>(`/api/v1/admin/packs?page=${page}&page_size=100`);
    items.push(...res.items);
    if (res.items.length === 0 || items.length >= res.total) return items;
  }
}

export function KeyPackAccess({ keyId }: { keyId: string }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const packs = useQuery({ queryKey: ["packs-all"], queryFn: fetchAllPacks });
  const grants = useQuery({
    queryKey: ["key-packs", keyId],
    queryFn: () => api<KeyPacks>(`/api/v1/admin/api-keys/${encodeURIComponent(keyId)}/packs`),
  });
  // Optimistic copy while saving; null follows the server query.
  const [draft, setDraft] = useState<string[] | null>(null);
  const [query, setQuery] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const selected = draft ?? grants.data?.pack_ids ?? [];
  const needle = query.trim().toLowerCase();
  const visible = (packs.data ?? []).filter(
    (pack) => !needle || `${pack.name} ${pack.slug}`.toLowerCase().includes(needle),
  );

  async function save(next: string[]) {
    const previous = selected;
    setDraft(next);
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const result = await api<KeyPacks>(`/api/v1/admin/api-keys/${encodeURIComponent(keyId)}/packs`, {
        method: "PUT",
        json: { pack_ids: next },
      });
      setDraft(result.pack_ids);
      setSaved(true);
      qc.invalidateQueries({ queryKey: ["key-packs", keyId] });
      // The key list shows pack_count, which this save changes.
      qc.invalidateQueries({ queryKey: ["admin-keys"] });
    } catch (err) {
      setDraft(previous);
      setError(errorText(err));
    } finally {
      setSaving(false);
    }
  }

  // Saving is serialized: toggles stay disabled until the last PUT returns.
  function toggle(packId: string, granted: boolean) {
    if (saving) return;
    void save(granted ? [...selected, packId] : selected.filter((id) => id !== packId));
  }

  const visibleIds = visible.map((pack) => pack.id);

  if (packs.isLoading || grants.isLoading) return <Skeleton className="h-40 w-full" />;
  if (packs.isError || grants.isError) {
    return (
      <div className="flex flex-wrap items-center gap-3">
        <ErrorBanner message={errorText(packs.error ?? grants.error)} />
        <Button
          variant="secondary"
          size="sm"
          icon="refresh"
          onClick={() => {
            void packs.refetch();
            void grants.refetch();
          }}
        >
          {t("retry")}
        </Button>
      </div>
    );
  }

  return (
    <div data-qa="key-pack-access" className="space-y-3" aria-busy={saving || undefined}>
      <p className="text-xs text-ink-2">{t("packAccessHint")}</p>
      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      <div className="flex flex-wrap items-center gap-2">
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("packAccessSearch")}
          aria-label={t("packAccessSearch")}
          className={cx(inputClass, "sm:max-w-xs")}
        />
        <Button
          variant="secondary"
          size="sm"
          disabled={saving || visibleIds.length === 0}
          onClick={() => void save([...selected, ...visibleIds.filter((id) => !selected.includes(id))])}
        >
          {t("packAccessSelectAll")}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          disabled={saving || visibleIds.length === 0 || !visibleIds.some((id) => selected.includes(id))}
          onClick={() => void save(selected.filter((id) => !visibleIds.includes(id)))}
        >
          {t("packAccessClear")}
        </Button>
      </div>

      <p className="num text-xs text-ink-2">
        {t("packCount")}: {selected.length}
      </p>
      <p role="status" aria-live="polite" className="text-xs text-ok">
        {saving ? t("saving") : saved ? t("packAccessSaved") : ""}
      </p>

      {(packs.data ?? []).length === 0 ? (
        <p className="text-sm text-ink-2">{t("packAccessEmpty")}</p>
      ) : visible.length === 0 ? (
        <p className="text-sm text-ink-2">{t("packAccessFilteredEmpty")}</p>
      ) : (
        <ul className="divide-y divide-line border-y border-line">
          {visible.map((pack) => {
            const checked = selected.includes(pack.id);
            return (
              <li key={pack.id}>
                <label className="flex min-h-11 cursor-pointer items-center gap-2.5 py-2">
                  <input
                    type="checkbox"
                    data-qa="key-pack-option"
                    data-pack-id={pack.id}
                    className={checkClass}
                    checked={checked}
                    disabled={saving}
                    onChange={(event) => toggle(pack.id, event.target.checked)}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium text-ink">{pack.name}</span>
                    <span className="mono block truncate text-xs text-ink-3">{pack.slug}</span>
                  </span>
                  <StatusPill status={pack.status} />
                </label>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
