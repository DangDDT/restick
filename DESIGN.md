---
name: Restick Studio, Proof Table
description: A sticker is a print job. Neutral proofing grounds, one cut-line magenta accent, inked stamps, crop-mark artwork frames.
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
  ctl: "4px"
  panel: "8px"
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

# Design System: Restick Studio, Proof Table

Source of truth for values: `clients/web-next/app/globals.css` (OKLCH tokens; hex above are sRGB approximations). Components: `components/ui.tsx`, icons `components/icons.tsx`, shell `components/app-shell.tsx`.

> **Update:** the studio is now playful (candy tints `pink/lilac/mint/sky/butter/peach`, 10px/16px radii, bear mascot). The "one accent / neutral" text below is partly superseded; brand rules are in `docs/brand/BRAND.md`.

## Overview
Operate-mode production tool for one owner on desktop and phone. World: a print-shop job traveler on a proofing table. Neutral ground (chroma <= 0.01) keeps sticker colour true; structure comes from hairline rules and station order, never card stacks. Light and dark themes share tokens; default follows the OS, user override persists.

## Colors
- **Roles:** `paper` page, `panel` raised work surface, `sunk` recessed wells and artwork ground, `line`/`line-strong` hairlines, `ink` / `ink-2` / `ink-3` text (all >= 4.5:1 on paper).
- **One accent, cut-line magenta (`cut`)**: primary action, active station, kiss-cut selection (`.cutline`), focus ring. Never decorative. `cut-soft`/`cut-ink` for selected rows.
- **State:** `ok`, `warn`, `danger` each with a `-soft` tint; always paired with text or icon, never colour alone.

## Typography
Bricolage Grotesque for headings (500-700, tight tracking), Be Vietnam Pro for UI (full Vietnamese support), JetBrains Mono tabular only for ids, hex, counts (`.mono`, `.num`). Base 14px; inputs 16px under 768px to prevent iOS zoom. Body measure <= 68ch.

## Elevation and shape
Hairlines, not shadows. Shadow only on overlays (`--shadow-pop`). Radii: 4px controls, 8px panels, 3px stamps. Never nest panels; group with ruled `Section`.

## Components
Button (primary/secondary/danger/ghost, sm/md, loading, 44px on mobile), IconButton, Field (+error), input/select/check classes, Switch, Tabs (underline/segmented, arrow keys), Panel, Section, PageHeader, Stamp/StatusPill (inked rectangle, pulsing when running), JobTag (`№ abc123`), ArtFrame (crop marks / checker proof ground / `cutline` when selected), Dialog (native `<dialog>`, centered / drawer / mobile sheet), ConfirmDialog (only red element is the action), CopyButton, EmptyState, Skeleton, banners, Pager, Kbd, CommandPalette (⌘K).

## Motion
120-220ms exponential ease-out. One authored moment: `cut-draw` selection outline and `peel` staggered grid reveal once per mount. `prefers-reduced-motion` disables all.

## Responsive
Desktop (>=768): sticky station rail with traveler line. Mobile: top bar, 5-slot bottom tab bar (+ More sheet), fixed prompt bar on `/factory`, 44px touch targets, no horizontal page scroll.

## Do / Don't
Do: one magenta primary per screen; show real artwork on the proof ground; confirm destructive actions naming the consequence. Don't: pastel pills, side-stripe borders, gradient text, eyebrow labels, emoji icons, raw palette classes (zinc/indigo/...), nested cards.
