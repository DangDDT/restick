"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  Pager,
  Spinner,
  StatusPill,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type { MessageKey } from "@/lib/i18n";
import type { AdminPack, AdminPackPage } from "@/lib/types";

const PAGE_SIZE = 20;
const STATUSES = ["", "draft", "ready", "revoked"] as const;
const STATUS_LABEL: Record<"draft" | "ready" | "revoked", MessageKey> = {
  draft: "statusDraft",
  ready: "statusReady",
  revoked: "statusRevoked",
};

function PackRow({ pack }: { pack: AdminPack }) {
  const { t } = useI18n();
  return (
    <li className="flex flex-wrap items-center gap-4 px-4 py-3">
      <div className="h-16 w-16 shrink-0 overflow-hidden rounded-lg border border-zinc-200 bg-zinc-50">
        {pack.thumbnail_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={pack.thumbnail_url}
            alt={pack.name}
            className="h-full w-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-xl text-zinc-300">▦</div>
        )}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="truncate text-sm font-semibold text-zinc-900">{pack.name}</p>
          <StatusPill status={pack.status} />
        </div>
        <p className="mt-0.5 truncate text-xs text-zinc-500">
          {pack.slug} · {pack.author} · {t("versionLabel")} {pack.current_version} ·{" "}
          {pack.sticker_count} {t("stickerCount")} · {t("updatedAt")} {formatDate(pack.updated_at)}
        </p>
        {(pack.tags ?? []).length > 0 && (
          <div className="mt-1 flex flex-wrap gap-1">
            {(pack.tags ?? []).map((tag) => (
              <span
                key={tag}
                className="rounded bg-zinc-100 px-1.5 py-0.5 text-[10px] font-medium text-zinc-600"
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>
      <Link
        href={`/packs/detail?id=${encodeURIComponent(pack.id)}`}
        className="rounded-lg border border-zinc-300 bg-white px-4 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-50"
      >
        {t("open")}
      </Link>
    </li>
  );
}

function Packs() {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [tag, setTag] = useState("");
  const [applied, setApplied] = useState({ query: "", status: "", tag: "", page: 1 });

  const params = new URLSearchParams();
  if (applied.query) params.set("q", applied.query);
  if (applied.status) params.set("status", applied.status);
  if (applied.tag) params.set("tag", applied.tag);
  params.set("page", String(applied.page));
  params.set("page_size", String(PAGE_SIZE));

  const packs = useQuery({
    queryKey: ["admin-packs", applied],
    queryFn: () => api<AdminPackPage>(`/api/v1/admin/packs?${params.toString()}`),
  });

  return (
    <>
      <PageHeader
        title={t("packsTitle")}
        subtitle={t("packsSubtitle")}
        actions={
          <Link
            href="/packs/new"
            className="inline-flex items-center rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            {t("newPack")}
          </Link>
        }
      />

      <form
        onSubmit={(event) => {
          event.preventDefault();
          setApplied({ query: query.trim(), status, tag: tag.trim(), page: 1 });
        }}
        className="mb-5 grid gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-4"
      >
        <Field label={t("search")}>
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("searchPlaceholder")}
            className={inputClass}
          />
        </Field>
        <Field label={t("filterStatus")}>
          <select
            value={status}
            onChange={(event) => setStatus(event.target.value)}
            className={inputClass}
          >
            {STATUSES.map((value) => (
              <option key={value || "all"} value={value}>
                {value ? t(STATUS_LABEL[value]) : t("allStatuses")}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("filterTag")}>
          <input
            value={tag}
            onChange={(event) => setTag(event.target.value)}
            className={inputClass}
          />
        </Field>
        <div className="flex items-end">
          <Button type="submit">{t("search")}</Button>
        </div>
      </form>

      {packs.isLoading ? (
        <Spinner />
      ) : packs.isError ? (
        <ErrorBanner message={errorText(packs.error)} />
      ) : packs.data && packs.data.items.length > 0 ? (
        <>
          <ul className="divide-y divide-zinc-100 overflow-hidden rounded-xl border border-zinc-200 bg-white">
            {packs.data.items.map((pack) => (
              <PackRow key={pack.id} pack={pack} />
            ))}
          </ul>
          <Pager
            page={packs.data.page}
            pageSize={packs.data.page_size}
            total={packs.data.total}
            onChange={(page) => setApplied({ ...applied, page })}
          />
        </>
      ) : (
        <EmptyState title={t("noPacks")} />
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <Packs />
      </AppShell>
    </RequireAuth>
  );
}
