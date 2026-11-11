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

Pre-create both host mount directories with UID/GID 10001; otherwise Docker
creates missing directories as root on Linux. The restore target must be empty:

```bash
install -d -o 10001 -g 10001 ./backups ./data-restored
```

On Docker Desktop, apply the same file-sharing ownership setup as the data
directory. Recreate `secrets/admin_password` with the new password and the
ownership/modes from Quickstart before reset or restore. After a successful
restore, point `RESTICK_DATA_HOST` at the new directory before starting;
keep the original directory untouched until recovery is accepted.

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

The factory round-trip integration test (`internal/integration`) runs the real
Python machine job; `python` on `PATH` must resolve to an interpreter with
`services/img_tool` installed (the CI `Go API` job installs Python 3.12 and the
editable package beside the Go toolchain for exactly this reason).

Run these commands from the repository root. The virtual environment keeps the
same real processor available to both Go integration tests and Python tests:

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install -e services/img_tool

# Static studio export (required by ./cmd/restick tests)
(cd clients/web-next && npm ci && npm run build)   # writes out/

# Go tests (module-local)
go -C services/api test ./...

# Real-processor factory round trip only (no missing-Python skip)
go -C services/api test -count=1 -run '^TestFactoryRoundTrip$' ./internal/integration

# Python core, local CLI and machine job
python -m unittest discover -s services/img_tool/tests -v
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

## Local acceptance evidence (2026-11-11)

Executed on the isolated factory branch, against the real single-service image
and local HTTPS origin. This is local acceptance, not a VPS deployment or a
claim that hosted CI ran successfully.

| Criterion | Exercised evidence |
| --- | --- |
| AC01 — one runtime and persistence | Actual image ran healthy as UID10001 with read-only root, only one app service; force-recreate preserved metadata and exact SDK/ZIP bytes. No Node/PostgreSQL/MinIO/gRPC runtime. |
| AC02 — owner/client separation | Auth/admin HTTP tests and production-dependency round trip; real browser login/logout, key rotation/disable; restore rejected old session and key. Offline password-reset path covered by command tests. |
| AC03 — seed and snapshots | Canonical 754 fragments +24 palettes inventory checked during seed implementation; studio restart/edit/snapshot regressions; custom mobile/desktop groups, fragments and palettes created/edited in the browser. |
| AC04 — composition | Desktop and390px mobile: pinned selection and every editable grid field unchanged after suggestion; compile, denied-clipboard manual selection, save and full reload verified. |
| AC05 — processing | Real Python child produced four512×512 stickers and128×128 thumbnails from owned2×2 fixture; ready pack downloaded, ZIP members compared byte-for-byte with public objects. |
| AC06 — failure isolation | Python machine/core and Go assets/jobs regressions cover invalid input, bounds, timeout/crash and preserving the committed version; affected packages passed with race detector. |
| AC07 — job/recovery boundaries | Jobs/publication regressions cover active-job exclusion, interrupted recovery and private work/orphans; browser reload of its running job URL resumed polling to ready. |
| AC08 — SDK integrity | Flutter config/downloader13 tests passed, including checksum/traversal/hostile ZIP cases. Real SDK payload checksum was SHA256 of sticker bytes in manifest order, independently compared with bundle members (not ZIP hash). |
| AC09 — revoke/sync | Real desktop/mobile revoke made all10 captured URLs per pack return410; repeated owner revoke preserved inclusive tombstone cursor. Round-trip regression covers both committed versions. Already-downloaded offline files cannot be recalled. |
| AC10 — file/auth boundaries | HTTP/assets/auth regressions cover traversal/symlink/private roots, query-key rejection and auth before HEAD/range/conditional responses. Cookie-backed private reference previews loaded in both browser sizes. |
| AC11 — static studio | Full12-step desktop/mobile workflow, runtime pack IDs and eight direct route/root reloads; no Next server. Both browser console error lists empty; mobile document had no horizontal overflow. |
| AC12 — offline recovery | Stopped writer, same-image backup, restore into a different empty mount with new password; old session/key401, all restored keys disabled, new key fetched the exact pre-backup pack/ZIP. Backup manifest held14 hashed allowlisted files, no work or bootstrap secret. |
| AC13 — cutover | Canonical OpenAPI validated; Compose config checked with empty bootstrap directory; actual image/static export exercised. CI installs Node24 export and Python3.12 processor in the Go job. Old host data/volumes untouched; no push, merge or remote deploy. |

Final integration checks: affected Go module `go test -race -count=1 ./...`
(11 packages), Python35 tests, Flutter config/downloader13 tests all passed.
Web lint/static export passed; the unchanged export was served by the actual
container for browser checks. Dart tests used the CLI fallback after the Dart
MCP environment reported a standalone-SDK configuration failure.

Recovery fixture: SDK checksum
`3065839d21c56d9de53358bf26c869a9b1753045bac18dfe3bfca77857a386f2`;
ZIP128168 bytes, SHA256
`5798b0b0ab4e5c332157bc14d3a8f2fdca363def75261c9751e7eaef27ee0670`,
identical before recreate and after restore.

**Not exercised:** owner-supplied AI artwork, AI-output quality, resource
benchmarks, public-domain TLS or a VPS deployment. The four-shape fixture does
not substitute for the owner's required real-AI-image acceptance run. Local
browser checks trusted only the generated test certificate. SwaggerUI loads
its assets from `unpkg.com` and needs that host reachable in the browser.
