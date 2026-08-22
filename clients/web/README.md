# Restick Web

Modern Flutter Web application for sticker contributors and platform administrators.

## 🎯 Features

- 🎨 **Contributor Workspace**:
  - Upload individual sticker files or raw sticker sheets.
  - Preview sticker assets with transparent checkerboard background.
  - Edit metadata: Pack name, slug, author, tags, and per-sticker emoji hints & search keywords.
  - Track pack status: `draft` ➔ `submitted` ➔ `processing` ➔ `approved` / `rejected` ➔ `published`.
- 🛡️ **Admin Moderation Console**:
  - Review incoming sticker packs with zoomable asset inspect dialogs.
  - Approve packs for instant distribution to the public catalog.
  - Reject packs with standard reason codes (`inappropriate`, `low_quality`, `copyright`, `other`) and feedback notes.
  - Emergency pack revocation and tombstone tracking.
- 📊 **Rejection Analytics**:
  - Breakdown of rejections by reason code and creator trends.
- 🔑 **API Key Management**:
  - Create personal and machine API keys with optional expiration dates (30, 90 days, or never).
  - One-time secure token modal with clipboard copying.
  - Instant key rotation and revocation.
  - Admin view for managing all active system tokens.
- 🌐 **Multilingual & Theming**:
  - Full English (`en`) and Vietnamese (`vi`) localization.
  - Dark and light theme modes.

---

## 🛠️ Architecture & Tech Stack

- **Framework**: Flutter 3.47+ (Targeting Web / CanvasKit)
- **State Management**: `flutter_bloc` 9.1+
- **Networking**: `dio` 5.11+ & `retrofit` 4.9+ (Type-safe declarative REST clients)
- **Local Persistence**: `drift` 2.31+ SQLite Web cache & `shared_preferences`
- **Routing**: `go_router` 17.0+
- **Dependency Injection**: `get_it`

---

## 🚀 Development Setup

### 1. Install Dependencies
```bash
flutter pub get
```

### 2. Generate Code (Retrofit & JSON Serializable)
```bash
flutter pub run build_runner build --delete-conflicting-outputs
flutter gen-l10n
```

### 3. Run Locally in Chrome
```bash
flutter run -d chrome --dart-define-from-file=configuration.dev.json
```

---

## 🐳 Production Build & Docker

```bash
# 1. Compile Flutter Web release
flutter build web --release --dart-define-from-file=configuration.dev.json

# 2. Build and launch Nginx container
docker compose up -d --build web
```
