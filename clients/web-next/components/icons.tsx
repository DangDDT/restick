// One drawn icon set: 20px grid, 1.5 stroke, round caps. No emoji or glyph icons.
import type { SVGProps } from "react";

const PATHS = {
  factory: "M3 16V8l4 2.5V8l4 2.5V4h3v12M3 16h14M6.5 13h1M10.5 13h1",
  library: "M4 4h3v12H4zM9 4h3v12H9zM14 5.5l2.8-.7 2.2 10.6-2.8.7z",
  character: "M10 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM4 17c.6-3 3-5 6-5s5.4 2 6 5",
  pack: "M3.5 7 10 3.5 16.5 7v6L10 16.5 3.5 13zM3.5 7 10 10.5 16.5 7M10 10.5v6",
  key: "M7.5 12.5a3.5 3.5 0 1 1 2.8-1.4L17 17.5V15h-2v-2h-2l-2.2-2.2",
  docs: "M5 3h7l3 3v11H5zM12 3v3h3M7.5 10h5M7.5 13h5",
  plus: "M10 4v12M4 10h12",
  search: "M9 15a6 6 0 1 1 0-12 6 6 0 0 1 0 12zM13.5 13.5 17 17",
  close: "M5 5l10 10M15 5 5 15",
  check: "M4.5 10.5 8.5 14.5 15.5 6",
  copy: "M7 7V4h9v9h-3M4 7h9v9H4z",
  upload: "M10 13V4M6.5 7.5 10 4l3.5 3.5M4 13v3h12v-3",
  download: "M10 4v9M6.5 9.5 10 13l3.5-3.5M4 16h12",
  trash: "M4 6h12M8 6V4h4v2M6 6l.7 10h6.6L14 6M8.5 9v4M11.5 9v4",
  edit: "M4 16l.8-3.5L13 4.3a1.4 1.4 0 0 1 2 0l.7.7a1.4 1.4 0 0 1 0 2L7.5 15.2zM11.5 5.8l2.7 2.7",
  chevronRight: "M8 4.5 13.5 10 8 15.5",
  chevronDown: "M4.5 8 10 13.5 15.5 8",
  menu: "M3.5 6h13M3.5 10h13M3.5 14h13",
  more: "M5 10h.01M10 10h.01M15 10h.01",
  sun: "M10 14a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM10 2.5v1.5M10 16v1.5M2.5 10H4M16 10h1.5M4.7 4.7l1 1M14.3 14.3l1 1M4.7 15.3l1-1M14.3 5.7l1-1",
  moon: "M16.5 11.5A6.5 6.5 0 0 1 8.5 3.5a6.5 6.5 0 1 0 8 8z",
  logout: "M8 4H4.5v12H8M12 6.5 15.5 10 12 13.5M15.5 10H8",
  refresh: "M16 10a6 6 0 1 1-1.8-4.2M16 3.5v3.3h-3.3",
  alert: "M10 3 17.5 16h-15zM10 8v3.5M10 13.7v.01",
  info: "M10 17a7 7 0 1 0 0-14 7 7 0 0 0 0 14zM10 9v4.5M10 6.7v.01",
  image: "M3.5 4.5h13v11h-13zM3.5 13l4-4 3 3 2-2 4 4M13 8a1 1 0 1 0 0-.01",
  sparkle: "M10 3l1.6 4.4L16 9l-4.4 1.6L10 15l-1.6-4.4L4 9l4.4-1.6zM15.5 14.5v3M14 16h3",
  lock: "M5.5 9h9v7h-9zM7.5 9V6.5a2.5 2.5 0 0 1 5 0V9",
  external: "M8 5H4.5v10.5H15V12M11 4.5h4.5V9M15.5 4.5 9 11",
  scissors: "M5.5 7a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM5.5 17a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM7 6l9 8M7 14l9-8",
  filter: "M3.5 5h13l-5 6v5l-3-1.5V11z",
  grid: "M4 4h5v5H4zM11 4h5v5h-5zM4 11h5v5H4zM11 11h5v5h-5z",
  list: "M7 5h9M7 10h9M7 15h9M4 5h.01M4 10h.01M4 15h.01",
  heart: "M10 16.5S3.5 12.6 3.5 8a3.3 3.3 0 0 1 6.5-.8A3.3 3.3 0 0 1 16.5 8c0 4.6-6.5 8.5-6.5 8.5z",
  star: "M10 3l2.1 4.3 4.7.7-3.4 3.3.8 4.7L10 13.7 5.8 16l.8-4.7L3.2 8l4.7-.7z",
  bear: "M5.2 4.2a2 2 0 1 0 0 3.2M14.8 4.2a2 2 0 1 1 0 3.2M10 17a6 5.2 0 1 0 0-10.4A6 5.2 0 0 0 10 17zM7.6 11h.01M12.4 11h.01M9 13.2c.6.5 1.4.5 2 0",
} as const;

