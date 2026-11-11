"use client";

import { Wordmark } from "@/components/app-shell";
import { Icon } from "@/components/icons";
import { Button } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

// Rendered for unknown paths in the static export. Deliberately outside
// RequireAuth so it works for signed-out visitors too.
export default function NotFound() {
  const { t } = useI18n();
  return (
    <div className="flex min-h-dvh flex-col bg-paper">
      <header className="flex items-center border-b border-line bg-panel px-4 py-3 md:px-8">
        <Wordmark />
      </header>
      <main className="mx-auto flex w-full max-w-xl flex-1 items-center px-4 py-10">
        <div className="crop w-full rounded-panel px-6 py-12 text-center">
          <span className="mx-auto flex size-10 items-center justify-center rounded-full border border-line-strong bg-panel text-ink-3">
            <Icon name="alert" size={20} />
          </span>
          <h1 className="mt-4 font-display text-[1.375rem] font-semibold leading-7 text-ink">{t("pageNotFound")}</h1>
          <p className="mx-auto mt-2 max-w-[44ch] text-sm text-ink-2">{t("pageNotFoundHint")}</p>
          <div className="mt-6 flex flex-wrap justify-center gap-2">
            <Button href="/factory" icon="factory">
              {t("goToFactory")}
            </Button>
            <Button variant="secondary" icon="refresh" onClick={() => window.location.reload()}>
              {t("retry")}
            </Button>
          </div>
        </div>
      </main>
    </div>
  );
}
