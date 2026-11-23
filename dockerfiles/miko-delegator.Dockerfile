FROM golang:1.24.7-bookworm as builder
# https://stackoverflow.com/questions/75209245/golang-cross-compiling-with-cgo-inside-docker-image
RUN apt-get update && apt-get install -y curl g++-x86-64-linux-gnu libc6-dev-amd64-cross && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY node node

WORKDIR /app/node

# build info (see pkg/utils/buildinfo); empty or unset falls back to dev/unknown
ARG VERSION
ARG COMMIT
ARG BUILD_TIME

# https://github.com/klaytn/klaytn/issues/197#issuecomment-612597933
RUN CGO_ENABLED=1 CGO_CFLAGS="-O -D__BLST_PORTABLE__" CGO_CFLAGS_ALLOW="-O -D__BLST_PORTABLE__" CC=x86_64-linux-gnu-gcc GOOS=linux GOARCH=amd64 go build -o delegatorbin -ldflags="-w -s -X bisonai.com/miko/node/pkg/utils/buildinfo.Version=${VERSION:-dev} -X bisonai.com/miko/node/pkg/utils/buildinfo.Commit=${COMMIT:-unknown} -X bisonai.com/miko/node/pkg/utils/buildinfo.BuildTime=${BUILD_TIME:-unknown}" ./cmd/delegator
# debian:bookworm-slim
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/node/delegatorbin /usr/bin

CMD ["delegatorbin"]
