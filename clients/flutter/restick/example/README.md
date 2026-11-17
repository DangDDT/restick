# restick_example

Host-app example for the Restick Flutter SDK. It exercises every public
surface:

- `Restick.open` + `RestickConfig` (base URL, API key, auto-sync interval,
  `persistApiKey`, `onError` hook)
- `RestickScope` (client provider, `RestickLoadingBar`, `RestickTheme` colors)
- Drop-in widgets: `StickerMarketplace` (with tag filter), `StickerGallery`
  (compact + regular), `StickerPicker` (embedded and bottom sheet),
  `StickerGrid`, `PackThumbnail`, `PackDownloadBanner`
- `RestickClient` API: catalog, cached catalog, pack lookup, download/delete,
  download streams, local gallery, sticker manifests, pack selection
  persistence, tombstone sync, periodic auto-sync
- Failure reporting through `RestickErrorCubit` and `config.onError`

## Run

The setup screen asks for the gateway URL and a read-only API key. Both can be
passed as `--dart-define` values instead:

```bash
cd clients/flutter/restick/example
flutter run -d macos \
  --dart-define=RESTICK_BASE_URL=https://restick.dangddt.com \
  --dart-define=RESTICK_API_KEY=rsk_your_key_here
```

Other devices:

```bash
flutter run -d <ios-device> ...
flutter run -d <android-emulator> ...
```

Append `--dart-define=RESTICK_AUTO_OPEN=true` to skip the setup form when both
`RESTICK_BASE_URL` and `RESTICK_API_KEY` are provided.

Web is not supported: the SDK needs `dart:io` and native SQLite.

macOS uses `com.apple.security.network.client` in
`macos/Runner/*.entitlements`. The example defaults `persistApiKey` to `false`
there because ad-hoc signed runners cannot use the Keychain; add
`keychain-access-groups` and a development signing team to test persistence.

## What to try

1. **Marketplace** – refresh, filter by a tag such as `con vat`, download a
   pack. Progress shows in the tile, the loading bar and the browser log.
2. **Gallery** – pick a downloaded pack, toggle compact mode, tap a sticker.
   The selected sticker appears above the tabs with its file path and animated
   variant (when the pack has one and reduced motion is off).
3. **Picker** – tap stickers in the embedded grid or open the bottom-sheet
   picker. Both emit the same `StickerAsset`.
4. **SDK API** – run each client method and watch the log: `listMarketplace`,
   `cachedMarketplace`, `getPack`, `downloadPack`, `deletePack`, `watchPack`,
   `watchDownloads`, `getStickers`, `listGallery`, `selectPack`,
   `lastSelectedPackId`, `syncRevocations`, start/stop auto-sync.
   `config.onError` and the cubit states are shown on the same screen.
