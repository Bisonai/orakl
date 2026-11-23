FROM golang:1.24.7-bookworm as builder

WORKDIR /app

COPY node node

WORKDIR /app/node

# build info (see pkg/utils/buildinfo); empty or unset falls back to dev/unknown
ARG VERSION
ARG COMMIT
ARG BUILD_TIME

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dalstreambin -ldflags="-w -s -X bisonai.com/miko/node/pkg/utils/buildinfo.Version=${VERSION:-dev} -X bisonai.com/miko/node/pkg/utils/buildinfo.Commit=${COMMIT:-unknown} -X bisonai.com/miko/node/pkg/utils/buildinfo.BuildTime=${BUILD_TIME:-unknown}" ./cmd/dal_stream

# debian:bookworm-slim
FROM debian:bookworm-slim

# curl for container health checks (GET /health)
RUN apt-get update && apt-get install -y ca-certificates curl && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/node/dalstreambin /usr/bin

CMD ["dalstreambin"]
