## Unreleased

- Added `StickerAsset.keywords` (from the manifest `keywords`, empty on older packs) and `RestickClient.suggestStickers(text)`, which returns downloaded stickers whose emoji hint or keyword matches the text (case and Vietnamese-diacritic insensitive, whole words only).
- Gallery and marketplace reads no longer block on revocation sync; startup sync is best-effort, the marketplace paints the last in-process catalog before refreshing, sticker manifests are memoized per pack, download progress rebuilds are capped to 2% steps, and network thumbnails decode at display size.
- Added optional animated WebP variants without duplicate sticker entries; the grid honors reduced motion and keeps the static file as a fallback.
- Updated bundle verification to include animated bytes after each static sticker, rejecting missing, unsafe and corrupt variant references. Upgrade clients before publishing animated packs; static-only packs remain compatible.
- Fixed ZIP verification compatibility across the declared archive 3.x/4.x dependency range without weakening raw-name or symlink checks.
- Redesigned marketplace: hero carousel, compact preview cards, search, mascot, and a pack detail sheet with remove. The internal `MarketplacePackTile` (not exported from `lib/restick.dart`) was removed.
- Animated download progress, success and failure states, with reduced-motion support and `Semantics` labels.
- Bundled the Baloo 2 font (OFL) inside the package with full Vietnamese coverage; no `google_fonts` dependency.
- Added `RestickStrings` (default Vietnamese `RestickStrings.vi`, English `RestickStrings.en` with "1 sticker / 24 stickers" plurals) via `StickerMarketplace(strings: ...)`.
- Consumes the new server fields `preview_stickers` and `theme_color`; older servers still work with fallbacks. Exposed as `StickerPack.previews` and `StickerPack.themeColor`.

## 0.1.0

- Security: bundle extraction now validates every ZIP member and `pack.json` reference before writing to disk, rejecting traversal, absolute paths, Windows-separator names, nested escapes, duplicate members, symlinks and manifest escapes; checksum is still SHA256 over sticker bytes in manifest order.
- Initial release of the official Restick Flutter client SDK.
- Seamless single Gateway ingress integration (`baseUrl`).
- Ready-to-use sticker marketplace and picker widgets (`RestickScope`, `StickerPicker`, `PackMarketplace`).
- Full offline caching for `.zip` sticker bundles and WebP assets with Drift SQLite.
- Automatic delta tombstone synchronization and local cache pruning.
- Dynamic API key authentication (`rstk_live_...`).
