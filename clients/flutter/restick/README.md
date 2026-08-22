# Restick Flutter SDK

<p align="center">
  <a href="https://pub.dev/packages/restick"><img src="https://img.shields.io/pub/v/restick.svg" alt="Pub Version"/></a>
  <a href="https://pub.dev/packages/restick/score"><img src="https://img.shields.io/pub/points/restick.svg" alt="Pub Points"/></a>
  <a href="https://flutter.dev"><img src="https://img.shields.io/badge/Platform-Flutter%20%7C%20iOS%20%7C%20Android%20%7C%20macOS%20%7C%20Windows%20%7C%20Web-blue.svg" alt="Platforms"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"/></a>
</p>

Official Flutter client SDK for the **Restick Platform**. Provides out-of-the-box sticker marketplaces, picker dialogs, offline ZIP bundle caching, and delta tombstone synchronization for mobile messaging applications.

---

## ✨ Features

- 🛍️ **Sticker Marketplace**: Ready-to-use UI for browsing, searching, and previewing sticker packs.
- 🎨 **Sticker Picker & Keyboard**: Fast, responsive sticker grid with emoji tabs, search, and recent stickers.
- 📦 **Offline Caching & Bundles**: Automatically downloads complete `.zip` sticker archives, extracts assets locally, and indexes them with Drift SQLite.
- 🔄 **Delta Tombstone Sync**: Automatically detects and purges revoked or deleted packs across app launches without redownloading the entire catalog.
- 🔐 **API Key Authentication**: Seamlessly integrates with Restick dynamic API keys (`rstk_live_...`).
- ⚡ **Zero Hassle Setup**: Wraps Drift database, Dio networking, and local storage into a unified `RestickClient`.

---

## 📦 Installation (via OnePub Private Registry)

### 1. Authenticate with OnePub (One-time setup per developer/CI)
```bash
dart pub global activate onepub
onepub login
```

### 2. Add `restick` to your `pubspec.yaml`

**Via OnePub Private Registry:**
```yaml
dependencies:
  restick:
    hosted:
      url: https://onepub.dev/api/xdhdsdavuh
      name: restick
    version: 0.1.0-flutter3.32.8
```

**Or directly via GitHub Private Repository:**
```yaml
dependencies:
  restick:
    git:
      url: git@github.com:DangDDT/restick.git
      path: clients/flutter/restick
      ref: support/flutter-3.32
```

---

## 🚀 Publishing New Releases to OnePub

To publish updates to your organization's private registry:

```bash
cd clients/flutter/restick

# 1. Verify package analysis & tests
flutter analyze
flutter test

# 2. Publish to OnePub
onepub pub publish
# Or using standard dart tool:
dart pub publish
```

---

## 🚀 Getting Started

### 1. Initialize the SDK

Initialize `Restick` in your `main()` before calling `runApp()`:

```dart
import 'package:flutter/material.dart';
import 'package:restick/restick.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final RestickClient client = await Restick.open(
    RestickConfig(
      baseUrl: 'https://restick.your-domain.com',
      apiKey: 'rstk_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
    ),
  );

  runApp(MyApp(client: client));
}
```

### 2. Wrap with `RestickScope`

Wrap your widget tree (or screen) with `RestickScope`:

```dart
class MyApp extends StatelessWidget {
  const MyApp({required this.client, super.key});

  final RestickClient client;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: RestickScope(
        client: client,
        child: const HomeScreen(),
      ),
    );
  }
}
```

---

## 🎨 UI Widgets

### 1. Sticker Picker (`StickerPicker`)
Embed the sticker picker into your chat input bar or bottom sheet:

```dart
IconButton(
  icon: const Icon(Icons.emoji_emotions_outlined),
  onPressed: () {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => SizedBox(
        height: 360,
        child: StickerPicker(
          onStickerSelected: (File stickerFile, StickerAsset asset) {
            print('Selected sticker file: ${stickerFile.path}');
            print('Emoji hint: ${asset.emojiHint}');
            Navigator.pop(context);
          },
        ),
      ),
    );
  },
);
```

### 2. Sticker Marketplace (`StickerMarketplace`)
Open the full marketplace screen for users to discover and download new packs:

```dart
Navigator.of(context).push(
  MaterialPageRoute(
    builder: (context) => Scaffold(
      appBar: AppBar(title: const Text('Sticker Store')),
      body: const StickerMarketplace(),
    ),
  ),
);
```

---

## 🛠️ Programmatic API

You can also use `RestickClient` directly for custom UI:

```dart
final client = RestickScope.of(context);

// 1. Fetch available packs
final List<StickerPack> packs = await client.listPacks();

// 2. Download and install a pack
await client.downloadPack(packId: 'pack-123');

// 3. Check installed packs
final List<StickerPack> installed = await client.getInstalledPacks();

// 4. Delete an installed pack locally
await client.deletePackLocally(packId: 'pack-123');
```

---

## ⚙️ Configuration Options

| Option | Type | Description |
| :--- | :--- | :--- |
| `baseUrl` | `String` | Single Gateway base URL for all catalog, API, and sticker assets |
| `apiKey` | `String` | API Key (`rstk_live_...`) for authorized requests |
| `autoSyncInterval` | `Duration` | Automatic tombstone / catalog synchronization interval (default: 5 minutes) |
| `persistApiKey` | `bool` | Persist API Key in secure storage (default: true) |

---

## 📱 Example App

Check out the fully functional example application in [`clients/flutter/restick/example`](https://github.com/DangDDT/restick/tree/main/clients/flutter/restick/example).

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.
