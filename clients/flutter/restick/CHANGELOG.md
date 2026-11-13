## Unreleased

- Added optional animated WebP variants without duplicate sticker entries; the grid honors reduced motion and keeps the static file as a fallback.
- Updated bundle verification to include animated bytes after each static sticker, rejecting missing, unsafe and corrupt variant references. Upgrade clients before publishing animated packs; static-only packs remain compatible.
- Fixed ZIP verification compatibility across the declared archive 3.x/4.x dependency range without weakening raw-name or symlink checks.

## 0.1.0

- Security: bundle extraction now validates every ZIP member and `pack.json` reference before writing to disk, rejecting traversal, absolute paths, Windows-separator names, nested escapes, duplicate members, symlinks and manifest escapes; checksum is still SHA256 over sticker bytes in manifest order.
- Initial release of the official Restick Flutter client SDK.
- Seamless single Gateway ingress integration (`baseUrl`).
- Ready-to-use sticker marketplace and picker widgets (`RestickScope`, `StickerPicker`, `PackMarketplace`).
- Full offline caching for `.zip` sticker bundles and WebP assets with Drift SQLite.
- Automatic delta tombstone synchronization and local cache pruning.
- Dynamic API key authentication (`rstk_live_...`).
