"use client";

// Read-only summary of the design a pack is cut from: grid, outline, character
// and the compiled prompt that produced its sheets.

import { useI18n } from "@/lib/i18n";
import type { Design, DesignSummary } from "@/lib/types";
import { CopyButton } from "./ui";

export function DesignPanel({ design }: { design: Design | DesignSummary }) {
  const { t } = useI18n();
  // Only a full Design carries its compiled snapshot; the pack detail summary has the id alone.
  const character = "snapshot" in design ? design.snapshot.character : undefined;
  const characterLabel = character?.name
    ? `${character.name}${design.character_id ? ` · ${design.character_id}` : ""}`
    : design.character_id;

  return (
    <div data-qa="design-panel" className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="font-display text-base font-semibold text-ink">{design.name}</p>
        <span className="num text-xs text-ink-2">
          {design.rows}×{design.columns}
        </span>
      </div>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
        <dt className="text-ink-3">{t("dimensions")}</dt>
        <dd className="num text-ink">
          {design.rows}×{design.columns}
        </dd>
        <dt className="text-ink-3">{t("outlineWidth")}</dt>
        <dd className="flex items-center gap-2 text-ink">
          <span className="num">{design.outline_width} px</span>
          <span
            aria-hidden="true"
            className="size-4 rounded-[3px] border border-line-strong"
            style={{ backgroundColor: design.outline_color }}
          />
          <code className="mono text-xs text-ink-2">{design.outline_color}</code>
        </dd>
        <dt className="text-ink-3">{t("character")}</dt>
        <dd className="min-w-0 break-words text-ink">{characterLabel || t("characterNone")}</dd>
      </dl>
      <div>
        <div className="flex items-center justify-between gap-2">
          <p className="text-xs font-medium text-ink-2">{t("compiledPrompt")}</p>
          <CopyButton text={design.compiled_prompt} variant="secondary" />
        </div>
        <pre className="mono mt-1.5 max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-ctl bg-sunk p-2.5 text-xs leading-5 text-ink-2">
          {design.compiled_prompt || t("noPrompt")}
        </pre>
      </div>
    </div>
  );
}
