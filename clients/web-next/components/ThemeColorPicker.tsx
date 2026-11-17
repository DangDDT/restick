"use client";

import { useI18n } from "@/lib/i18n";
import { inputClass } from "@/components/ui";

export const THEME_PRESETS = ["#EBE3FF", "#FFDDEE", "#FFE6CF", "#D9F4E7", "#FFF2C2"] as const;

const HEX = /^#[0-9a-fA-F]{6}$/;

/** True for "" (use suggestion) or a #RRGGBB colour, the same set the API accepts. */
export const isThemeColor = (value: string) => value === "" || HEX.test(value);

/**
 * Admin pack background picker. `value` is the admin choice; "" means "use the
 * suggestion". `previews` are sticker thumbnail URLs (first 6 are shown).
 */
export function ThemeColorPicker({
  value,
  suggested,
  previews,
  onChange,
}: {
  value: string;
  suggested: string;
  previews: string[];
  onChange: (value: string) => void;
}) {
  const { t } = useI18n();
  const valid = isThemeColor(value);
  const effective = value || suggested;
  const swatch = "h-8 w-8 rounded-full border border-line-strong focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cut aria-pressed:ring-2 aria-pressed:ring-cut";

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        {THEME_PRESETS.map((color) => (
          <button
            key={color}
            type="button"
            className={swatch}
            style={{ backgroundColor: color }}
            aria-label={color}
            aria-pressed={value.toUpperCase() === color}
            onClick={() => onChange(color)}
          />
        ))}
        <input
          value={value}
          onChange={(e) => onChange(e.target.value.trim())}
          placeholder="#RRGGBB"
          maxLength={7}
          aria-label={t("themeColor")}
          aria-invalid={!valid || undefined}
          className={`${inputClass} !w-28 font-mono`}
        />
        <button type="button" className="text-sm text-cut underline" onClick={() => onChange("")}>
          {t("themeColorUseSuggestion")}
        </button>
      </div>
      {!valid && <p role="alert" className="text-xs text-danger">{t("themeColorInvalid")}</p>}
      <p className="text-xs text-ink-2">
        {t("themeColorSuggested")}: {suggested || "-"} · {t("themeColorEffective")}: {valid ? effective || "-" : "-"}
      </p>
      <div
        className="flex min-h-16 flex-wrap items-center gap-2 rounded-ctl border border-line p-2"
        style={{ backgroundColor: valid && effective ? effective : undefined }}
      >
        {previews.slice(0, 6).map((src) => (
          // eslint-disable-next-line @next/next/no-img-element
          <img key={src} src={src} alt="" className="h-12 w-12 object-contain" />
        ))}
      </div>
    </div>
  );
}
