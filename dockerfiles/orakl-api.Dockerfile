
FROM golang:1.24.7-bookworm as builder

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY node node

WORKDIR /app/node

# build info (see pkg/utils/buildinfo); empty or unset falls back to dev/unknown
ARG VERSION
ARG COMMIT
ARG BUILD_TIME

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o apibin -ldflags="-w -s -X bisonai.com/orakl/node/pkg/utils/buildinfo.Version=${VERSION:-dev} -X bisonai.com/orakl/node/pkg/utils/buildinfo.Commit=${COMMIT:-unknown} -X bisonai.com/orakl/node/pkg/utils/buildinfo.BuildTime=${BUILD_TIME:-unknown}" ./cmd/api/main.go

# debian:bookworm-slim
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
   mv ./migrate /usr/bin

WORKDIR /app

RUN mkdir /app/migrations

COPY --from=builder /app/node/migrations/api /app/migrations

COPY --from=builder /app/node/apibin /usr/bin

COPY dockerfiles/start-go.sh .

RUN chmod +x start-go.sh

CMD ["./start-go.sh", "apibin"]
