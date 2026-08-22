FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY ftp_svc/go.mod ftp_svc/go.sum ./ftp_svc/
COPY contracts ./contracts
COPY ftp_svc ./ftp_svc
WORKDIR /src/ftp_svc
RUN go build -o /bin/ftp_svc ./cmd/ftp_svc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=build /bin/ftp_svc /usr/local/bin/ftp_svc
EXPOSE 8080 50052
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD curl -fsS http://127.0.0.1:8080/healthz
ENTRYPOINT ["/usr/local/bin/ftp_svc"]