export type IconName = keyof typeof PATHS;

export function Icon({
  name,
  size = 18,
  ...rest
}: { name: IconName; size?: number } & Omit<SVGProps<SVGSVGElement>, "name">) {
  return (
    <svg
      viewBox="0 0 20 20"
      width={size}
      height={size}
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...rest}
    >
      <path d={PATHS[name]} />
    </svg>
  );
}

export type Tone = "pink" | "lilac" | "mint" | "sky" | "butter" | "peach";

/** Candy tints, one per station. Text-on-soft pairs are >= 4.5:1 in both themes. */
export const TONE: Record<Tone, string> = {
  pink: "bg-pink-soft text-pink-ink",
  lilac: "bg-lilac-soft text-lilac-ink",
  mint: "bg-mint-soft text-mint-ink",
  sky: "bg-sky-soft text-sky-ink",
  butter: "bg-butter-soft text-butter-ink",
  peach: "bg-peach-soft text-peach-ink",
};

/** Rounded colour tile around an icon. */
export function IconTile({ name, tone, size = 32, className }: { name: IconName; tone: Tone; size?: number; className?: string }) {
  return (
    <span
      aria-hidden="true"
      style={{ width: size, height: size }}
      className={`wiggle-target inline-flex shrink-0 items-center justify-center rounded-[10px] ${TONE[tone]} ${className ?? ""}`}
    >
      <Icon name={name} size={Math.round(size * 0.56)} strokeWidth={1.75} />
    </span>
  );
}

/** Restick's bear mascot. Fixed brand colours (docs/brand/BRAND.md); identical to public/mascot.svg. */
export function Mascot({ size = 48, className }: { size?: number; className?: string }) {
  return (
    <svg viewBox="0 0 64 64" width={size} height={size} aria-hidden="true" focusable="false" className={className}>
      <circle cx="15" cy="16" r="9.5" fill="#A98BF0" />
      <circle cx="49" cy="16" r="9.5" fill="#A98BF0" />
      <circle cx="15.5" cy="16.5" r="5" fill="#D2C0FA" />
      <circle cx="48.5" cy="16.5" r="5" fill="#D2C0FA" />
      <ellipse cx="32" cy="36" rx="25.5" ry="22.5" fill="#A98BF0" />
      <ellipse cx="32" cy="43.5" rx="11" ry="8.5" fill="#fff" />
      <circle cx="22" cy="33" r="3.4" fill="#2B2433" />
      <circle cx="42" cy="33" r="3.4" fill="#2B2433" />
      <circle cx="23.2" cy="31.8" r="1.1" fill="#fff" />
      <circle cx="43.2" cy="31.8" r="1.1" fill="#fff" />
      <ellipse cx="14" cy="41" rx="4.6" ry="3.2" fill="#F58BC0" opacity="0.8" />
      <ellipse cx="50" cy="41" rx="4.6" ry="3.2" fill="#F58BC0" opacity="0.8" />
      <ellipse cx="32" cy="40.6" rx="3" ry="2.2" fill="#2B2433" />
      <path d="M27.8 45.2c1.9 2 6.5 2 8.4 0" fill="none" stroke="#2B2433" strokeWidth="1.9" strokeLinecap="round" />
    </svg>
  );
}

/** Route -> icon and tint, shared by the page header. */
export const STATIONS: Array<{ href: string; icon: IconName; tone: Tone }> = [
  { href: "/start", icon: "sparkle", tone: "butter" },
  { href: "/factory", icon: "factory", tone: "pink" },
  { href: "/library", icon: "library", tone: "lilac" },
  { href: "/characters", icon: "character", tone: "peach" },
  { href: "/packs", icon: "pack", tone: "mint" },
  { href: "/keys", icon: "key", tone: "butter" },
  { href: "/api-docs", icon: "docs", tone: "sky" },
];
