# Restick single-operator factory image. One process serves the Go API, the
# exported studio UI and the canonical OpenAPI document; it execs one Python
# child only while a processing job runs.
# syntax=docker/dockerfile:1

FROM node:24-alpine AS web
WORKDIR /src/clients/web-next
COPY clients/web-next/package.json clients/web-next/package-lock.json ./
RUN npm ci
COPY clients/web-next ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src/services/api
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/restick ./cmd/restick

FROM python:3.12-slim AS runtime
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    RESTICK_DATA_DIR=/data \
    RESTICK_HTTP_ADDR=:8080 \
    RESTICK_WEB_DIR=/app/web \
    RESTICK_OPENAPI_PATH=/app/openapi/v1.yaml

# OpenCV headless needs the GLib/GCC runtime libraries; no curl, no compiler.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libglib2.0-0 libgomp1 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 restick \
    && useradd --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin restick \
    && mkdir -p /data /app/openapi \
    && chown 10001:10001 /data

WORKDIR /app
COPY services/img_tool/pyproject.toml /app/img_tool/pyproject.toml
COPY services/img_tool/src /app/img_tool/src
RUN pip install /app/img_tool
COPY --from=api /out/restick /usr/local/bin/restick
COPY --from=web /src/clients/web-next/out /app/web
COPY contracts/openapi/v1.yaml /app/openapi/v1.yaml

USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=5 \
    CMD ["restick", "healthcheck"]
ENTRYPOINT ["restick"]
CMD ["serve"]
