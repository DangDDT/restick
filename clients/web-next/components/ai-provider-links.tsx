"use client";

import { AI_PROVIDERS } from "@/lib/ai-providers";
import { useI18n } from "@/lib/i18n";
import { BUTTON_VARIANT } from "./ui";
import { Icon } from "./icons";

/** One gesture: copy the prompt, then the link opens the tool in a new tab. */
export function AiProviderLinks({ prompt }: { prompt: string }) {
  const { t } = useI18n();
  return (
    <div className="mt-4">
      <h3 className="text-sm font-semibold">{t("aiOpenTitle")}</h3>
      <p className="mb-2 text-xs text-ink-2">{t("aiOpenHint")}</p>
      <ul className="grid grid-cols-2 gap-2">
        {AI_PROVIDERS.map((p) => (
          <li key={p.id}>
            <a
              href={p.url}
              target="_blank"
              rel="noopener noreferrer"
              title={p.hint}
              // Best effort: a failed clipboard write must not block the navigation.
              onClick={() => {
                if (prompt) void navigator.clipboard?.writeText(prompt).catch(() => {});
              }}
              className={`inline-flex w-full select-none items-center justify-between gap-2 rounded-ctl px-3 min-h-9 max-md:min-h-11 text-sm font-medium ${BUTTON_VARIANT.secondary}`}
            >
              <span className="truncate">{p.name}</span>
              <Icon name="external" size={14} className="shrink-0" />
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}
