---
name: Restick Studio, Proof Table (playful)
description: A sticker is a print job, dressed friendly. Station-ordered production line, candy tints per station, cut-line magenta action colour, bear mascot, crop-mark artwork frames.
colors:
  paper: "#f5f5f2"
  panel: "#fcfcfa"
  sunk: "#ededea"
  line: "#dcdcd8"
  line-strong: "#bdbdb9"
  ink: "#1c1c22"
  ink-2: "#4d4d56"
  ink-3: "#62626b"
  cut: "#d4126f"
  cut-soft: "#fde6f1"
  cut-ink: "#a80058"
  ok: "#1f7a4a"
  warn: "#8a5a00"
  danger: "#b3261e"
  dark-paper: "#1a1a1d"
  dark-panel: "#222226"
  dark-cut: "#ff7fba"
  tints: "pink, lilac, mint, sky, butter, peach (each with -soft and -ink)"
typography:
  display:
    fontFamily: "Bricolage Grotesque, Be Vietnam Pro, sans-serif"
    fontSize: "1.625rem"
    fontWeight: 600
    lineHeight: "2rem"
    letterSpacing: "-0.015em"
  body:
    fontFamily: "Be Vietnam Pro, Segoe UI, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  data:
    fontFamily: "JetBrains Mono, ui-monospace, monospace"
    fontSize: "0.6875rem"
    fontWeight: 400
rounded:
  ctl: "10px"
  panel: "16px"
spacing:
  unit: "4px"
  row: "36px"
  touch: "44px"
components:
  button-primary:
    backgroundColor: "{colors.cut}"
    textColor: "#ffffff"
    rounded: "{rounded.ctl}"
    padding: "0 14px"
  button-secondary:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    rounded: "{rounded.ctl}"
  panel:
    backgroundColor: "{colors.panel}"
    rounded: "{rounded.panel}"
---

# Design System: Restick Studio, Proof Table (playful)

**Status: locked 2026-11-11** (commit 416832c plus the AI provider launcher). Source of truth for values: `clients/web-next/app/globals.css` (OKLCH tokens; hex above are sRGB approximations). Components: `components/ui.tsx`, icons and mascot `components/icons.tsx`, shell `components/app-shell.tsx`, guide `components/guide.tsx`. Brand assets and rules: `docs/brand/BRAND.md`. Rationale: `docs/redesign/DIRECTION.md`.

## Overview
Operate-mode production tool for one owner on desktop and phone. Structure is the **Proof Table**: stations in order (Soạn → Tải sheet → Cắt → Pack → Key), ruled sections, job numbers. The surface is **playful**: soft candy tints, round corners, a bear mascot, plain warm Vietnamese copy. Tagline: "Làm sticker dễ như chơi." Light and dark themes share tokens; default follows the OS, user override persists.

## Colors
- **Roles:** `paper` page (faint pink wash), `panel` raised surface, `sunk` recessed wells and artwork ground, `line`/`line-strong` hairlines, `ink` / `ink-2` / `ink-3` text (>= 4.5:1).
- **Action colour: cut-line magenta (`cut`)**: the single primary button per screen, active station, kiss-cut selection (`.cutline`), focus ring. `cut-soft`/`cut-ink` for selected rows.
- **Candy tints** `pink lilac mint sky butter peach`, each as `-soft` ground + `-ink` text (>= 4.5:1 both themes): one per station/section icon tile (`IconTile`). Decorative grouping only, never state.
- **State:** `ok`, `warn`, `danger` each with `-soft`; always paired with text or icon.
- **Brand violet** `#A98BF0` is mascot, logo and app icon only; it is not a UI action colour.

## Typography
Bricolage Grotesque for headings, Be Vietnam Pro for UI (full Vietnamese), JetBrains Mono tabular for ids, hex, counts and the prompt text. Base 14px; inputs 16px under 768px (iOS zoom). Body measure <= 68ch.

## Elevation and shape
Radii: 10px controls, 16px panels, 3px stamps. Soft shadows (`--shadow-soft`) on raised panels; `--shadow-pop` on overlays. Primary button is a "sticker": solid `cut` with a 3px lower edge that presses down on `:active`. Never nest panels; group with `Section`.

## Components
Button (primary/secondary/danger/ghost, sm/md, loading, 44px on mobile; `BUTTON_VARIANT` exported for anchors), IconButton, IconTile, Field, Switch, Tabs, Panel, Section, PageHeader, Stamp/StatusPill, JobTag, ArtFrame (crop marks / checker ground / `cutline`), Dialog (centered / drawer / mobile sheet), ConfirmDialog, CopyButton, **AiProviderLinks**, EmptyState, Skeleton, banners, Pager, Kbd, CommandPalette (⌘K), Mascot, step strip + `/start` guide.

### AiProviderLinks (`/factory`)
Under the compiled prompt (desktop aside and mobile prompt sheet): ten secondary link buttons (ChatGPT, Gemini, Ideogram, Leonardo, Midjourney, Adobe Firefly, Recraft, Flux, Kling, Runway) in a 2-column grid. Click copies the prompt (best effort) and opens the tool in a new tab (`noopener noreferrer`). Data lives in `lib/ai-providers.ts`; URLs checked 2026-11-11, no pricing shown. Copy stays the only magenta primary.

## Motion
120-220ms ease-out. Authored moments: `cut-draw` selection outline, `peel` staggered grid reveal once per mount, `wiggle` on icon-tile hover, `bob` on the mascot (splash, login, empty, 404 only). `prefers-reduced-motion` disables all.

## Responsive
Desktop (>=768): sticky station rail with traveler line. Mobile: top bar, 5-slot bottom tab bar (+ More sheet), fixed prompt bar on `/factory` opening the prompt sheet (with provider links), 44px touch targets, no horizontal page scroll.

## Do / Don't
Do: one magenta primary per screen; show real artwork on the proof ground; confirm destructive actions naming the consequence; keep tints decorative. Don't: tints as status, gradient text, emoji icons, raw palette classes (zinc/indigo/...), nested cards, recolouring the bear.
