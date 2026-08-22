# Restick Ingress API Gateway

Production Nginx reverse proxy providing a unified entrypoint (Single Ingress) for the Restick platform.

---

## 🧭 Routing Map

| External Path | Target Service | Purpose / Notes |
| :--- | :--- | :--- |
| `GET/POST /api/*` | `api_svc:8080/api/*` | Auth, catalog, moderation, API key verification |
| `GET /docs`, `/openapi.yaml` | `api_svc:8080/docs` | Interactive Swagger API documentation |
| `GET/PUT /v1/objects/*` | `ftp_svc:8080/v1/objects/*` | High-throughput sticker WebP assets & uploads (`proxy_buffering off`) |
| `GET /v1/bundles/*` | `ftp_svc:8080/v1/bundles/*` | Complete `.zip` sticker pack bundles for offline client caching |
| `* /internal/*` | *Blocked (HTTP 403)* | Protects internal cluster management endpoints |
| `GET /*` | `web:8080/*` | Flutter Web Admin Dashboard & SPA routing |
| `GET /healthz` | `gateway:80/healthz` | Gateway liveness probe |

---

## ⚡ Key Optimizations

- **Zero-Buffering Streaming**: `proxy_buffering off;` and `proxy_request_buffering off;` prevents Nginx from caching multi-megabyte sticker archive downloads in memory.
- **Unified CORS**: Clients (Flutter Web & Mobile SDK) only talk to a single origin, avoiding preflight browser overhead.
- **50MB Upload Limit**: Configured `client_max_body_size 50M;` for large sticker sheet uploads.
- **HTTP/2 & Keepalive**: Upstream keepalive connections to Go microservices for reduced socket latency.

---

## 🚀 Running

The gateway is enabled by default in the root `docker-compose.yml`:

```bash
# Start all microservices + unified gateway on port 80
docker compose up -d

# Or customize host port
GATEWAY_PORT=8000 docker compose up -d
```
