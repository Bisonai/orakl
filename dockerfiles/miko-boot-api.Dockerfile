FROM golang:1.24.7-bookworm as builder

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY node node

WORKDIR /app/node

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootbin -ldflags="-w -s" ./cmd/boot_api/main.go

# debian:bookworm-slim
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/node/bootbin /usr/bin

# peers live in the SQLite file at $BOOT_SQLITE_PATH (mount a volume)
CMD ["bootbin"]
