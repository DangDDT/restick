# Infrastructure Tier (PostgreSQL & MinIO S3)

Docker Compose definitions and persistent storage configurations for the Restick data tier.

---

## 📖 Overview

The `infra` directory contains containerized definitions for the two stateful persistence components of Restick:

1. **PostgreSQL 16 Alpine** (`infra/postgres`): Relational catalog store for user accounts, sticker pack metadata, moderation records, audit logs, and SHA-256 hashed API keys.
2. **MinIO S3 Server** (`infra/minio`): High-performance S3-compatible object storage for raw sticker sheets, processed 512×512 WebP sticker files, and deterministic `.zip` pack bundles.

---

## 📂 Subsystem Breakdown

### 1. PostgreSQL (`infra/postgres`)
- **Docker Image**: `postgres:16-alpine`
- **Volume**: `restick_pg` mapped to `/var/lib/postgresql/data`
- **Health Check**: Runs `pg_isready -U ${POSTGRES_USER:-restick} -d ${POSTGRES_DB:-restick}` every 5 seconds.
- **Environment Variables**:
  - `POSTGRES_USER`: Database superuser name (default: `restick`)
  - `POSTGRES_PASSWORD`: Database password (default: `restick`)
  - `POSTGRES_DB`: Primary catalog database (default: `restick`)

### 2. MinIO S3 Server & Init Container (`infra/minio`)
- **Docker Image**: `minio/minio:latest`
- **Volume**: `restick_minio` mapped to `/data`
- **Console Web UI**: Internal port `:9001` (accessible locally via `http://localhost:9001` in dev)
- **Bucket Auto-Provisioning (`minio-init`)**:
  - A lightweight companion container running `minio/mc:latest` that polls MinIO until healthy, configures the local alias, and automatically provisions the target bucket (`MINIO_BUCKET`, default: `restick-assets`) before terminating.

---

## 🚀 Running Infrastructure Independently

You can launch the storage containers separately during backend development:

```bash
# Start PostgreSQL only
docker compose -f infra/postgres/docker-compose.yml up -d

# Start MinIO only (includes minio-init auto-provisioning)
docker compose -f infra/minio/docker-compose.yml up -d

# Start both together
docker compose \
  -f infra/postgres/docker-compose.yml \
  -f infra/minio/docker-compose.yml \
  up -d
```

---

## 💾 Backup & Disaster Recovery

### PostgreSQL Backup & Restore

#### 1. Create a Snapshot Backup:
```bash
docker compose exec -T postgres pg_dump -U restick -d restick > restick_backup_$(date +%F).sql
```

#### 2. Restore from Backup:
```bash
docker compose exec -T postgres psql -U restick -d restick < restick_backup_2026-08-24.sql
```

---

### MinIO Asset Backup & Restore

#### 1. Install MinIO Client (`mc`):
```bash
# macOS
brew install minio/stable/mc

# Linux
curl -O https://dl.min.io/client/mc/release/linux-amd64/mc && chmod +x mc && sudo mv mc /usr/local/bin/
```

#### 2. Configure Alias:
```bash
mc alias set local http://localhost:9000 restick resticksecret
```

#### 3. Mirror Bucket to Local Disk:
```bash
mc mirror local/restick-assets ./minio_backup_assets/
```

#### 4. Restore Assets to MinIO:
```bash
mc mirror ./minio_backup_assets/ local/restick-assets
```
