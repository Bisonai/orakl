FROM golang:1.21.5-bullseye as builder

RUN apt-get update && apt-get install -y curl g++-x86-64-linux-gnu libc6-dev-amd64-cross

WORKDIR /app

COPY node node

WORKDIR /app/node

RUN CGO_ENABLED=1 CC=x86_64-linux-gnu-gcc GOOS=linux GOARCH=amd64 go build -o porbin -ldflags="-w -s" ./cmd/por/main.go

# debian:bullseye-slim
FROM debian@sha256:4b48997afc712259da850373fdbc60315316ee72213a4e77fc5a66032d790b2a

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY --from=builder /app/node/porbin /usr/bin

CMD ["porbin"]
