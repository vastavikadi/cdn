# Multi-stage Dockerfile for CDN Origin and Edge services
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Build static Linux binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/origin .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/edge ./edge

# --- Origin Image ---
FROM alpine:3.20 AS origin
RUN apk --no-cache add ca-certificates curl
WORKDIR /app

COPY --from=builder /bin/origin /usr/local/bin/origin
RUN mkdir -p /app/data /app/assets

ENV PORT=8080
ENV ASSETS_ROOT=/app/assets
ENV EDGE_INVALIDATION_TOKEN=cdn-local-invalidation-secret-change-me
ENV EDGE_URL_1=http://edge-india:8081
ENV EDGE_URL_2=http://edge-europe:8082

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=3s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/origin"]

# --- Edge Image ---
FROM alpine:3.20 AS edge
RUN apk --no-cache add ca-certificates curl
WORKDIR /app

COPY --from=builder /bin/edge /usr/local/bin/edge
COPY edge/config /app/edge/config
RUN mkdir -p /app/edge/cache

EXPOSE 8081 8082

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:$PORT/health || exit 1

ENTRYPOINT ["/usr/local/bin/edge"]
