# syntax=docker/dockerfile:1
# Build the Next.js static export, then serve it via nginx.
# Replaces the Flutter web build (clients/web) with a Node-built static bundle.

FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
RUN apk add --no-cache curl
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/out /usr/share/nginx/html
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD curl -fsS http://127.0.0.1:8080/healthz
CMD ["nginx", "-g", "daemon off;"]
