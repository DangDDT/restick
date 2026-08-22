FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY api_svc/go.mod api_svc/go.sum ./api_svc/
COPY contracts ./contracts
COPY api_svc ./api_svc
WORKDIR /src/api_svc
RUN go build -o /bin/api_svc ./cmd/api_svc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=build /bin/api_svc /usr/local/bin/api_svc
COPY --from=build /src/api_svc/migrations /migrations
EXPOSE 8080 50051
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD curl -fsS http://127.0.0.1:8080/healthz
ENTRYPOINT ["/usr/local/bin/api_svc"]
