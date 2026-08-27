<p align="center">
  <img src="https://raw.githubusercontent.com/DangDDT/restick/main/docs/assets/restick-banner.png" alt="Restick Banner" width="520" onerror="this.style.display='none'"/>
</p>

<h1 align="center">Restick</h1>

<p align="center">
  <strong>Enterprise-Grade Sticker Management Platform, Processing Engine & Mobile Flutter SDK</strong>
</p>

<p align="center">
  <a href="https://github.com/DangDDT/restick/actions"><img src="https://img.shields.io/badge/build-passing-brightgreen.svg" alt="Build Status"/></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8.svg?logo=go" alt="Go Version"/></a>
  <a href="https://flutter.dev"><img src="https://img.shields.io/badge/Flutter-3.47+-02569B.svg?logo=flutter" alt="Flutter Version"/></a>
  <a href="https://www.python.org"><img src="https://img.shields.io/badge/Python-3.12+-3776AB.svg?logo=python" alt="Python Version"/></a>
  <a href="https://grpc.io"><img src="https://img.shields.io/badge/gRPC-v1-244c5a.svg?logo=grpc" alt="gRPC"/></a>
  <a href="https://pub.dev/packages/restick"><img src="https://img.shields.io/pub/v/restick.svg?logo=dart" alt="Pub Version"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"/></a>
</p>

---

## 📖 Table of Contents

