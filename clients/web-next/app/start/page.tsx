"use client";

import { AppShell } from "@/components/app-shell";
import { StepGuide } from "@/components/guide";
import { RequireAuth } from "@/components/guards";
import { PageHeader } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

function Start() {
  const { t } = useI18n();
  return (
    <>
      <PageHeader title={t("startTitle")} subtitle={t("startSubtitle")} />
      <div className="max-w-3xl">
        <StepGuide />
      </div>
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <Start />
      </AppShell>
    </RequireAuth>
  );
}
