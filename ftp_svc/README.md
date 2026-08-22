# ftp_svc

High-throughput object storage gateway and pack bundle generation service for Restick.

## 🎯 Responsibilities

- **S3 / MinIO Integration**: Manages asset upload, retrieval, and bucket lifecycle in MinIO.
- **Pack Bundle Streaming**: Asynchronously generates or streams deterministic `.zip` sticker pack archives containing optimized `.webp` stickers and `manifest.json`.
- **Public & Authenticated Asset Serving**:
  - Raw objects: `/api/v1/objects/{key}`
  - Pack bundles: `/api/v1/packs/{id}/bundle`
- **Dynamic API Key Validation with Caching**:
  - Calls `api_svc` via gRPC (`AuthService.ValidateAPIKey`) to authenticate incoming client requests.
  - Maintains an in-memory `sync.Map` cache with a 60-second TTL to handle heavy download traffic with near-zero latency.

---

## ⚙️ Configuration (Environment Variables)

| Variable | Default | Description |
| :--- | :--- | :--- |
| `FTP_HTTP_ADDR` | `:8080` | Bind address for HTTP streaming gateway |
| `FTP_GRPC_ADDR` | `:50052` | Bind address for gRPC server |
| `MINIO_ENDPOINT` | `minio:9000` | MinIO host and port |
| `MINIO_ACCESS_KEY` | `restick` | MinIO access key / root user |
| `MINIO_SECRET_KEY` | `resticksecret` | MinIO secret key / root password |
| `MINIO_BUCKET` | `restick-assets` | Target bucket for sticker files & bundles |
| `MINIO_USE_SSL` | `false` | Enable SSL/TLS for MinIO connection |
| `API_SVC_GRPC_ADDR` | `api_svc:50051` | gRPC address of `api_svc` for auth & ping |
| `IMG_TOOL_GRPC_ADDR` | `img_tool:50053` | gRPC address of `img_tool` |

---

## 🚀 Running Locally

### With Go
```bash
cd ftp_svc
go run ./cmd/ftp_svc
```

### With Docker
```bash
docker compose -f ../deploy/minio/docker-compose.yml -f docker-compose.yml up --build
```

---

## 🧪 Testing & Linting

```bash
cd ftp_svc
go test -v ./...
go vet ./...
```
