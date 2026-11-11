"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { Icon } from "@/components/icons";
import {
  ArtFrame,
  Button,
  EmptyState,
  ErrorBanner,
  Field,
  JobTag,
  PageHeader,
  Pager,
  SkeletonRows,
  StatusPill,
  cx,
  inputClass,
  selectClass,
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
  const detailHref = `/packs/detail?id=${encodeURIComponent(pack.id)}`;
  return (
    <li className="relative flex items-center gap-4 py-3 pr-1">
      <ArtFrame className="size-16 shrink-0 rounded-[4px]" checker>
        {pack.thumbnail_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={pack.thumbnail_url}
            alt=""
            loading="lazy"
            className="size-full object-contain"
          />
        ) : (
          <Icon name="pack" size={20} className="text-ink-3" />
        )}
      </ArtFrame>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1">
          <Link
            href={detailHref}
            className="min-w-0 truncate font-display text-[0.9375rem] font-semibold text-ink after:absolute after:inset-0 after:content-[''] hover:underline"
          >
            {pack.name}
          </Link>
          <StatusPill status={pack.status} />
          <JobTag id={pack.id} />
        </div>
        <p className="mono mt-0.5 truncate text-xs text-ink-3">
          {pack.slug} · v{pack.current_version} · {formatDate(pack.updated_at)}
        </p>
        <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-ink-2">
          <span className="num">
            {pack.sticker_count} {t("stickerCount")}
          </span>
          <span className="truncate">{pack.author}</span>
          {(pack.tags ?? []).map((tag) => (
            <span key={tag} className="rounded-[3px] border border-line-strong px-1.5 py-px text-[11px] text-ink-2">
              {tag}
            </span>
          ))}
        </div>
      </div>
      <Icon name="chevronRight" size={16} className="shrink-0 text-ink-3" aria-hidden="true" />
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
          <Button href="/packs/new" icon="plus">
            {t("newPack")}
          </Button>
        }
      />

      <form
        onSubmit={(event) => {
          event.preventDefault();
          setApplied({ query: query.trim(), status, tag: tag.trim(), page: 1 });
        }}
        className="mb-5 grid gap-3 border-y border-line py-4 sm:grid-cols-2 lg:grid-cols-[1fr_11rem_11rem_auto]"
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
          <select value={status} onChange={(event) => setStatus(event.target.value)} className={selectClass}>
            {STATUSES.map((value) => (
              <option key={value || "all"} value={value}>
                {value ? t(STATUS_LABEL[value]) : t("allStatuses")}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("filterTag")}>
          <input value={tag} onChange={(event) => setTag(event.target.value)} className={inputClass} />
        </Field>
        <div className="flex items-end">
          <Button type="submit" icon="search" className="w-full lg:w-auto">
            {t("search")}
          </Button>
        </div>
      </form>

      {packs.isLoading ? (
        <SkeletonRows rows={5} className="border-y border-line py-3" />
      ) : packs.isError ? (
        <div className="space-y-3">
          <ErrorBanner message={errorText(packs.error)} />
          <Button variant="secondary" icon="refresh" onClick={() => void packs.refetch()}>
            {t("retry")}
          </Button>
        </div>
      ) : packs.data && packs.data.items.length > 0 ? (
        <>
          <ul className={cx("divide-y divide-line border-y border-line")}>
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
        <EmptyState title={t("noPacks")} icon="pack">
          <Button href="/packs/new" icon="plus">
            {t("newPack")}
          </Button>
        </EmptyState>
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
