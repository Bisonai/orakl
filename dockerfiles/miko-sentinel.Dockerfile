FROM golang:1.24.7-bookworm as builder

RUN apt-get update && apt-get install -y curl g++-x86-64-linux-gnu libc6-dev-amd64-cross && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY node node

WORKDIR /app/node

RUN CGO_ENABLED=1 CGO_CFLAGS="-O -D__BLST_PORTABLE__" CGO_CFLAGS_ALLOW="-O -D__BLST_PORTABLE__" CC=x86_64-linux-gnu-gcc GOOS=linux GOARCH=amd64 go build -o sentinelbin -ldflags="-w -s" ./cmd/sentinel/main.go

# debian:bookworm-slim
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y curl jq && rm -rf /var/lib/apt/lists/*

RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/bin

WORKDIR /app

COPY --from=builder /app/node/taskfile.yml /app/taskfile.yml

COPY --from=builder /app/node/taskfiles /app/taskfiles

COPY --from=builder /app/node/sentinelbin /usr/bin

CMD ["sentinelbin"]