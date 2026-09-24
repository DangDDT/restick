# api_svc

Core catalog, user authentication, moderation workflow, database migration, and API key lifecycle management microservice for the Restick platform.

---

## 📖 Overview

`api_svc` is the primary business logic and metadata authority in the Restick ecosystem. Written in Go 1.25, it implements Clean Architecture principles to deliver high concurrency, strict type safety, and seamless integration with PostgreSQL and the internal gRPC service mesh.

---

## 🎯 Key Responsibilities

1. **Sticker Pack Lifecycle**:
   - Manages state transitions: `draft` ➔ `submitted` ➔ `processing` ➔ `approved` / `rejected` ➔ `published` ➔ `revoked`.
   - Supports single sticker assets and multi-sticker composite sheets.
2. **User Authentication & RBAC**:
   - Password hashing with `bcrypt`.
   - HMAC-SHA256 JWT generation and verification for `contributor` and `admin` roles.
3. **Dynamic API Key Management (`rstk_live_...`)**:
   - Issues 256-bit cryptographically secure API keys.
   - Stores keys hashed with SHA-256 in PostgreSQL.
   - Provides single-use secret display, rotation, and revocation.
4. **Moderation Queue & Analytics**:
   - Administrative review queue with structured rejection reason codes (`copyright`, `inappropriate`, `low_quality`, `other`).
   - Emergency revocation and generation of persistent tombstone records.
5. **gRPC Services**:
   - `AuthService.ValidateAPIKey`: Real-time key validation consumed by `ftp_svc` (cached for 60s).
   - `JobService`: Job dispatch and status tracking with `img_tool` and `ftp_svc`.
   - `PingService`: Mesh health check.
6. **Automatic Database Migrations**:
   - Applies forward schema migrations automatically upon startup.

---

## 🏗️ Architecture & Internal Layout

```
services/api/
├── cmd/
│   └── api_svc/              # Service entrypoint (main.go)
└── internal/
    ├── domain/               # Pure domain models (user, pack, apikey, sticker, tombstone)
    ├── application/          # Application use-cases and service orchestrators
    ├── infrastructure/
    │   ├── config/           # Environment variable loading and multi-origin CSV parsing
    │   └── postgres/         # Database migrations, queries, and store implementation
    └── interfaces/
        ├── http/             # REST HTTP Handlers, Chi router, CORS, Swagger docs
        └── grpc/             # Inbound gRPC Servers (AuthService, JobService, PingService)
```

---

## ⚙️ Configuration Reference

| Environment Variable | Default Value | Description |
| :--- | :--- | :--- |
| `API_HTTP_ADDR` | `:8080` | Bind address for HTTP REST server |
| `API_GRPC_ADDR` | `:50051` | Bind address for internal gRPC server |
| `POSTGRES_DSN` | `postgres://restick:restick@postgres:5432/restick?sslmode=disable` | PostgreSQL connection DSN |
| `POSTGRES_USER` | `restick` | Database user name (for Docker Compose fallback) |
| `POSTGRES_PASSWORD` | `restick` | Database password (for Docker Compose fallback) |
| `POSTGRES_DB` | `restick` | Database catalog name |
| `JWT_SECRET` | `change-me-jwt` | HMAC secret key used to sign authentication JWT tokens |
| `INTERNAL_TOKEN` | `dev-internal-token` | Shared bearer token for trusted inter-service callbacks |
| `WEB_ORIGIN` | `http://localhost:8088` | Allowed CORS origins (comma-separated: `https://a.com,https://b.com`) |
| `PUBLIC_FTP_ORIGIN` | `http://localhost` | Base origin used for constructing downloadable asset URLs |
| `BOOTSTRAP_ADMIN_EMAIL` | `admin@restick.local` | Email for initial admin account (seeded if DB empty) |
| `BOOTSTRAP_ADMIN_PASSWORD` | `changeme` | Password for initial admin account |
| `BOOTSTRAP_API_KEY` | `dev-api-key` | Fallback legacy seed API key for bootstrap compatibility |
| `FTP_SVC_GRPC_ADDR` | `ftp_svc:50052` | gRPC address of `ftp_svc` for background ping/jobs |
| `IMG_TOOL_GRPC_ADDR` | `img_tool:50053` | gRPC address of `img_tool` for background ping/jobs |

---

## 📖 REST API Reference

