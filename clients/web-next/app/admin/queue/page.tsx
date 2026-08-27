"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/lib/api";
import type { PackListResponse } from "@/lib/types";
import { AppShell } from "@/components/app-shell";
import { RequireAdmin } from "@/components/guards";
import { EmptyState, ErrorBanner, PageHeader, Spinner, StatusPill } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

function Queue() {
  const { request } = useApi();
  const { t } = useI18n();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-queue"],
    queryFn: () => request<PackListResponse>("/api/v1/admin/packs"),
  });

  return (
    <>
      <PageHeader title={t("queueTitle")} subtitle={t("queueSubtitle")} />
      {isError && (
        <div className="mb-4">
          <ErrorBanner message={error instanceof Error ? error.message : t("loadQueueFailed")} />
        </div>
      )}
      {isLoading ? (
        <Spinner />
      ) : data && data.items.length > 0 ? (
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-zinc-200 bg-zinc-50 text-xs uppercase tracking-wider text-zinc-500">
              <tr>
                <th className="px-4 py-3 font-semibold">Pack</th>
                <th className="px-4 py-3 font-semibold">Author</th>
                <th className="px-4 py-3 font-semibold">Stickers</th>
                <th className="px-4 py-3 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {data.items.map((p) => (
                <tr key={p.id} className="hover:bg-zinc-50">
                  <td className="px-4 py-3">
                    <Link
                      href={`/admin/queue/${p.id}`}
                      className="font-medium text-indigo-600 hover:underline"
                    >
                      {p.name}
                    </Link>
                    <span className="block text-xs text-zinc-400">{p.slug}</span>
                  </td>
                  <td className="px-4 py-3 text-zinc-600">{p.author}</td>
                  <td className="px-4 py-3 text-zinc-600">{p.sticker_count}</td>
                  <td className="px-4 py-3">
                    <StatusPill status={p.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState title={t("queueEmpty")} hint={t("queueEmptyHint")} />
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAdmin>
      <AppShell>
        <Queue />
      </AppShell>
    </RequireAdmin>
  );
}
