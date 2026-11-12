"use client";

import { useId } from "react";
import { checkClass, cx } from "@/components/ui";
import { useI18n } from "@/lib/i18n";
import type { Fragment } from "@/lib/types";

const COLOR_ROLES = [
  ["primary", "rolePrimary"],
  ["secondary", "roleSecondary"],
  ["accent", "roleAccent"],
  ["line", "roleLine"],
] as const;

export function PalettePicker({ palettes, value, onChange, disabled, characterSelected }: {
  palettes: readonly Fragment[];
  value: string;
  onChange: (id: string) => void;
  disabled?: boolean;
  characterSelected: boolean;
}) {
  const { t } = useI18n();
  const name = useId();
  const visible = palettes.filter((palette) => palette.colors &&
    (!palette.archived || palette.id === value) &&
    (!characterSelected || palette.scope !== "identity" || palette.id === value));

  return (
    <fieldset disabled={disabled} className="min-w-0">
      <legend className="mb-2 text-[13px] font-medium text-ink">{t("palette")}</legend>
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
        <label className={cx("flex min-h-11 items-center gap-2 rounded-panel border p-3 focus-within:ring-2 focus-within:ring-cut", value === "" ? "border-cut bg-cut-soft" : "border-line bg-panel")}>
          <input type="radio" name={name} value="" checked={value === ""} onChange={() => onChange("")} className={checkClass} />
          <span className="text-sm text-ink">{t("paletteNone")}</span>
        </label>
        {visible.map((palette) => {
          const conflict = characterSelected && palette.scope === "identity";
          return (
            <label key={palette.id} className={cx(
              "min-w-0 rounded-panel border p-3 focus-within:ring-2 focus-within:ring-cut",
              palette.id === value ? "border-cut bg-cut-soft" : "border-line bg-panel",
              conflict && "opacity-60",
            )}>
              <span className="flex items-start gap-2 text-sm text-ink">
                <input type="radio" name={name} value={palette.id} checked={value === palette.id} disabled={conflict} onChange={() => onChange(palette.id)} className={cx(checkClass, "mt-0.5 shrink-0")} />
                <span className="min-w-0 break-words font-medium">{palette.name}{palette.archived ? ` (${t("archived")})` : ""}</span>
              </span>
              {conflict && <span className="mt-2 block text-xs text-ink-2">{t("paletteIdentityConflict")}</span>}
              <span className="mt-3 grid grid-cols-2 gap-2">
                {COLOR_ROLES.map(([role, label]) => (
                  <span key={role} className="flex min-w-0 items-center gap-1.5">
                    <span aria-hidden="true" className="size-5 shrink-0 rounded-[3px] border border-line-strong" style={{ backgroundColor: palette.colors![role] }} />
                    <span className="min-w-0 text-[11px] leading-4 text-ink-2">
                      <span className="block">{t(label)}</span>
                      <code className="mono block">{palette.colors![role]}</code>
                    </span>
                  </span>
                ))}
              </span>
            </label>
          );
        })}
      </div>
      <p className="mt-2 text-xs text-ink-2">{t("paletteCompositionHint")}</p>
    </fieldset>
  );
}
