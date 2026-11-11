# Restick

Single-operator sticker factory: a local studio for composing prompts for an
external AI image tool, slicing the resulting sheet into a sticker pack, and
serving that pack to the existing Flutter SDK. One Go process, one SQLite
database, one bind-mounted data directory, and one Python child only while a
job runs. There is no Node runtime, PostgreSQL, MinIO, gRPC mesh or FTP service.

- **Studio UI:** `/login`, `/factory`, `/library`, `/characters`, `/packs`,
  `/packs/new`, `/packs/detail?id=…`, `/keys`, `/api-docs` (Vietnamese UI).
- **Read API for the SDK:** `GET /api/v1/packs*`, `/api/v1/tombstones`,
  `/v1/objects/{key...}`, `/v1/bundles/{id}/v/{version}` with `X-Api-Key` or the
  owner session cookie.
- **Admin API:** `/api/v1/admin/*`, owner session only.
- **Canonical contract:** [`contracts/openapi/v1.yaml`](contracts/openapi/v1.yaml)
  (served read-only at `/openapi.yaml`, Swagger shell at `/docs`).

The design is in
[`docs/specs/2026-11-10-single-operator-sticker-factory-design.md`](docs/specs/2026-11-10-single-operator-sticker-factory-design.md).

## Repository layout

```
Dockerfile                 multi-stage Node 24 web + Go 1.26 + Python 3.12 runtime
docker-compose.yml         the one application service
services/api               Go server, SQLite storage, auth, packs, jobs, assets
services/img_tool          Python sheet slicer, local CLI and machine job
clients/web-next           Next.js static studio export (built into the image)
clients/flutter/restick    Flutter SDK (unchanged read contract)
contracts/openapi/v1.yaml  single HTTP source of truth
```

## Quickstart (Docker Compose)

1. **Create the data directory with the image's owner.** The container runs as
   UID/GID `10001` and never chowns or escalates. On Linux:

   ```bash
   install -d -o 10001 -g 10001 ./data
   ```

   On Docker Desktop, grant the mounted folder to UID/GID `10001` using the
   Desktop file-sharing settings. A data directory owned by root makes startup
   fail before traffic is accepted.

2. **Create the one-time owner password secret inside a directory** (a file,
   never an environment value). The whole directory is mounted read-only at
   `/run/secrets`, so the file may be deleted after bootstrap without breaking
   `docker compose up`:

   ```bash
   # Linux: the container runs as UID/GID 10001 and never chowns.
   install -d -m 700 -o 10001 -g 10001 ./secrets
   printf '%s\n' 'replace-with-a-strong-password' > ./secrets/admin_password
   chown 10001:10001 ./secrets/admin_password && chmod 600 ./secrets/admin_password
   ```

   On Docker Desktop, create `./secrets` (mode 700) and `./secrets/admin_password`
   (mode 600); the Desktop file-sharing layer presents them to container UID
   10001. The first start of a fresh data directory requires the file. Once the
   owner exists the file is never read: you may delete it, and restarting never
   changes the password hash.

3. **Configure the environment** (`cp .env.example .env`) and set at least
   `RESTICK_PUBLIC_BASE_URL` to the public HTTPS origin. Pin an immutable image
   tag or digest in `RESTICK_IMAGE`; never deploy a mutable `latest` tag.

4. **Start and verify:**

   ```bash
   docker compose up -d --build
   docker compose ps          # health: healthy
   docker compose logs -f restick
   ```

   The HTTP port is bound to `127.0.0.1:8080` only. Open the studio through the
   TLS proxy origin (below), then log in with the owner password.

## TLS reverse proxy prerequisite

The application never terminates TLS and exposes no proxy service. Put a real
TLS-terminating proxy (host Nginx, Caddy, Traefik, Cloudflare, …) in front of
`127.0.0.1:8080` and configure:

- `client_max_body_size` **at least 8 MiB + 64 KiB** (the largest multipart
  upload); `proxy_request_buffering off` while uploads stream.
- Upstream/response timeouts suitable for ZIP downloads; `proxy_buffering off`
  so bundles stream instead of buffering in the proxy.
- `RESTICK_PUBLIC_BASE_URL` exactly equal to the HTTPS origin clients use;
  admin unsafe requests enforce this `Origin`.
- `RESTICK_TRUSTED_PROXY_CIDRS` only when the immediate peer is your proxy
  (for example the Docker bridge gateway). It is empty by default, so
  `X-Forwarded-For` is ignored and login throttling uses the socket peer.

