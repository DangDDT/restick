# Restick Web

Modern Flutter Web application providing a comprehensive Contributor Workspace and Admin Moderation Console for the Restick platform.

---

## 📖 Overview

`clients/web` is the primary graphical interface for artists, creators, and platform moderators. Built with Flutter 3.47+ Web (CanvasKit / WASM), it communicates with the unified Ingress Gateway to manage the complete lifecycle of sticker packs, moderation reviews, rejection analytics, and API keys.

---

## ✨ Features & User Capabilities

### 1. 🎨 Contributor Workspace
- **Pack Creation**: Create and draft new sticker packs with localized titles, slugs, author profiles, and category tags.
- **Asset Upload & Slicing**:
  - Drag & drop individual PNG/WebP files or full composite sticker sheets.
  - Transparent checkerboard preview for validating alpha contours and padding.
- **Rich Metadata Editing**:
  - Assign emoji shortcuts (e.g., `😃`, `❤️`, `🔥`) to individual stickers.
  - Add search keywords to power in-app client search.
- **Lifecycle Tracking**:
  - Real-time status indicators: `draft` ➔ `submitted` ➔ `processing` ➔ `approved` / `rejected` ➔ `published`.
  - View rejection feedback and reason codes with one-click re-submission after fixing assets.

### 2. 🛡️ Admin Moderation Console
- **Review Queue**: Real-time list of all submissions awaiting review.
- **Zoomable Asset Inspector**: High-resolution inspect modal to verify sticker transparency, edge quality, and copyright compliance.
- **One-Click Approval**: Instantly publishes approved packs to the public catalog and triggers background ZIP bundle compilation.
- **Structured Rejection Workflow**: Select reason codes (`copyright`, `inappropriate`, `low_quality`, `other`) and provide creator guidance notes.
- **Emergency Revocation**: Revoke compromised or infringing packs immediately, generating a persistent **Tombstone** to purge cached assets from client devices.

### 3. 🔑 API Key Management
- Generate personal and machine API keys (`rstk_live_...`) with optional 30, 90 day, or perpetual expiry.
- Secure one-time token display modal with clipboard integration.
- Instant token rotation and revocation.
- Admin view for managing and auditing all active system tokens.

### 4. 🌐 Localization & Themes
- Built-in multi-language support: **English (`en`)** and **Vietnamese (`vi`)**.
- Responsive layout supporting both desktop browsers and tablet/mobile screens.
- Adaptive Dark and Light theme modes.

---

## 🛠️ Architecture & Tech Stack

```
clients/web/
├── lib/
│   ├── core/
│   │   ├── config/          # Runtime environment & dynamic same-origin resolution
│   │   ├── database/        # Drift SQLite web persistence (cache & offline state)
│   │   ├── di/              # GetIt dependency injection setup
│   │   ├── network/         # Dio HTTP client & Auth interceptors
│   │   └── theme/           # Color palettes, typography, theme cubits
│   ├── features/
│   │   ├── auth/            # Login, registration, JWT session management
│   │   ├── packs/           # Contributor pack creator, asset uploader, sheet preview
│   │   ├── moderation/      # Admin review queue, inspector dialogs, rejection modals
│   │   └── api_keys/        # Key generation, rotation, copy dialogs
│   ├── l10n/                # ARB localization files (app_en.arb, app_vi.arb)
│   ├── bootstrap.dart       # App initialization, service locators, BLoC observer
│   └── main.dart            # Flutter Web entrypoint
├── configuration.dev.json   # Local development config
└── configuration.prod.json  # Production config
```

- **Framework**: Flutter 3.47+ Web (CanvasKit rendering engine)
- **State Management**: `flutter_bloc` 9.1+
- **REST Client**: `dio` 5.11+ & `retrofit` 4.9+ (Type-safe declarative API bindings)
- **Local Persistence**: `drift` 2.31+ SQLite Web & `shared_preferences`
- **Routing**: `go_router` 17.0+ (Declarative URL routing & deep linking)

---

## ⚙️ Environment Configuration & Base URL

`EnvConfig` supports both compile-time defines and **dynamic same-origin runtime auto-detection**:

```dart
// clients/web/lib/core/config/env_config.dart
factory EnvConfig.load() {
  const fromBase = String.fromEnvironment('BASE_URL');
  const fromRestick = String.fromEnvironment('RESTICK_BASE_URL');
  const fromApi = String.fromEnvironment('API_BASE_URL');
  const compiled = fromBase != '' ? fromBase : (fromRestick != '' ? fromRestick : fromApi);
  
  final String base;
  if (compiled.isNotEmpty) {
    base = compiled;
  } else if (kIsWeb) {
    base = Uri.base.origin; // Same-origin relative to the served domain
  } else {
    base = 'http://localhost';
  }
  return EnvConfig(baseUrl: base);
}
```

- **When hosted behind the Gateway**: `BASE_URL` is automatically detected as `Uri.base.origin` (e.g. `https://restick.your-domain.com`), requiring **zero rebuilds** when moving across staging or production domains.

---

## 🚀 Development Setup

### 1. Install Dependencies
```bash
cd clients/web
flutter pub get
```

### 2. Generate Code (Retrofit & Drift)
```bash
flutter pub run build_runner build --delete-conflicting-outputs
flutter gen-l10n
```

### 3. Run Locally in Chrome
```bash
flutter run -d chrome --dart-define-from-file=configuration.dev.json
```

### 4. Run Static Analysis
```bash
flutter analyze
```

---

## 🐳 Production Build & Docker

### 1. Compile Release Web Assets
```bash
cd clients/web
flutter build web --release --dart-define-from-file=configuration.prod.json
```
*(Or omit `--dart-define-from-file` to enable automatic same-origin detection).*

### 2. Launch with Docker Compose
```bash
# Build and run the web container (served via internal Nginx on port 8080)
docker compose up -d --build web
```
The application is automatically served at the root URL (`http://localhost/` or `https://your-domain.com/`) via the Restick Ingress Gateway.
