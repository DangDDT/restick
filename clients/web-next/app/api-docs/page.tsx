"use client";

import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { Icon } from "@/components/icons";
import { Button, PageHeader, cx } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

function DocsFrame() {
  const { t } = useI18n();
  // Remounting the iframe (new key) is the only retry available: a cross-document
  // iframe cannot report an HTTP error, so the frame is treated as best effort.
  const [frame, setFrame] = useState(0);
  const [loaded, setLoaded] = useState(false);

  function reload() {
    setLoaded(false);
    setFrame((value) => value + 1);
  }

  return (
    <section className="overflow-hidden rounded-panel border border-line bg-panel">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line px-4 py-2.5">
        <p className="text-sm text-ink-2">{t("docsFrameHint")}</p>
        <Button variant="ghost" size="sm" icon="refresh" onClick={reload}>
          {t("retry")}
        </Button>
      </div>
      <div className="relative">
        {!loaded && (
          <div
            role="status"
            className="absolute inset-0 z-10 flex flex-col gap-3 bg-panel p-6 text-ink-2"
          >
            <span className="skeleton h-5 w-1/3" aria-hidden="true" />
            <span className="skeleton h-4 w-2/3" aria-hidden="true" />
            <span className="skeleton h-4 w-1/2" aria-hidden="true" />
            <span className="sr-only">{t("loading")}</span>
          </div>
        )}
        <iframe
          key={frame}
          src="/docs"
          title={t("docsTitle")}
          onLoad={() => setLoaded(true)}
          className={cx(
            "block h-[calc(100dvh-17rem)] min-h-[28rem] w-full bg-panel max-md:h-[calc(100dvh-15rem)]",
            !loaded && "invisible",
          )}
        />
      </div>
    </section>
  );
}

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
              className="inline-flex min-h-9 items-center gap-2 whitespace-nowrap rounded-ctl border border-line-strong bg-panel px-3.5 text-sm font-medium text-ink hover:border-ink-3 hover:bg-sunk max-md:min-h-11"
            >
              <Icon name="external" size={16} />
              {t("docsOpenApi")}
            </a>
          }
        />
        <DocsFrame />
      </AppShell>
    </RequireAuth>
  );
}
