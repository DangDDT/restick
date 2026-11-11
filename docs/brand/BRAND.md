# Restick brand

**Name:** Restick (product: Restick Studio). **Tagline:** "Làm sticker dễ như chơi."
**Mascot:** the Restick bear, a round pastel-purple bear head with small ears, white muzzle and pink cheeks. (Unnamed; give it a name when the product wants one.)
**Personality:** friendly, playful, tidy. A cute face on a precise tool: warm words, clear steps, no jargon.

## Colours (fixed brand values)
| Role | Hex |
|---|---|
| Bear violet (primary brand) | `#A98BF0` |
| Bear ear / light violet | `#D2C0FA` |
| Blush pink | `#F58BC0` |
| Bear ink (eyes, nose, wordmark) | `#2B2433` |
| Brand violet text (on light) | `#7C5CD6` |
| Brand blush ground | `#FDF3F9` / `#FDE6F1` |

The app UI keeps its own semantic tokens (`app/globals.css`); the magenta `cut` colour stays the action colour. Brand violet is for the mascot, logo and app icon.

## Assets (`docs/brand/`)
- `mascot.svg`, `mascot-512.png`: the bear, transparent. Source of truth for the `Mascot` component (`components/icons.tsx`).
- `logo-horizontal.svg/.png`, `logo-horizontal-dark.*`: bear + "Restick STUDIO". The SVG text uses Bricolage Grotesque and Be Vietnam Pro; the PNGs are rendered with them.
- `app-icon.svg`, `app-icon-512/192.png`, `apple-touch-icon.png`, `app-icon-maskable*`: rounded blush tile for home screens and PWA.
- `og-image.svg/.png`: 1200x630 social card.
- Web copies live in `clients/web-next/public/` (`icon.svg`, `favicon.ico` in `app/`, `apple-touch-icon.png`, `icon-192/512.png`, `icon-maskable-512.png`, `og-image.png`, `manifest.webmanifest`).

## Use
- Clear space around the bear: at least the ear radius. Minimum size 16 px (favicon) as the head only.
- Do not recolour the bear, stretch it, add outlines or put it on busy photos. On dark grounds keep the bear as is and use `logo-horizontal-dark`.
- The bear may bob gently (`.bob`) on splash, login, empty and 404 screens; it is decorative and never carries meaning alone.
- Wordmark: "Restick" in Bricolage Grotesque Bold, "STUDIO" in Be Vietnam Pro SemiBold, wide tracking.

## Not decided here
Print/sticker sheet versions of the bear, mascot expressions and a name for the mascot.
