# ftp_svc

High-throughput object storage gateway, MinIO/S3 bridge, and deterministic ZIP pack bundle generator for Restick.

---

## 📖 Overview

`ftp_svc` is the asset delivery backbone of the Restick platform. Written in Go 1.25, it specializes in streaming large volumes of binary sticker data (individual 512×512 WebP files and compiled `.zip` pack archives) directly from MinIO S3 object storage to client applications with minimal CPU and memory overhead.

---

## 🎯 Key Responsibilities

1. **Object Storage Bridge (MinIO S3)**:
   - Manages raw asset uploads, secure retrieval, and bucket lifecycle in MinIO.
   - Supports both HTTP and TLS/HTTPS MinIO connections via `MINIO_USE_SSL`.
2. **Deterministic Pack Bundle Generation**:
   - Compiles all approved stickers of a pack into a standardized `.zip` archive containing:
     - `manifest.json` (metadata, sticker list, emoji hints, author info)
     - `stickers/*.webp` (lossless or high-quality lossy 512×512 WebP assets)
     - `thumbnail.webp`
   - Generates bundles once and stores them in MinIO for subsequent instant streaming.
3. **High-Performance In-Memory API Key Cache**:
   - Incoming downloads authenticate via `X-Api-Key: rstk_live_...`.
   - Validates keys on-the-fly by querying `api_svc` via gRPC `AuthService.ValidateAPIKey`.
   - Caches validation results in memory with a **60-second TTL** to withstand thousands of concurrent downloads without saturating the database.
4. **CORS & Expose-Headers Compliant**:
   - Supports multi-origin CORS configurations via `WEB_ORIGIN`.
   - Exposes headers necessary for client download tracking (`ETag`, `Content-Length`, `Content-Type`, `Content-Range`, `Accept-Ranges`).

---

## 📦 Pack Bundle Structure (.zip)

When a client app requests a pack bundle via `GET /v1/bundles/{pack_id}`, `ftp_svc` streams a ZIP archive with the following standardized internal layout:

```
cute-cats-vol-1.zip
├── manifest.json
├── thumbnail.webp
└── stickers/
    ├── 01_cat_happy.webp
    ├── 02_cat_sleep.webp
    ├── 03_cat_love.webp
    └── ... (up to 32 stickers)
```

### Example `manifest.json`
```json
{
  "id": "pack-101-cute-cats",
  "title": "Cute Cats Vol 1",
  "version": "1.0.0",
  "author": "Creative Studio",
  "sticker_count": 16,
  "stickers": [
    {
      "id": "stk-001",
      "file_name": "stickers/01_cat_happy.webp",
      "emoji_hint": "😺",
      "tags": ["happy", "smile", "cat"]
    }
  ]
}
```

---

## ⚙️ Configuration Reference

| Environment Variable | Default Value | Description |
| :--- | :--- | :--- |
| `FTP_HTTP_ADDR` | `:8080` | Bind address for HTTP asset streaming server |
| `FTP_GRPC_ADDR` | `:50052` | Bind address for internal gRPC server |
| `MINIO_ENDPOINT` | `minio:9000` | S3 MinIO host and port |
| `MINIO_ACCESS_KEY` | `restick` | MinIO root access key / username |
| `MINIO_SECRET_KEY` | `resticksecret` | MinIO root secret key / password |
| `MINIO_BUCKET` | `restick-assets` | Target bucket name for sticker files & pack archives |
| `MINIO_USE_SSL` | `false` | Enable TLS/SSL connection to MinIO (`true` / `false`) |
| `INTERNAL_TOKEN` | `dev-internal-token` | Shared token for authenticating internal cluster uploads |
| `JWT_SECRET` | `change-me-jwt` | Secret key for verifying JWT tokens |
| `API_KEYS` | `dev-api-key` | Static fallback API keys for development |
| `WEB_ORIGIN` | `http://localhost:8088` | Allowed CORS origins (comma-separated for multi-domain) |
| `API_SVC_GRPC_ADDR` | `api_svc:50051` | gRPC address of `api_svc` for real-time key validation |
| `IMG_TOOL_GRPC_ADDR` | `img_tool:50053` | gRPC address of `img_tool` |

---

## 📖 HTTP Endpoints Reference

### 1. Public & Client Endpoints (via Ingress Gateway)

#### Stream Raw Sticker Asset
```http
GET /v1/objects/{key...}
X-Api-Key: rstk_live_xxxxxxxxxxxxxxxxxxxxxxxx
```
- Returns raw `.webp` / image binary stream with `Content-Type: image/webp` and caching headers (`Cache-Control: public, max-age=31536000`).

#### Download Pack ZIP Bundle
```http
GET /v1/bundles/{pack_id}
X-Api-Key: rstk_live_xxxxxxxxxxxxxxxxxxxxxxxx
```
- Returns deterministic `.zip` binary stream with `Content-Type: application/zip` and `Content-Disposition: attachment; filename="<pack_slug>.zip"`.

#### Service Health Check
```http
GET /healthz
```
- Returns `{"status":"ok","service":"ftp_svc"}`.

---

### 2. Internal Microservice Endpoints (`/internal/v1/*`)
> *Protected by `X-Internal-Token` and blocked from external ingress at the Gateway.*

#### Upload Asset
```http
PUT /internal/v1/objects/{key...}
X-Internal-Token: dev-internal-token
Content-Type: image/webp

<binary payload>
```

#### Build Pack Bundle
```http
POST /internal/v1/bundles
X-Internal-Token: dev-internal-token
Content-Type: application/json

{
  "pack_id": "pack-101",
  "version": "1.0.0",
  "keys": ["objects/stk-001.webp", "objects/stk-002.webp"]
}
```

---

## 🚀 Running Locally

### Prerequisites
- Go 1.25+
- MinIO instance running on `localhost:9000` (can be launched via `infra/minio/docker-compose.yml`)

### Start Service
```bash
cd services/ftp

MINIO_ENDPOINT="localhost:9000" \
MINIO_ACCESS_KEY="restick" \
MINIO_SECRET_KEY="resticksecret" \
go run ./cmd/ftp_svc
```

### Run Unit Tests
```bash
cd services/ftp
go test -v ./...
```
