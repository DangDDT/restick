## 0.1.2-flutter3.32.8

- Removed `sqlite3_flutter_libs` to eliminate native symbol conflicts with `sqlcipher_flutter_libs`.
- Supported custom `QueryExecutor` injection via `RestickConfig(databaseExecutor: ...)` for SQLCipher / custom database openers.
- Exported `QueryExecutor`, `DatabaseConnection`, and `RestickDatabase` for host application custom database wiring.
- Widened `archive: ">=3.6.1 <5.0.0"` to support host applications using `image ^4.5.4`.

## 0.1.0-flutter3.32.8

- Dedicated Flutter 3.32.8 backward-compatibility release stream.
- Relaxed SDK and dependency constraints (`flutter: ">=3.32.0 <3.47.0"`).
- Full feature parity with 0.1.0 (Single Gateway, Drift SQLite, Marketplace, Picker, Tombstone sync).

## 0.1.0

- Initial release of the official Restick Flutter client SDK.
- Seamless single Gateway ingress integration (`baseUrl`).
- Ready-to-use sticker marketplace and picker widgets (`RestickScope`, `StickerPicker`, `PackMarketplace`).
- Full offline caching for `.zip` sticker bundles and WebP assets with Drift SQLite.
- Automatic delta tombstone synchronization and local cache pruning.
- Dynamic API key authentication (`rstk_live_...`).