Example host Nginx location:

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto https;
    client_max_body_size 8300k;
    proxy_request_buffering off;
    proxy_buffering off;
    proxy_read_timeout 300s;
}
```

## Runtime configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `RESTICK_DATA_DIR` | `/data` | SQLite + `sources/`, `work/`, `packs/`, `bundles/`; must be writable by UID 10001 |
| `RESTICK_HTTP_ADDR` | `:8080` | Container listen address |
| `RESTICK_PUBLIC_BASE_URL` | — | Required HTTPS origin; source of asset URLs and Origin checks |
| `RESTICK_ADMIN_PASSWORD_FILE` | — | Bootstrap secret for a fresh database only |
| `RESTICK_TRUSTED_PROXY_CIDRS` | empty | Comma-separated real proxy peers for client-IP parsing |
| `RESTICK_PROCESS_TIMEOUT` | `300s` | One processing child at most |
| `RESTICK_WEB_DIR` | `/app/web` | Exported static UI inside the image |
| `RESTICK_OPENAPI_PATH` | `/app/openapi/v1.yaml` | Canonical OpenAPI runtime copy |

Health: `GET /healthz` returns `{"status":"ok","service":"restick"}` only after
the database answers and the data root accepts a write; otherwise `503` with no
private diagnostics. The image healthcheck runs `restick healthcheck`, which
probes loopback HTTP without taking the app lock or opening the database.

## Offline maintenance

All maintenance commands are in the same image and take the exclusive app lock.
**Stop the server first** (`docker compose stop restick`); a running writer makes
them fail with "stop the service first".

```bash
# Back up database + sources/packs/bundles (work/ and secrets excluded).
docker compose run --rm -v "$PWD/backups:/backup" restick backup --output /backup/2026-11-11

# Restore into a NEW EMPTY data dir with a NEW owner password.
docker compose run --rm \
  -v "$PWD/backups:/backup" -v "$PWD/data-restored:/data" \
  restick restore --input /backup/2026-11-11 --password-file /run/secrets/admin_password

# Reset the owner password (and every session) while the server is stopped.
docker compose run --rm restick admin set-password --password-file /run/secrets/admin_password
```

Notes:

- `restick restore` requires an empty target; after a failed restore, retry into
  **another new empty directory**. Never delete or initialise a partial tree:
  startup refuses a data dir that has `sources/`, `work/`, `packs/`, `bundles/`
  or `.restick-restore-*` without `restick.sqlite`, and never creates a fresh
  database there.
- The restore password file is trimmed of surrounding whitespace, so a trailing
  newline is fine but a password cannot begin or end with whitespace.
- Backup checksums detect corruption, not authenticity; keep backups outside the
  product and only restore backups produced on this instance.
- Restoring disables every restored API key and clears owner sessions; create a
  new key in `/keys` afterwards.

## Local development

Runtime needs Go 1.26 and Python 3.12. `cmd/restick` tests serve the actual
exported UI, so build the export before the full Go test command; targeted
storage-only tests do not need it.

```bash
# Static studio export (required by ./cmd/restick tests)
cd clients/web-next && npm ci && npm run build   # writes out/

# Go tests (module-local)
cd services/api && go test ./...

# Python core, local CLI and machine job
cd services/img_tool && pip install -e . && python -m unittest discover -s tests -v
img_tool --help
```

Run the server locally with a real data dir and secret:

```bash
export RESTICK_DATA_DIR="$PWD/.localdata"
export RESTICK_PUBLIC_BASE_URL=https://localhost
export RESTICK_ADMIN_PASSWORD_FILE="$PWD/secrets/admin_password"
export RESTICK_WEB_DIR="$PWD/clients/web-next/out"
export RESTICK_OPENAPI_PATH="$PWD/contracts/openapi/v1.yaml"
cd services/api && go run ./cmd/restick
```

`go run ./cmd/restick` defaults to `serve`; use `go run ./cmd/restick healthcheck`
to probe the running instance.

The Flutter SDK is unchanged and still reads the same summary/bundle/tombstone
contract. Create a read-only client key in `/keys` and pass it to the SDK; the
raw key is shown once.

## Image publishing

`.github/workflows/deploy-app.yml` builds the root Dockerfile and publishes
`ghcr.io/<owner>/restick:<commit-sha>` only — an immutable tag, no `latest` and
no SSH deploy. Deploy by pulling that tag or digest on the VPS and recreating
the one compose service.
