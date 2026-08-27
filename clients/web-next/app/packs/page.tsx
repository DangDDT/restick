"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/lib/api";
import type { PackListResponse } from "@/lib/types";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { EmptyState, ErrorBanner, PageHeader, Spinner, StatusPill } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

function PackCard({ pack }: { pack: PackListResponse["items"][number] }) {
  const { t } = useI18n();
  return (
    <Link
      href={`/packs/${pack.id}`}
      className="group overflow-hidden rounded-xl border border-zinc-200 bg-white transition-shadow hover:shadow-md"
    >
      <div className="aspect-square w-full bg-zinc-100">
        {pack.thumbnail_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={pack.thumbnail_url}
            alt={pack.name}
            className="h-full w-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-4xl text-zinc-300">▦</div>
        )}
      </div>
      <div className="p-4">
        <div className="flex items-start justify-between gap-2">
          <h3 className="truncate text-sm font-semibold text-zinc-900 group-hover:text-indigo-600">
            {pack.name}
          </h3>
          <StatusPill status={pack.status} />
        </div>
        <p className="mt-1 truncate text-xs text-zinc-500">
          {pack.author} · {pack.sticker_count} {t("packStickers")} · v{pack.version}
        </p>
        {pack.tags.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {pack.tags.slice(0, 4).map((t) => (
              <span
                key={t}
                className="rounded bg-zinc-100 px-1.5 py-0.5 text-[10px] font-medium text-zinc-600"
              >
                {t}
              </span>
            ))}
          </div>
        )}
      </div>
    </Link>
  );
}

function MyPacks() {
  const { request } = useApi();
  const { t } = useI18n();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["my-packs"],
    queryFn: () => request<PackListResponse>("/api/v1/me/packs"),
  });

  return (
    <>
      <PageHeader
        title={t("myPacksTitle")}
        subtitle={t("myPacksSubtitle")}
        actions={
          <Link
            href="/packs/new"
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            {t("navUploadPack")}
          </Link>
        }
      />
      {isError && (
        <div className="mb-4">
          <ErrorBanner message={error instanceof Error ? error.message : t("loadPacksFailed")} />
        </div>
      )}
      {isLoading ? (
        <Spinner />
      ) : data && data.items.length > 0 ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {data.items.map((p) => (
            <PackCard key={p.id} pack={p} />
          ))}
        </div>
      ) : (
        <EmptyState
          title={t("noPacks")}
          hint={t("noPacksHint")}
        />
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <MyPacks />
      </AppShell>
    </RequireAuth>
  );
}
