"use client";

import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/lib/api";
import type { RejectionAnalyticsResponse } from "@/lib/types";
import { REJECTION_LABELS } from "@/lib/constants";
import { AppShell } from "@/components/app-shell";
import { RequireAdmin } from "@/components/guards";
import { EmptyState, ErrorBanner, PageHeader, Spinner } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

function Analytics() {
  const { request } = useApi();
  const { t } = useI18n();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-analytics"],
    queryFn: () => request<RejectionAnalyticsResponse>("/api/v1/admin/analytics/rejections"),
  });

  const stats = data?.analytics ?? [];
  const total = stats.reduce((sum, s) => sum + s.rejection_count, 0);

  return (
    <>
      <PageHeader title={t("analyticsTitle")} subtitle={t("analyticsSubtitle")} />
      {isError && (
        <div className="mb-4">
          <ErrorBanner message={error instanceof Error ? error.message : t("loadAnalyticsFailed")} />
        </div>
      )}
      {isLoading ? (
        <Spinner />
      ) : stats.length > 0 ? (
        <div className="space-y-4">
          <div className="rounded-xl border border-zinc-200 bg-white p-6">
            <p className="text-sm text-zinc-500">{t("totalRejections")}</p>
            <p className="text-3xl font-bold text-zinc-900">{total}</p>
          </div>
          <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-zinc-200 bg-zinc-50 text-xs uppercase tracking-wider text-zinc-500">
                <tr>
                  <th className="px-4 py-3 font-semibold">{t("reason")}</th>
                  <th className="px-4 py-3 font-semibold">{t("count")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {stats.map((s) => (
                  <tr key={s.reason_code} className="hover:bg-zinc-50">
                    <td className="px-4 py-3 text-zinc-700">
                      {REJECTION_LABELS[s.reason_code] ?? s.reason_code}
                    </td>
                    <td className="px-4 py-3 font-semibold text-zinc-900">
                      {s.rejection_count}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <EmptyState title={t("analyticsEmpty")} hint={t("analyticsEmptyHint")} />
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAdmin>
      <AppShell>
        <Analytics />
      </AppShell>
    </RequireAdmin>
  );
}