- [Overview](#-overview)
- [System Architecture](#-system-architecture)
- [Repository Structure](#-repository-structure)
- [Quickstart: Local Development](#-quickstart-local-development)
- [Production Deployment & Hardening](#-production-deployment--hardening)
- [End-to-End User Workflows](#-end-to-end-user-workflows)
  - [1. Contributor / Creator Workflow](#1-contributor--creator-workflow)
  - [2. Administrator / Moderator Workflow](#2-administrator--moderator-workflow)
  - [3. Mobile App Integrator Workflow](#3-mobile-app-integrator-workflow)
- [Configuration Reference (.env)](#-configuration-reference-env)
- [Security & Authentication Model](#-security--authentication-model)
- [Developer Tooling & Testing](#-developer-tooling--testing)
- [Troubleshooting & FAQ](#-troubleshooting--faq)
- [License](#-license)

---

## 📖 Overview

**Restick** is a complete, production-ready sticker infrastructure platform designed for modern messaging apps, social platforms, and creator communities. It solves the entire lifecycle of sticker assets:

- **Ingestion & Slicing**: Accepts raw composite sticker sheets or individual PNG/WebP files. Automatically slices, alpha-thresholds, and contour-detects stickers, standardizing them to optimized 512×512 transparent WebP assets.
- **Catalog & Moderation**: Provides an editorial review queue with zoomable inspections, rejection analytics, versioning, and lifecycle management (`draft` ➔ `submitted` ➔ `processing` ➔ `approved` / `rejected` ➔ `published` ➔ `revoked`).
- **High-Throughput Distribution**: Streams deterministically built `.zip` pack bundles and individual assets directly via MinIO S3 object storage with zero-buffering Nginx edge gateway.
- **Offline-First Flutter SDK**: Ready-to-use Flutter components (`StickerPicker`, `StickerMarketplace`, `StickerGallery`) with Drift SQLite offline caching, background bundle unzipping, and delta tombstone synchronization.

> [!NOTE]
> Restick is **not** an AI image generator. It is a high-performance content pipeline and delivery network for real graphic sticker assets.

---

## 🏗️ System Architecture

Restick is designed around a **Single Ingress Edge Gateway** paired with an internal **gRPC Service Mesh** and high-throughput HTTP streaming paths:

```mermaid
flowchart TD
    subgraph Clients["Clients & Consumers"]
        SDK["Flutter Mobile SDK\n(iOS / Android / Desktop)"]
        WEB["Admin & Contributor Portal\n(Next.js Web App)"]
        CLI["Developer CLI / 3rd Party Integrators"]
    end

    subgraph Ingress["Unified Edge Gateway (Public Port 80 / 443)"]
        GW["Nginx Ingress Gateway (:80)\nPreserves X-Forwarded-Proto | Single Origin | Zero-Buffering"]
    end

    subgraph Internal_HTTP["Internal HTTP Endpoints"]
        API_HTTP["api_svc (:8080)\n/api/*, /docs, /openapi.yaml"]
        FTP_HTTP["ftp_svc (:8080)\n/v1/objects/*, /v1/bundles/*"]
        WEB_HTTP["web (:8080)\nNext.js Static Export Nginx"]
    end

    subgraph Mesh["Internal gRPC Mesh (:50051 - :50053)"]
        API_SVC["api_svc (Go 1.25)\nAuth, Catalog, DB Migrations, Jobs"]
        FTP_SVC["ftp_svc (Go 1.25)\nS3 Gateway, ZIP Pack Bundler"]
        IMG_TOOL["img_tool (Python 3.12)\nOpenCV Slicing, Normalization Engine"]
    end

    subgraph Storage["Storage & Persistence"]
        PG[("PostgreSQL 16\nUsers, Packs, Keys, Audit Store")]
        MINIO[("MinIO S3 Object Storage\nRaw Sheets, WebP Stickers, Zip Bundles")]
    end

    %% Client Ingress
    SDK -->|"Single Base URL (https://domain.com)"| GW
    WEB -->|"Single Base URL (https://domain.com)"| GW
    CLI -->|"Single Base URL (https://domain.com)"| GW

    %% Gateway Routing
    GW -->|"/api/*, /docs"| API_HTTP
    GW -->|"/v1/objects/*, /v1/bundles/*"| FTP_HTTP
    GW -->|"/* (SPA)"| WEB_HTTP
    GW -.->|"/internal/*"| BLOCKED["Blocked: HTTP 403"]

    %% HTTP to gRPC internal
    API_HTTP --> API_SVC
    FTP_HTTP --> FTP_SVC

    %% gRPC Mesh
    API_SVC <-->|"gRPC: JobService, Ping"| FTP_SVC
    API_SVC <-->|"gRPC: ProcessJob, Ping"| IMG_TOOL
    FTP_SVC -->|"gRPC: ValidateAPIKey (TTL Cached 60s)"| API_SVC

    %% Storage connections
    API_SVC --- PG
    FTP_SVC --- MINIO
```

---

## 📦 Repository Structure

```
.
├── docker-compose.yml              # Production root include manifest (isolated internal ports)
├── docker-compose.override.yml     # Development-only port overrides (auto-merged locally)
├── .env.example                    # Exhaustive template for all environment configurations
├── contracts/
│   └── proto/restick/v1/          # Shared gRPC Protobuf definitions (auth, job, object, ping)
├── infra/
│   ├── postgres/                  # PostgreSQL 16 Compose definition & persistent volume
│   └── minio/                     # MinIO S3 Server & MinIO Client init container
├── services/
│   ├── gateway/                   # Nginx Edge Ingress (Single Base URL, streaming, CORS)
│   ├── api/                       # Core API microservice (Go 1.25, Chi, pgx, gRPC)
│   ├── ftp/                       # High-throughput asset streaming & pack bundler (Go 1.25)
│   └── img_tool/                  # Slicing worker & standalone CLI (Python 3.12, OpenCV)
├── clients/
│   ├── web-next/                  # Contributor & Admin Web Dashboard (Next.js Static Export)
│   └── flutter/restick/           # Official Mobile Client SDK & Example App
└── scripts/
    └── gen-proto.sh               # Protobuf code generation script for Go & Python
```

---

## 🚀 Quickstart: Local Development

### 1. Prerequisites
- **Docker Engine 24+** & **Docker Compose v2.20+**
- **Go 1.25+** (optional, for backend development)
- **Python 3.12+** & **pip** (optional, for image tool development)
- **Node.js 24+ & npm** (optional, for web-next development)

### 2. Clone & Configure Environment
```bash
git clone https://github.com/DangDDT/restick.git
cd restick

# Copy example environment configuration
cp .env.example .env
```

### 3. Start Stack
```bash
# The web dashboard (clients/web-next) is built inside Docker (multi-stage
# Node build → nginx static export). No host-side web build step needed.
docker compose up --build -d
```

### 4. Verify System Health
```bash
# Check Ingress Gateway
curl -s http://localhost/healthz
# Output: {"status":"ok","service":"gateway"}

# Check API Health
curl -s http://localhost/api/v1/healthz
# Output: {"status":"ok","service":"api_svc"}

# Check FTP Service Health (via Ingress)
curl -s http://localhost/v1/objects/healthz
# Or direct dev port: curl -s http://localhost:8081/healthz

# Check img_tool Health
curl -s http://localhost:8090/healthz
# Output: {"status":"ok","service":"img_tool"}
```

### 5. Access Dashboards
- **Web Admin & Contributor Console**: [http://localhost](http://localhost) (or [http://localhost:8088](http://localhost:8088) in dev)
  - **Default Admin Account**: `admin@restick.local`
  - **Default Password**: `changeme` (configured via `BOOTSTRAP_ADMIN_PASSWORD`)
  - **Default Bootstrap API Key**: `dev-api-key`
- **Interactive Swagger / OpenAPI Docs**: [http://localhost/docs](http://localhost/docs)
- **MinIO Console**: [http://localhost:9001](http://localhost:9001) (`restick` / `resticksecret`)

---

## 🔒 Production Deployment & Hardening

Restick is designed to run securely behind any TLS-terminating reverse proxy (Cloudflare, Nginx, Traefik, AWS ALB, Caddy) without code modifications.

### 1. Port Isolation Architecture
In production, only the **Nginx Gateway (port 80/443)** is exposed to the host. Internal databases, gRPC ports, and backend HTTP servers are strictly isolated within the internal `restick` Docker bridge network:

```bash
# Production deployment command (ignores docker-compose.override.yml):
docker compose -f docker-compose.yml up -d --build
```

| Service | Production Exposure | Local Dev Exposure (`override.yml`) |
| :--- | :--- | :--- |
| `gateway` | **Public Port 80 / 443** | Port 80 |
| `postgres` | *Internal Only* (`postgres:5432`) | Port 5432 |
| `minio` | *Internal Only* (`minio:9000`) | Port 9000, Console 9001 |
| `api_svc` | *Internal Only* (`api_svc:8080`, `50051`) | Port 8080, gRPC 50051 |
| `ftp_svc` | *Internal Only* (`ftp_svc:8080`, `50052`) | Port 8081, gRPC 50052 |
| `img_tool` | *Internal Only* (`img_tool:8080`, `50053`) | Port 8090, gRPC 50053 |
| `web` | *Internal Only* (`web:8080`) | Port 8088 |

### 2. TLS Termination & Upstream Proxy Setup
The Restick Gateway automatically detects and preserves `X-Forwarded-Proto` (supporting both direct connections and TLS-terminating reverse proxies):

#### Example: Host Nginx with Let's Encrypt / Certbot
```nginx
server {
    listen 443 ssl http2;
    server_name restick.your-domain.com;

    ssl_certificate /etc/letsencrypt/live/restick.your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/restick.your-domain.com/privkey.pem;

    client_max_body_size 50M;

    location / {
        proxy_pass http://127.0.0.1:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-Host $host;
        
        # Optimize streaming for large sticker bundles
        proxy_buffering off;
        proxy_request_buffering off;
    }
}
```

### 3. Production Environment Checklist
Before deploying to production, configure the following in `.env`:
- [ ] **Generate strong secrets**:
  ```bash
  JWT_SECRET=$(openssl rand -hex 32)
  POSTGRES_PASSWORD=$(openssl rand -hex 24)
  MINIO_SECRET_KEY=$(openssl rand -hex 24)
  INTERNAL_TOKEN=$(openssl rand -hex 32)
  ```
- [ ] **Set `WEB_ORIGIN`**: Your production web domain (e.g., `https://restick.your-domain.com`). Supports comma-separated origins.
- [ ] **Set `PUBLIC_FTP_ORIGIN`**: Set to your public domain (e.g., `https://restick.your-domain.com`).
- [ ] **Set `BOOTSTRAP_ADMIN_PASSWORD`**: Set a strong initial admin password.

---

## 🔄 End-to-End User Workflows

### 1. Contributor / Creator Workflow

```mermaid
sequenceDiagram
    autonumber
    actor Creator as Sticker Artist / Contributor
    participant Web as Web Dashboard
    participant API as api_svc
    participant FTP as ftp_svc
    participant Worker as img_tool (OpenCV)

    Creator->>Web: Register & Login (POST /api/v1/auth/login)
    Web-->>Creator: Returns JWT Bearer Token

    Creator->>Web: Click "New Pack" -> Enter Title, Slug, Description
    Web->>API: POST /api/v1/packs (status: draft)
    API-->>Web: Pack Created (UUID)

    alt Upload Raw Composite Sticker Sheet
        Creator->>Web: Upload sticker_sheet.png
        Web->>FTP: Upload sheet file
        Web->>API: Trigger Slicing Job
        API->>Worker: gRPC ProcessJob (Contour Detection & Slicing)
        Worker-->>API: Sliced 512x512 WebP Assets + Bounding Boxes
    else Upload Individual Sticker Files
        Creator->>Web: Upload individual PNG / WebP files
        Web->>FTP: Upload assets directly
    end

    Creator->>Web: Assign Emoji Hints & Search Keywords per sticker
    Creator->>Web: Click "Submit for Review"
    Web->>API: POST /api/v1/packs/{id}/submit (status: submitted)
    API-->>Web: Pack queued in Moderation Queue
```

#### Step-by-Step Instructions:
1. **Log in** to the Web Portal at `https://your-domain.com`.
2. Navigate to **My Packs** and click **+ Create Pack**.
3. Fill in pack details: Name, Slug, Author Name, Category tags.
4. **Upload Assets**:
   - *Option A (Sheet)*: Drag & drop a composite sticker sheet PNG. The backend slices and normalizes stickers automatically.
   - *Option B (Individual)*: Drag & drop individual transparent PNG/WebP files.
5. Click on each sticker to assign **Emoji shortcuts** (e.g., `:smile:`, `:heart:`) and search keywords.
6. Click **Submit for Review**. The pack status updates to `submitted`.

---

### 2. Administrator / Moderator Workflow

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Platform Moderator
    participant Web as Admin Console
    participant API as api_svc
    participant FTP as ftp_svc

    Admin->>Web: Login as Admin
    Admin->>Web: Open "Moderation Queue"
    Web->>API: GET /api/v1/admin/packs?status=submitted
    API-->>Web: List of pending packs

    Admin->>Web: Inspect pack assets (Zoom dialog, Metadata check)
    
    alt Approve Pack
        Admin->>Web: Click "Approve & Publish"
        Web->>API: POST /api/v1/admin/packs/{id}/approve
        API->>FTP: Trigger ZIP Bundle Generation (Deterministic manifest.json + WebP)
        API-->>Web: Status updated to 'published' (Visible in Marketplace)
    else Reject Pack
        Admin->>Web: Select Reason Code (copyright, low_quality, inappropriate) + Note
        Web->>API: POST /api/v1/admin/packs/{id}/reject
        API-->>Web: Status updated to 'rejected' (Contributor notified with feedback)
    else Revoke Active Pack
        Admin->>Web: Click "Revoke Pack"
        Web->>API: POST /api/v1/admin/packs/{id}/revoke
        API->>API: Insert Tombstone record
        API-->>Web: Pack removed from market; client SDKs purge locally on next sync
    end
```

#### Step-by-Step Instructions:
1. Log in with your admin account (`admin@restick.local`).
2. Navigate to **Moderation Queue**.
3. Review pending submissions: Click on any pack to inspect every individual sticker with transparent background preview.
4. **Action**:
   - **Approve**: Instantly publishes the pack to the public catalog and triggers background bundle generation.
   - **Reject**: Choose a structured reason code (`copyright`, `inappropriate`, `low_quality`, `other`) and provide feedback notes for the creator.
5. **Emergency Revocation**: If a published pack violates copyright later, click **Revoke**. A **Tombstone** record is generated so all client mobile devices will delete the cached pack on their next sync cycle.

---

### 3. Mobile App Integrator Workflow

```mermaid
sequenceDiagram
    autonumber
    actor Dev as Mobile Developer
    participant Web as Web Dashboard
    actor User as App End User
    participant App as Mobile App (Flutter SDK)
    participant GW as Restick Gateway

    Dev->>Web: Log in -> Navigate to "API Keys" -> Click "+ Generate Key"
    Web-->>Dev: Returns 'rstk_live_xxxxxxxxxxxxxxxx' (One-time modal)

    Dev->>App: Add 'restick' SDK dependency & initialize in main()
    
    Note over App,GW: Runtime App Execution
    User->>App: Opens Chat Screen -> Taps Sticker Icon
    App->>GW: GET /api/v1/packs (with X-Api-Key)
    GW-->>App: Returns Catalog JSON

    User->>App: Clicks "Download Pack" in Marketplace
    App->>GW: GET /v1/bundles/{pack_id} (Streaming ZIP archive)
    GW-->>App: Binary ZIP Stream (Stickers + manifest.json)
    App->>App: Extracts to local storage & indexes in Drift SQLite

    User->>App: Selects a sticker from StickerPicker
    App-->>User: Emits local File & Emoji metadata instantly (0ms network delay)
```

#### Step-by-Step Instructions:

##### Step 1: Provision an API Key
1. In the Web Portal, go to **API Keys**.
2. Click **+ Generate Key**, specify a name (e.g., `Mobile Production App`), and optional expiry.
3. Copy the generated key (`rstk_live_...`).

##### Step 2: Install SDK in Flutter App
In your Flutter app's `pubspec.yaml`:
```yaml
dependencies:
  flutter:
    sdk: flutter
  restick:
    hosted: https://onepub.dev/api/xdhdsdavuh
    version: ^0.1.5-flutter3.32.8
```

##### Step 3: Initialize in `main.dart`
```dart
import 'package:flutter/material.dart';
import 'package:restick/restick.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Initialize Restick Client singleton
  final RestickClient client = await Restick.open(
    RestickConfig(
      baseUrl: 'https://restick.your-domain.com',
      apiKey: 'rstk_live_your_api_key_here',
      autoSyncInterval: const Duration(minutes: 5),
    ),
  );

  runApp(MyApp(client: client));
}
```

##### Step 4: Wrap App with `RestickScope` and Embed UI
```dart
class MyApp extends StatelessWidget {
  const MyApp({required this.client, super.key});
  final RestickClient client;

  @override
  Widget build(BuildContext context) {
    return RestickScope(
      client: client,
      child: const MaterialApp(
        home: ChatScreen(),
      ),
    );
  }
}

class ChatScreen extends StatelessWidget {
  const ChatScreen({super.key});

  void _openStickerPicker(BuildContext context) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) => SizedBox(
        height: 380,
        child: StickerPicker(
          onStickerSelected: (File file, StickerAsset asset) {
            print('Selected sticker: ${file.path}, emoji: ${asset.emojiHint}');
            Navigator.pop(context);
            // Send sticker message in your chat system
          },
        ),
      ),
    );
  }

  void _openMarketplace(BuildContext context) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (context) => Scaffold(
          appBar: AppBar(title: const Text('Sticker Store')),
          body: const StickerMarketplace(),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Chat'),
        actions: [
          IconButton(
            icon: const Icon(Icons.storefront),
            onPressed: () => _openMarketplace(context),
          ),
        ],
      ),
      body: Center(
        child: ElevatedButton.icon(
          icon: const Icon(Icons.emoji_emotions_outlined),
          label: const Text('Pick Sticker'),
          onPressed: () => _openStickerPicker(context),
        ),
      ),
    );
  }
}
```

---

## ⚙️ Configuration Reference (.env)

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `GATEWAY_PORT` | `80` | Host port mapped to Nginx Ingress Gateway |
| `API_HTTP_ADDR` | `:8080` | Internal bind address for `api_svc` REST server |
| `API_GRPC_ADDR` | `:50051` | Internal bind address for `api_svc` gRPC server |
| `POSTGRES_DSN` | `postgres://restick:restick@postgres:5432/restick?sslmode=disable` | PostgreSQL connection string |
| `POSTGRES_USER` | `restick` | Database user name |
| `POSTGRES_PASSWORD` | `restick` | Database password |
| `POSTGRES_DB` | `restick` | Database catalog name |
| `JWT_SECRET` | `change-me-jwt` | Secret key for signing contributor/admin JWT tokens |
| `BOOTSTRAP_ADMIN_EMAIL` | `admin@restick.local` | Initial admin account email |
| `BOOTSTRAP_ADMIN_PASSWORD` | `changeme` | Initial admin account password |
| `BOOTSTRAP_API_KEY` | `dev-api-key` | Fallback legacy seed API key |
| `INTERNAL_TOKEN` | `dev-internal-token` | Shared token for internal microservice communication |
| `WEB_ORIGIN` | `http://localhost:8088` | Allowed CORS origins for web clients (comma-separated) |
| `FTP_HTTP_ADDR` | `:8080` | Internal bind address for `ftp_svc` HTTP streaming server |
| `FTP_GRPC_ADDR` | `:50052` | Internal bind address for `ftp_svc` gRPC server |
| `MINIO_ENDPOINT` | `minio:9000` | S3 MinIO endpoint for internal asset storage |
| `MINIO_ACCESS_KEY` | `restick` | MinIO root access key |
| `MINIO_SECRET_KEY` | `resticksecret` | MinIO root secret key |
| `MINIO_BUCKET` | `restick-assets` | MinIO bucket name for sticker files & bundles |
| `MINIO_USE_SSL` | `false` | Enable TLS/SSL connection to MinIO |
| `IMG_HTTP_ADDR` | `:8080` | Internal bind address for `img_tool` HTTP health server |
| `IMG_GRPC_ADDR` | `:50053` | Internal bind address for `img_tool` gRPC worker |
| `IMG_TOOL_ENGINE` | `simple` | Slicing algorithm (`simple` or `contour`) |
| `PUBLIC_FTP_ORIGIN` | `http://localhost` | Base origin for generating asset download URLs |
| `API_SVC_GRPC_ADDR` | `api_svc:50051` | gRPC address of `api_svc` |
| `FTP_SVC_GRPC_ADDR` | `ftp_svc:50052` | gRPC address of `ftp_svc` |
| `IMG_TOOL_GRPC_ADDR` | `img_tool:50053` | gRPC address of `img_tool` |

---

## 🔐 Security & Authentication Model

1. **User Authentication (JWT Tokens)**:
   - High-entropy HMAC-SHA256 signed JWT tokens.
   - Used for authenticated actions on the Web Dashboard (Pack creation, submission, moderation).
2. **Dynamic API Keys (`rstk_live_...`)**:
   - Secure random 256-bit token issued with standard prefix `rstk_live_`.
   - Stored hashed via SHA-256 in PostgreSQL (`api_keys` table); raw token is displayed **strictly once** to the user.
   - Revocation and instant rotation supported with zero server downtime.
3. **Cross-Service gRPC Authentication Cache**:
   - `ftp_svc` validates incoming `X-Api-Key` headers on asset downloads by querying `api_svc` via gRPC `AuthService.ValidateAPIKey`.
   - Results are cached in an in-memory TTL cache (60 seconds) to avoid database bottlenecks during high-concurrency downloads.
4. **Internal Microservice Protection**:
   - Public gateway blocks any external request to `/internal/*` with HTTP 403 Forbidden.
   - Internal cluster communication is protected by shared `INTERNAL_TOKEN`.

---

## 🛠️ Developer Tooling & Testing

### Running Tests Across Services

```bash
# 1. Test Core API Service (Go)
cd services/api
go test -v ./...

# 2. Test Asset & Bundle Service (Go)
cd ../ftp
go test -v ./...

# 3. Lint & Build Web Dashboard (Next.js)
cd ../../clients/web-next
npm run lint
npm run build

# 4. Analyze & Test Mobile SDK (Flutter)
cd ../flutter/restick
flutter analyze
flutter test
```

### Rebuilding Protobuf Definitions
If you modify Protobuf files in `contracts/proto/restick/v1/`:
```bash
./scripts/gen-proto.sh
```

---

## ❓ Troubleshooting & FAQ

### Q1: Why do I get CORS errors when accessing from another domain?
**A**: Ensure `WEB_ORIGIN` in `.env` includes your client origin (e.g. `WEB_ORIGIN=https://app.yourdomain.com,https://admin.yourdomain.com`). The backend supports comma-separated origins and sets `Vary: Origin`.

### Q2: Asset URLs have `:80` or `http://` instead of `https://`
**A**: Ensure your upstream reverse proxy (Cloudflare/Nginx/ALB) sends `proxy_set_header X-Forwarded-Proto https;`. The Gateway and API service automatically detect and preserve this header.

### Q3: Mobile emulator cannot connect to `http://localhost`
**A**: Android emulators require `10.0.2.2` instead of `localhost`. When `PUBLIC_FTP_ORIGIN` is configured with `http://localhost`, the API service automatically rewrites loopback hosts to the client's reachable IP/Host header.

### Q4: How do I backup or restore database & assets?
- **Postgres**: Backup via `docker compose exec postgres pg_dump -U restick restick > backup.sql`.
- **MinIO**: Data is stored in the Docker volume `restick_minio` or can be synced using the `mc mirror` CLI.

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