All requests accept and return `application/json` unless otherwise specified.

### 1. Authentication Endpoints

#### Register Contributor
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "artist@example.com",
  "password": "SecurePassword123!",
  "display_name": "Creative Studio"
}
```
**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "u-451e-b871-9876",
    "email": "artist@example.com",
    "display_name": "Creative Studio",
    "role": "contributor"
  }
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "artist@example.com",
  "password": "SecurePassword123!"
}
```

#### Get Current Profile
```http
GET /api/v1/me
Authorization: Bearer <jwt_token>
```

---

### 2. API Key Lifecycle (`rstk_live_...`)

#### Create API Key
```http
POST /api/v1/me/api-keys
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "Mobile iOS Client",
  "expires_in_days": 90
}
```
**Response (201 Created):**
```json
{
  "id": "key-9912-ab12",
  "name": "Mobile iOS Client",
  "raw_key": "rstk_live_3f9b8a1c4e2d7f0e5b8a1c4e2d7f0e5b",
  "masked_key": "rstk_live_3f9b...0e5b",
  "created_at": "2026-08-24T10:00:00Z",
  "expires_at": "2026-11-22T10:00:00Z"
}
```
> [!WARNING]
> The `raw_key` is returned **strictly once** in this response and is never retrievable again.

#### Rotate API Key
```http
POST /api/v1/me/api-keys/{id}/rotate
Authorization: Bearer <jwt_token>
```

#### Revoke API Key
```http
DELETE /api/v1/me/api-keys/{id}
Authorization: Bearer <jwt_token>
```

---

### 3. Sticker Pack Management

#### List Published Packs (Public Marketplace)
```http
GET /api/v1/packs?page=1&limit=20&tag=cats
X-Api-Key: rstk_live_xxxxxxxxxxxxxxxxxxxxxxxx
```
**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "pack-101-cute-cats",
      "title": "Cute Cats Vol 1",
      "slug": "cute-cats-vol-1",
      "author": "Creative Studio",
      "version": "1.0.0",
      "thumbnail_url": "https://restick.example.com/v1/objects/pack-101/thumb.webp",
      "bundle_url": "https://restick.example.com/v1/bundles/pack-101",
      "sticker_count": 16,
      "tags": ["cats", "cute", "animals"]
    }
  ],
  "total": 1
}
```

#### Create Pack Draft
```http
POST /api/v1/packs
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "title": "Neon Cyberpunk",
  "slug": "neon-cyberpunk",
  "description": "High-contrast glowing stickers for dark mode chat",
  "tags": ["cyberpunk", "neon", "cool"]
}
```

#### Submit Pack for Review
```http
POST /api/v1/packs/{id}/submit
Authorization: Bearer <jwt_token>
```

---

### 4. Admin Moderation Workflow

#### List Pending Packs in Review Queue
```http
GET /api/v1/admin/packs?status=submitted
Authorization: Bearer <admin_jwt_token>
```

#### Approve & Publish Pack
```http
POST /api/v1/admin/packs/{id}/approve
Authorization: Bearer <admin_jwt_token>
```

#### Reject Pack
```http
POST /api/v1/admin/packs/{id}/reject
Authorization: Bearer <admin_jwt_token>
Content-Type: application/json

{
  "reason_code": "copyright",
  "feedback": "Assets appear to use copyrighted material without commercial distribution license."
}
```

#### Revoke Active Pack (Generate Tombstone)
```http
POST /api/v1/admin/packs/{id}/revoke
Authorization: Bearer <admin_jwt_token>
Content-Type: application/json

{
  "reason": "DMCA takedown notice received"
}
```

#### Query Tombstones (Sync delta for Client SDKs)
```http
GET /api/v1/tombstones?since=2026-08-01T00:00:00Z
X-Api-Key: rstk_live_xxxxxxxxxxxxxxxxxxxxxxxx
```

---

## 🚀 Running Locally

### Prerequisites
- Go 1.25+
- Running PostgreSQL instance (or launch via `infra/postgres/docker-compose.yml`)

### Start Service
```bash
cd services/api

# 1. Run migrations and start server
POSTGRES_DSN="postgres://restick:restick@localhost:5432/restick?sslmode=disable" \
go run ./cmd/api_svc
```

### Run Unit & Integration Tests
```bash
cd services/api

# Run all tests with verbose output
go test -v ./...

# Run static analysis
go vet ./...
```
