"use client";

import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { PageHeader } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

export default function Page() {
  const { t } = useI18n();
  return (
    <RequireAuth>
      <AppShell>
        <PageHeader
          title={t("docsTitle")}
          subtitle={t("docsSubtitle")}
          actions={
            <a
              href="/openapi.yaml"
              className="rounded-lg border border-zinc-300 bg-white px-4 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-50"
            >
              {t("docsOpenApi")}
            </a>
          }
        />
        <p className="mb-3 text-sm text-zinc-500">{t("docsFrameHint")}</p>
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white">
          <iframe
            src="/docs"
            title={t("docsTitle")}
            className="h-[calc(100vh-14rem)] w-full"
          />
        </div>
      </AppShell>
    </RequireAuth>
  );
}
