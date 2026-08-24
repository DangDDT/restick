# Restick SDK Example App

A complete, runnable Flutter demonstration application showcasing how to embed the **Restick Flutter SDK** into a client app.

---

## 📱 Features Demonstrated

This example app illustrates the 3 primary UI integration patterns:

1. 🛍️ **Sticker Marketplace (`StickerMarketplace`)**:
   - Browse all published packs on the Restick platform.
   - Search packs by title, author, and category tags.
   - Download packs with real-time download & unpack progress indicators.
2. 🖼️ **Sticker Gallery (`StickerGallery`)**:
   - View installed sticker packs saved in the local Drift SQLite database.
   - Tap any sticker to preview the locally stored 512×512 WebP image file.
3. 🎨 **Sticker Picker & Keyboard (`StickerPicker`)**:
   - Tabbed sticker picker partitioned by pack chips.
   - Emits a local `File` handle and `StickerAsset` metadata upon selection.
4. 🔄 **Live Asset Preview**:
   - Displays the selected sticker file instantly from disk with zero network delay.

---

## 🚀 Running the Example

### 1. Prerequisites
- Flutter 3.47+ (or Flutter 3.32+ with support branch)
- A running Restick backend (local via `docker compose up` or remote production URL)
- An active API Key (`rstk_live_...` generated from the Web Dashboard or `dev-api-key` in local dev)

### 2. Configure Environment Variables

The example app reads its configuration via compile-time `--dart-define` flags:

```bash
cd clients/flutter/restick/example

# 1. Install dependencies
flutter pub get

# 2. Run on Chrome / macOS / iOS Simulator / Android Emulator
flutter run \
  --dart-define=RESTICK_BASE_URL=http://localhost \
  --dart-define=RESTICK_API_KEY=rstk_live_your_api_key_here
```

### 3. Running on Android Emulator
When targeting an Android emulator connecting to a local dev server on your host machine:
```bash
flutter run -d android \
  --dart-define=RESTICK_BASE_URL=http://10.0.2.2 \
  --dart-define=RESTICK_API_KEY=dev-api-key
```

### 4. Running on Real Device
Replace `RESTICK_BASE_URL` with your computer's LAN IP (e.g. `http://192.168.1.50`) or your public domain (`https://restick.your-domain.com`).

---

## 📂 Source Code Structure

- `lib/main.dart`:
  - Initializes `RestickClient` using `RestickConfig.fromEnvironment()`.
  - Wraps the widget tree in `RestickScope`.
  - Implements bottom navigation between `StickerMarketplace`, `StickerGallery`, and `StickerPicker`.
