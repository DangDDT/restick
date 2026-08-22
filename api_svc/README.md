# api_svc

Core catalog, authentication, moderation workflow, and API key lifecycle management service for Restick.

## 🎯 Responsibilities

- **Catalog & Pack Lifecycle**: Create, draft, submit, process, approve, reject, publish, and revoke sticker packs.
- **User Authentication**: JWT-based authentication for contributors and administrators.
- **Dynamic API Keys**: Issuance, SHA-256 storage, prefix masking (`rstk_live_...`), rotation, and hard revocation for SDK & machine access.
- **gRPC Services**:
  - `PingService`: Inter-service liveness check across the microservice cluster.
  - `JobService`: Orchestrates processing jobs with `img_tool` and `ftp_svc`.
  - `AuthService`: Real-time gRPC validation (`ValidateAPIKey`) consumed by `ftp_svc`.
- **Database Migrations**: Automatic forward schema migrations for PostgreSQL.

---

## ⚙️ Configuration (Environment Variables)

| Variable | Default | Description |
| :--- | :--- | :--- |
| `API_HTTP_ADDR` | `:8080` | Bind address for HTTP REST server and health check |
| `API_GRPC_ADDR` | `:50051` | Bind address for gRPC server |
| `POSTGRES_DSN` | `postgres://restick:restick@postgres:5432/restick?sslmode=disable` | PostgreSQL connection DSN |
| `JWT_SECRET` | `change-me-jwt` | HMAC secret for signing user JWT tokens |
| `BOOTSTRAP_ADMIN_EMAIL` | `admin@restick.local` | Initial admin account email seeded on empty DB |
| `BOOTSTRAP_ADMIN_PASSWORD` | `changeme` | Initial admin account password seeded on empty DB |
| `BOOTSTRAP_API_KEY` | `dev-api-key` | Fallback legacy seed API key for bootstrap compatibility |
| `INTERNAL_TOKEN` | `dev-internal-token` | Shared bearer token for internal worker callbacks |
| `WEB_ORIGIN` | `http://localhost:8088` | Allowed CORS origin for web client |
| `FTP_SVC_GRPC_ADDR` | `ftp_svc:50052` | gRPC address of `ftp_svc` (leave empty to skip ping) |
| `IMG_TOOL_GRPC_ADDR` | `img_tool:50053` | gRPC address of `img_tool` (leave empty to skip ping) |

---

## 🚀 Running Locally

### With Go
```bash
cd api_svc
go run ./cmd/api_svc
```

### With Docker
```bash
docker compose -f ../deploy/postgres/docker-compose.yml -f docker-compose.yml up --build
```

---

## 🧪 Testing & Linting

```bash
cd api_svc
go test -v ./...
go vet ./...
```

---

## 📖 REST API Endpoints Summary

| Method | Endpoint | Auth | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | Public | Service health check |
| `POST` | `/api/v1/auth/register` | Public | Register contributor account |
| `POST` | `/api/v1/auth/login` | Public | Authenticate user & get JWT |
| `GET` | `/api/v1/me` | Bearer | Get current user profile |
| `GET` | `/api/v1/me/api-keys` | Bearer | List personal API keys |
| `POST` | `/api/v1/me/api-keys` | Bearer | Create new API key (returns raw secret once) |
| `POST` | `/api/v1/me/api-keys/{id}/rotate` | Bearer | Rotate API key (returns raw secret once) |
| `DELETE` | `/api/v1/me/api-keys/{id}` | Bearer | Revoke and delete personal API key |
| `GET` | `/api/v1/admin/api-keys` | Admin | List all system API keys |
| `DELETE` | `/api/v1/admin/api-keys/{id}` | Admin | Delete any API key |
| `GET` | `/api/v1/packs` | API Key / Bearer | List published packs in marketplace |
| `GET` | `/api/v1/packs/{id}` | API Key / Bearer | Get pack details |
| `POST` | `/api/v1/packs` | Bearer | Create new pack draft |
| `POST` | `/api/v1/packs/{id}/submit` | Bearer | Submit pack draft for review |
| `GET` | `/api/v1/admin/packs` | Admin | Review moderation queue |
| `POST` | `/api/v1/admin/packs/{id}/approve` | Admin | Approve pack and publish to catalog |
| `POST` | `/api/v1/admin/packs/{id}/reject` | Admin | Reject pack with reason code |
| `GET` | `/api/v1/tombstones` | API Key | Query deleted/revoked pack tombstones |
