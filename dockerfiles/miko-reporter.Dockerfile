FROM golang:1.23.7-bullseye as builder

RUN apt-get update && apt-get install -y curl g++-x86-64-linux-gnu libc6-dev-amd64-cross && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY node node

WORKDIR /app/node

RUN CGO_ENABLED=1 CGO_CFLAGS="-O -D__BLST_PORTABLE__" CGO_CFLAGS_ALLOW="-O -D__BLST_PORTABLE__" CC=x86_64-linux-gnu-gcc GOOS=linux GOARCH=amd64 go build -o reporterbin -ldflags="-w -s" ./cmd/reporter/main.go

# debian:bullseye-slim
FROM debian@sha256:4b48997afc712259da850373fdbc60315316ee72213a4e77fc5a66032d790b2a

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/node/reporterbin /usr/bin

CMD ["reporterbin"]
