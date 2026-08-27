# syntax=docker/dockerfile:1
# Build the Next.js app (Node standalone server), then run it on :8080.
# Replaces the Flutter web build (clients/web) — now with SSR-capable routes.

FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:24-alpine
WORKDIR /app
RUN apk add --no-cache curl
ENV NODE_ENV=production PORT=8080 HOSTNAME=0.0.0.0
# Next standalone output bundles the server + traced deps
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD curl -fsS http://127.0.0.1:8080/healthz
CMD ["node", "server.js"]