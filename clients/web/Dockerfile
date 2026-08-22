# syntax=docker/dockerfile:1
#
# Serve Restick Web (Flutter) via nginx.
# Build artifacts are produced on the host (Apple Silicon / Flutter 3.47),
# then copied into a slim nginx image — avoids linux/amd64 Flutter-in-Docker.
#
#   flutter build web --release --dart-define-from-file=configuration.dev.json
# Optional Wasm (Flutter 3.47 opt-in): add --wasm after verifying package:web interop.
#   docker compose up -d --build web

FROM nginx:1.27-alpine
RUN apk add --no-cache curl
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY build/web /usr/share/nginx/html
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD curl -fsS http://127.0.0.1:8080/healthz
CMD ["nginx", "-g", "daemon off;"]
