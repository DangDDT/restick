FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY services/ftp/go.mod services/ftp/go.sum ./services/ftp/
COPY contracts ./contracts
COPY services/ftp ./services/ftp
WORKDIR /src/services/ftp
RUN go build -o /bin/ftp_svc ./cmd/ftp_svc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=build /bin/ftp_svc /usr/local/bin/ftp_svc
EXPOSE 8080 50052
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD curl -fsS http://127.0.0.1:8080/healthz
ENTRYPOINT ["/usr/local/bin/ftp_svc"]
